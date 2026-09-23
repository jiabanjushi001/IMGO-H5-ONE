package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMediaCookieSecureFlagMatchesRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, target, host, forwarded string
		wantSecure                    bool
	}{
		{name: "local HTTP ignores production base URL", target: "http://127.0.0.1:8088/api", host: "127.0.0.1:8088"},
		{name: "matching production host stays secure", target: "http://imgo.myad.top/api", host: "imgo.myad.top", wantSecure: true},
		{name: "reverse proxy HTTPS stays secure", target: "http://127.0.0.1:8088/api", host: "imgo.myad.top", forwarded: "https", wantSecure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("GET", tc.target, nil)
			ctx.Request.Host = tc.host
			ctx.Request.Header.Set("X-Forwarded-Proto", tc.forwarded)
			a := &App{cfg: Config{BaseURL: "https://imgo.myad.top"}}
			a.setMediaCookie(ctx, "Bearer token-value", 3600)
			cookie := recorder.Header().Get("Set-Cookie")
			if strings.Contains(cookie, "Secure") != tc.wantSecure {
				t.Fatalf("Set-Cookie=%q, want secure=%v", cookie, tc.wantSecure)
			}
			if !strings.Contains(cookie, "imgo_media=token-value") || !strings.Contains(cookie, "HttpOnly") {
				t.Fatalf("unexpected cookie: %q", cookie)
			}
		})
	}
}
