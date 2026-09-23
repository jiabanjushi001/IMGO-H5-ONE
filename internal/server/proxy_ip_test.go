package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestRouterTrustsLocalReverseProxyForClientIP(t *testing.T) {
	a := &App{cfg: Config{}, hub: &Hub{}}
	router := a.Router()
	router.GET("/__client_ip", func(c *gin.Context) {
		c.String(http.StatusOK, a.clientIP(c))
	})

	tests := []struct {
		name         string
		forwardedFor string
		cloudflareIP string
		want         string
	}{
		{"nginx forwards visitor", "203.0.113.25", "", "203.0.113.25"},
		{"cloudflare visitor header", "172.64.10.20", "198.51.100.42", "198.51.100.42"},
		{"untrusted visitor cannot spoof cloudflare header", "203.0.113.25", "198.51.100.42", "203.0.113.25"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/__client_ip", nil)
			req.RemoteAddr = "127.0.0.1:54321"
			req.Header.Set("X-Forwarded-For", tc.forwardedFor)
			if tc.cloudflareIP != "" {
				req.Header.Set("CF-Connecting-IP", tc.cloudflareIP)
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if got := res.Body.String(); got != tc.want {
				t.Fatalf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRecordChatIPUsesDedicatedFields(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectExec("UPDATE `yu_user` SET last_chat_time=\\?,last_chat_ip=\\? WHERE user_id=\\?").
		WithArgs(sqlmock.AnyArg(), "198.51.100.42", int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := a.recordChatIP(context.Background(), 7, "198.51.100.42"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
