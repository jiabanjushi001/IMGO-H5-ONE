// imgo-loadtest uses dedicated user tokens; it never creates or deletes users.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type account struct {
	Token string `json:"token"`
}
type envelope struct {
	Code *int            `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}
type sample struct {
	start    time.Time
	accepted bool
	receipts map[int]time.Duration
}
type stats struct {
	sync.Mutex
	messages                  map[string]*sample
	apiLatency                []float64
	errors                    map[string]int
	attempts, failed, skipped int
}

func post(ctx context.Context, client *http.Client, base, path, token string, body any) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", base+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(token, "Bearer "), "bearer ")))
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("HTTP transport failed")
	}
	defer resp.Body.Close()
	var out envelope
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&out); err != nil {
		return nil, errors.New("invalid API response")
	}
	if out.Code == nil {
		return nil, errors.New("API response missing business code")
	}
	if *out.Code != 0 {
		return nil, fmt.Errorf("business code %d: %.100s", *out.Code, out.Msg)
	}
	return out.Data, nil
}
func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	i := int(float64(len(v)-1) * p)
	return v[i]
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	base := flag.String("base", "", "Dedicated test API origin")
	wsURL := flag.String("ws", "", "WebSocket URL (defaults to base/wss)")
	tokens := flag.String("tokens", "", "JSON array of unique dedicated user tokens")
	users := flag.Int("users", 10, "Concurrent distinct test accounts")
	ramp := flag.Duration("ramp", time.Second, "Pause between new connections")
	duration := flag.Duration("duration", time.Minute, "Steady phase duration after all connections bind")
	rate := flag.Float64("rate", 0, "Total group messages/sec; 0 tests idle online connections")
	group := flag.String("group", "", "Dedicated group ID, e.g. group-5; all accounts must be members")
	origin := flag.String("origin", "null", "WebSocket Origin")
	execute := flag.Bool("execute", false, "Actually connect; otherwise validate settings only")
	flag.Parse()
	u, err := url.Parse(*base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.Trim(u.Path, "/") != "" {
		return errors.New("-base must be an HTTP(S) origin")
	}
	*base = strings.TrimRight(*base, "/")
	if *users < 1 || *users > 10000 || *ramp < 10*time.Millisecond || *duration < time.Second || *duration > time.Hour || *rate < 0 || *rate > 1000 {
		return errors.New("limits: users 1..10000, ramp >=10ms, duration 1s..1h, rate 0..1000")
	}
	if math.IsNaN(*rate) || math.IsInf(*rate, 0) || *rate*duration.Seconds() > 100000 || *rate*duration.Seconds()*float64(*users) > 2000000 {
		return errors.New("limit each run to 100000 messages and 2000000 expected deliveries; shorten duration or reduce rate")
	}
	if *rate > 0 && !strings.HasPrefix(*group, "group-") {
		return errors.New("sending requires -group group-N")
	}
	data, err := os.ReadFile(*tokens)
	if err != nil {
		return errors.New("cannot read -tokens JSON file")
	}
	var accounts []account
	if json.Unmarshal(data, &accounts) != nil || len(accounts) < *users {
		return errors.New("tokens file must contain at least -users accounts")
	}
	accounts = accounts[:*users]
	seen := map[string]bool{}
	for _, a := range accounts {
		if a.Token == "" || seen[a.Token] {
			return errors.New("tokens must be nonempty and distinct")
		}
		seen[a.Token] = true
	}
	if *wsURL == "" {
		w := *u
		w.Path = "/wss"
		w.Scheme = "ws"
		if u.Scheme == "https" {
			w.Scheme = "wss"
		}
		*wsURL = w.String()
	}
	w, err := url.Parse(*wsURL)
	if err != nil || w.Hostname() != u.Hostname() || w.User != nil || (w.Scheme != "ws" && w.Scheme != "wss") || (u.Scheme == "https" && w.Scheme != "wss") {
		return errors.New("WS must use same host as API; HTTPS requires WSS")
	}
	fmt.Printf("Plan: users=%d ramp=%s duration=%s total_messages_per_second=%.2f group=%s\n", *users, *ramp, *duration, *rate, *group)
	if !*execute {
		fmt.Println("Dry run: no requests sent. Add -execute after confirming dedicated test target.")
		return nil
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, timeoutCancel := context.WithTimeout(ctx, time.Duration(*users)*(10*time.Second+*ramp)+*duration+30*time.Second)
	defer timeoutCancel()
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConns: 100, MaxIdleConnsPerHost: 100}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	st := &stats{messages: map[string]*sample{}, errors: map[string]int{}}
	var readers sync.WaitGroup
	var active, drops atomic.Int64
	var setupMu sync.Mutex
	conns := []*websocket.Conn{}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
		readers.Wait()
	}()
	seenUID := map[int64]bool{}
	connect := func(i int, a account) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Use a protected profile response to ensure one simulated person per token.
		raw, err := post(ctx, client, *base, "/enterprise/im/getUserInfo", a.Token, map[string]any{})
		if err != nil {
			return fmt.Errorf("account %d identity preflight: %w", i, err)
		}
		var identity struct {
			UserID int64 `json:"user_id"`
		}
		if json.Unmarshal(raw, &identity) != nil || identity.UserID < 1 {
			return fmt.Errorf("account %d invalid or duplicate user identity", i)
		}
		setupMu.Lock()
		duplicate := seenUID[identity.UserID]
		seenUID[identity.UserID] = true
		setupMu.Unlock()
		if duplicate {
			return fmt.Errorf("account %d duplicate user identity", i)
		}
		if *rate > 0 {
			raw, err = post(ctx, client, *base, "/enterprise/group/groupInfo", a.Token, map[string]any{"group_id": *group})
			var info struct {
				IsJoin int `json:"isJoin"`
			}
			if err != nil || json.Unmarshal(raw, &info) != nil || info.IsJoin < 1 {
				return fmt.Errorf("account %d is not a member of the dedicated group", i)
			}
		}
		dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
		conn, _, err := dialer.DialContext(ctx, *wsURL, http.Header{"Origin": []string{*origin}})
		if err != nil {
			return fmt.Errorf("account %d WS handshake failed", i)
		}
		setupMu.Lock()
		conns = append(conns, conn)
		setupMu.Unlock()
		conn.SetReadLimit(2 << 20)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		var init struct {
			Type     string `json:"type"`
			ClientID string `json:"client_id"`
		}
		if err = conn.ReadJSON(&init); err != nil || init.Type != "init" || init.ClientID == "" {
			return fmt.Errorf("account %d missing WS init", i)
		}
		// HTTP bind provides an explicit auth success response (WS bind has no ack).
		if _, err = post(ctx, client, *base, "/common/pub/bindUid", a.Token, map[string]any{"client_id": init.ClientID}); err != nil {
			return fmt.Errorf("account %d bind: %w", i, err)
		}
		conn.SetReadDeadline(time.Now().Add(75 * time.Second))
		conn.SetPingHandler(func(message string) error {
			conn.SetReadDeadline(time.Now().Add(75 * time.Second))
			return conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(5*time.Second))
		})
		active.Add(1)
		readers.Add(1)
		go func(index int, c *websocket.Conn) {
			defer readers.Done()
			defer active.Add(-1)
			for {
				var event struct {
					Type string          `json:"type"`
					Data json.RawMessage `json:"data"`
				}
				if c.ReadJSON(&event) != nil {
					if ctx.Err() == nil {
						drops.Add(1)
					}
					return
				}
				if event.Type == "group" {
					var message struct {
						ID string `json:"id"`
					}
					if json.Unmarshal(event.Data, &message) != nil {
						continue
					}
					st.Lock()
					if s := st.messages[message.ID]; s != nil {
						if _, exists := s.receipts[index]; !exists {
							s.receipts[index] = time.Since(s.start)
						}
					}
					st.Unlock()
				}
			}
		}(i, conn)
		return nil
	}
	var setups sync.WaitGroup
	slots := make(chan struct{}, 16)
	failures := make(chan error, len(accounts))
setupLoop:
	for i, a := range accounts {
		select {
		case <-ctx.Done():
			break setupLoop
		case slots <- struct{}{}:
		}
		setups.Add(1)
		go func(i int, a account) {
			defer setups.Done()
			defer func() { <-slots }()
			if e := connect(i, a); e != nil {
				failures <- e
				cancel()
			}
		}(i, a)
		select {
		case <-ctx.Done():
			break setupLoop
		case <-time.After(*ramp):
		}
	}
	setups.Wait()
	close(failures)
	for e := range failures {
		return e
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if active.Load() != int64(*users) {
		return errors.New("connections dropped during ramp; steady phase not started")
	}
	fmt.Printf("Bound %d distinct users. Beginning steady phase.\n", active.Load())
	seed := make([]byte, 8)
	if _, err = rand.Read(seed); err != nil {
		return err
	}
	prefix := hex.EncodeToString(seed)
	start := time.Now()
	deadline := time.NewTimer(*duration)
	defer deadline.Stop()
	progress := time.NewTicker(5 * time.Second)
	defer progress.Stop()
	var ticks <-chan time.Time
	if *rate > 0 {
		ticker := time.NewTicker(time.Duration(float64(time.Second) / *rate))
		defer ticker.Stop()
		ticks = ticker.C
	}
	sem := make(chan struct{}, 32)
	var sends sync.WaitGroup
	seq := 0
	stopped := "duration complete"
loop:
	for {
		select {
		case <-ctx.Done():
			stopped = "interrupted"
			break loop
		case <-deadline.C:
			break loop
		case <-progress.C:
			st.Lock()
			fmt.Printf("online=%d disconnected=%d sent=%d failed=%d generator_skipped=%d\n", active.Load(), drops.Load(), st.attempts, st.failed, st.skipped)
			failureStop := st.attempts >= 20 && float64(st.failed)/float64(st.attempts) > 0.05
			st.Unlock()
			if drops.Load() > 0 || failureStop {
				stopped = "stopped: disconnect or API errors >5%"
				break loop
			}
		case <-ticks:
			select {
			case sem <- struct{}{}:
			default:
				st.Lock()
				st.skipped++
				st.Unlock()
				continue
			}
			seq++
			id := fmt.Sprintf("lt-%s-%d", prefix, seq)
			a := accounts[(seq-1)%len(accounts)]
			st.Lock()
			st.messages[id] = &sample{start: time.Now(), receipts: map[int]time.Duration{}}
			st.attempts++
			st.Unlock()
			sends.Add(1)
			go func(id string, a account) {
				defer sends.Done()
				defer func() { <-sem }()
				began := time.Now()
				_, err := post(ctx, client, *base, "/enterprise/im/sendMessage", a.Token, map[string]any{"id": id, "toContactId": *group, "type": "text", "content": "IMGO load test " + id})
				st.Lock()
				defer st.Unlock()
				st.apiLatency = append(st.apiLatency, float64(time.Since(began).Microseconds())/1000)
				if err != nil {
					st.failed++
					st.errors[err.Error()]++
				} else {
					st.messages[id].accepted = true
				}
			}(id, a)
		}
	}
	sends.Wait()
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
	}
	st.Lock()
	defer st.Unlock()
	accepted, received := 0, 0
	delivery := []float64{}
	for _, s := range st.messages {
		if s.accepted {
			accepted++
			received += len(s.receipts)
			for _, d := range s.receipts {
				delivery = append(delivery, float64(d.Microseconds())/1000)
			}
		}
	}
	report := map[string]any{"stop_reason": stopped, "errors": st.errors, "started_at": start.UTC().Format(time.RFC3339), "observed_seconds": time.Since(start).Seconds(), "online_end": active.Load(), "disconnects": drops.Load(), "attempted": st.attempts, "accepted": accepted, "failed": st.failed, "generator_skipped": st.skipped, "expected_deliveries": accepted * len(accounts), "received_deliveries": received, "api_p95_ms": percentile(st.apiLatency, .95), "delivery_p95_ms": percentile(delivery, .95), "delivery_p99_ms": percentile(delivery, .99)}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(encoded))
	cancel()
	return nil
}
