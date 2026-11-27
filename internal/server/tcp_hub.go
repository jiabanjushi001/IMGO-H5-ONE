package server

import (
	"encoding/json"
	"net"
	"sync"
	"time"
)

type tcpEvent struct {
	Version int    `json:"version"`
	Event   string `json:"event"`
	Time    int64  `json:"time"`
	Data    any    `json:"data,omitempty"`
}

type tcpPeer struct {
	id     string
	uid    int64
	claims claims
	conn   net.Conn
	send   chan []byte
	done   chan struct{}
	once   sync.Once
}

func (p *tcpPeer) stop() {
	p.once.Do(func() {
		close(p.done)
		_ = p.conn.Close()
	})
}

type tcpHub struct {
	a      *App
	mu     sync.RWMutex
	peers  map[string]*tcpPeer
	closed bool
}

func newTCPHub(a *App) *tcpHub {
	return &tcpHub{a: a, peers: map[string]*tcpPeer{}}
}

func (h *tcpHub) add(conn net.Conn) *tcpPeer {
	p := &tcpPeer{
		id:   randomID(),
		conn: conn,
		send: make(chan []byte, 128),
		done: make(chan struct{}),
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		p.stop()
		return p
	}
	h.peers[p.id] = p
	h.mu.Unlock()
	return p
}

func (h *tcpHub) remove(p *tcpPeer) int64 {
	if p == nil {
		return 0
	}
	h.mu.Lock()
	uid := p.uid
	delete(h.peers, p.id)
	h.mu.Unlock()
	p.stop()
	return uid
}

func (h *tcpHub) bind(p *tcpPeer, uid int64, cl claims) error {
	if p == nil {
		return nil
	}
	h.mu.Lock()
	current, ok := h.peers[p.id]
	if !ok || current != p {
		h.mu.Unlock()
		return clientError{"TCP 连接已断开", 400}
	}
	if p.uid != 0 && p.uid != uid {
		h.mu.Unlock()
		return deny()
	}
	changed := p.uid == 0
	p.uid = uid
	p.claims = cl
	h.mu.Unlock()
	if changed && h.a != nil && h.a.hub != nil {
		h.a.hub.broadcast("isOnline", M{"id": uid, "is_online": 1})
	}
	return nil
}

func (h *tcpHub) enqueue(p *tcpPeer, value any) bool {
	payload, err := json.Marshal(value)
	if err != nil {
		return false
	}
	select {
	case <-p.done:
		return false
	case p.send <- payload:
		return true
	default:
		p.stop()
		return false
	}
}

func (h *tcpHub) send(uids []int64, kind string, data any) {
	wanted := map[int64]bool{}
	for _, uid := range uids {
		wanted[uid] = true
	}
	now := time.Now()
	event := tcpEvent{Version: tcpProtocolVersion, Event: kind, Time: now.Unix(), Data: data}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid > 0 && wanted[p.uid] && p.claims.Exp > now.Unix() {
			h.enqueue(p, event)
		}
	}
}

func (h *tcpHub) broadcast(kind string, data any) {
	now := time.Now()
	event := tcpEvent{Version: tcpProtocolVersion, Event: kind, Time: now.Unix(), Data: data}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid > 0 && p.claims.Exp > now.Unix() {
			h.enqueue(p, event)
		}
	}
}

func (h *tcpHub) online(uid int64, now time.Time) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid != uid || p.claims.Exp <= now.Unix() {
			continue
		}
		select {
		case <-p.done:
			continue
		default:
			return 1
		}
	}
	return 0
}

func (h *tcpHub) collectOnline(users map[int64]bool, allowed map[int64]bool, all bool, now time.Time) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
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
	return devices
}

func (h *tcpHub) disconnectUser(uid int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.uid == uid {
			p.stop()
		}
	}
}

func (h *tcpHub) disconnectSession(sid string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.peers {
		if p.claims.SID == sid {
			p.stop()
		}
	}
}

func (h *tcpHub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, p := range h.peers {
		p.stop()
	}
}
