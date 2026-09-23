package server

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSocketAnyOriginStillRequiresToken(t *testing.T) {
	for _, origin := range []string{"", "null", "file://", "https://untrusted.example", "custom://app", "arbitrary-value"} {
		t.Run(origin, func(t *testing.T) {
			a, _ := testApp(t)
			server := httptest.NewServer(a.Router())
			defer server.Close()
			headers := http.Header{}
			if origin != "" {
				headers.Set("Origin", origin)
			}
			conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/wss", headers)
			if err != nil {
				t.Fatalf("origin handshake: %v", err)
			}
			defer conn.Close()
			if response.StatusCode != http.StatusSwitchingProtocols {
				t.Fatalf("status %d", response.StatusCode)
			}
			conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			var msg M
			if err := conn.ReadJSON(&msg); err != nil || str(msg["type"]) != "init" {
				t.Fatalf("init: %v %v", msg, err)
			}
			if err := conn.WriteJSON(M{"type": "bindUid", "user_id": 1, "token": "invalid-token"}); err != nil {
				t.Fatal(err)
			}
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatal(err)
			}
			if str(msg["type"]) != "error" || number(msg["code"]) != 401 {
				t.Fatalf("invalid token was not rejected: %v", msg)
			}

		})
	}
}
