package server

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

type peer struct {
	id     string
	uid    int64
	claims claims
	conn   *websocket.Conn
	send   chan []byte
	done   chan struct{}
	once   sync.Once
}

func (p *peer) stop() { p.once.Do(func() { close(p.done); _ = p.conn.Close() }) }

type Hub struct {
	a      *App
	mu     sync.RWMutex
	peers  map[string]*peer
	closed bool
}

var errSocketDisconnected = clientError{"WebSocket 连接已断开", 400}

func newHub(a *App) *Hub { return &Hub{a: a, peers: map[string]*peer{}} }

// Online means authenticated, live WebSocket connections. Multiple
// connections from one user count as devices, but only one online user.
func (h *Hub) onlineCounts(now time.Time) (int, int) {
	return h.onlineCountsMatching(nil, true, now)
}

func (h *Hub) onlineCountsForUsers(allowed map[int64]bool, now time.Time) (int, int) {
	return h.onlineCountsMatching(allowed, false, now)
}

func (h *Hub) onlineCountsMatching(allowed map[int64]bool, all bool, now time.Time) (int, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := map[int64]bool{}
	devices := 0
	for _, p := range h.peers {
		if p.uid == 0 || p.claims.Exp <= now.Unix() || !all && !allowed[p.uid] {
			continue
		}
		select {
		case <-p.done:
			continue
		default:
		}
		users[p.uid] = true
		devices++
	}
	return len(users), devices
}

func (h *Hub) serve(c *gin.Context) {
	connectionIP := h.a.clientIP(c)
	// Clients authenticate with an explicit token; Origin is not an access restriction.
	up := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool {
		return true
	}, HandshakeTimeout: 5 * time.Second}
	conn, e := up.Upgrade(c.Writer, c.Request, nil)
	if e != nil {
		return
	}
	p := &peer{id: randomID(), conn: conn, send: make(chan []byte, 128), done: make(chan struct{})}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		p.stop()
		return
	}
	h.peers[p.id] = p
	h.mu.Unlock()
	defer func() {
		p.stop()
		h.mu.Lock()
		uid := p.uid
		delete(h.peers, p.id)
		h.mu.Unlock()
		if uid > 0 && h.online(uid) == 0 {
			h.broadcast("isOnline", M{"id": uid, "is_online": 0})
		}
	}()
	go h.writer(p)
	h.enqueue(p, M{"type": "init", "client_id": p.id})
	conn.SetReadLimit(64 << 10)
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(90 * time.Second)) })
	for {
		var m M
		if e = conn.ReadJSON(&m); e != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		switch str(m["type"]) {
		case "ping":
			h.enqueue(p, M{"type": "pong", "multiport": true})
		case "bindUid":
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			u, cl, err := h.a.authenticate(ctx, str(m["token"]))
			if err != nil {
				cancel()
				h.enqueue(p, M{"type": "error", "code": 401})
				continue
			}
			uid := number(u["user_id"])
			if h.bind(p.id, uid, cl) != nil {
				cancel()
				p.stop()
				return
			}
			connectedAt, err := h.a.recordChatIP(ctx, uid, connectionIP)
			cancel()
			if err == nil && connectionIP != "" {
				h.broadcast("userIPChanged", M{"id": uid, "time": connectedAt})
			}
		}
	}
}
func (h *Hub) writer(p *peer) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer p.stop()
	for {
		select {
		case <-p.done:
			return
		case b := <-p.send:
			_ = p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if p.conn.WriteMessage(websocket.TextMessage, b) != nil {
				return
			}
		case <-ticker.C:
			h.mu.RLock()
			cl := p.claims
			h.mu.RUnlock()
			if cl.UID != 0 {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, e := one(ctx, h.a.db, "SELECT sid FROM "+h.a.t("imgo_session")+" WHERE sid=? AND expires_at>?", cl.SID, time.Now().Unix())
				cancel()
				if e != nil {
					return
				}
			}
			if p.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)) != nil {
				return
			}
		}
	}
}
func (h *Hub) enqueue(p *peer, v any) {
	b, e := json.Marshal(v)
	if e != nil {
		return
	}
	select {
	case <-p.done:
		return
	case p.send <- b:
	default:
		p.stop()
	}
}
func (h *Hub) bind(id string, uid int64, cl claims) error {
	if id == "" {
		return clientError{"缺少 client_id", 400}
	}
	h.mu.Lock()
	p, ok := h.peers[id]
	if !ok {
		h.mu.Unlock()
		return errSocketDisconnected
	}
	if p.uid != 0 && p.uid != uid {
		h.mu.Unlock()
		return deny()
	}
	changed := p.uid == 0
	p.uid = uid
	p.claims = cl
	h.mu.Unlock()
	if changed {
		h.broadcast("isOnline", M{"id": uid, "is_online": 1})
	}
	return nil
}
func (h *Hub) send(uids []int64, kind string, data any) {
	wanted := map[int64]bool{}
	for _, uid := range uids {
		wanted[uid] = true
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid > 0 && wanted[p.uid] && p.claims.Exp > time.Now().Unix() {
			h.enqueue(p, M{"type": kind, "time": time.Now().Unix(), "data": data})
		}
	}
}
func (h *Hub) broadcast(kind string, data any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid > 0 && p.claims.Exp > time.Now().Unix() {
			h.enqueue(p, M{"type": kind, "time": time.Now().Unix(), "data": data})
		}
	}
}
func (h *Hub) online(uid int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid == uid && p.claims.Exp > time.Now().Unix() {
			select {
			case <-p.done:
				continue
			default:
				return 1
			}
		}
	}
	return 0
}
func (h *Hub) disconnectClient(id string, uid int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if p := h.peers[id]; p != nil && p.uid == uid {
		p.stop()
	}
}
func (h *Hub) disconnectUser(uid int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid == uid {
			p.stop()
		}
	}
}
func (h *Hub) disconnectSession(sid string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.claims.SID == sid {
			p.stop()
		}
	}
}
func (h *Hub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, p := range h.peers {
		p.stop()
	}
}
func (a *App) groupEvent(ctx context.Context, gid int64, kind string, data any) {
	members, e := rows(ctx, a.db, "SELECT user_id FROM "+a.t("group_user")+" WHERE group_id=? AND status=1", gid)
	if e != nil {
		a.log.Error("group event lookup", "error", e)
		return
	}
	uids := []int64{}
	for _, u := range members {
		uids = append(uids, number(u["user_id"]))
	}
	a.hub.send(uids, kind, data)
}
