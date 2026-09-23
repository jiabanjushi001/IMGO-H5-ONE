package server

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUniappEmptyJSONBody(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{{"", true}, {"  ", true}, {"{}", true}, {"null", true}, {"{", false}, {"undefined", false}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/common/pub/getSystemInfo?test=ok", strings.NewReader(tc.body))
		c.Request.Header.Set("Content-Type", "application/json")
		p, e := parseParams(c)
		if (e == nil) != tc.valid {
			t.Errorf("body %q: %v", tc.body, e)
		}
		if tc.valid && str(p["test"]) != "ok" {
			t.Fatal("query parameters lost")
		}
	}
}

func TestTokenOnlyWildcardCORS(t *testing.T) {
	a, _ := testApp(t)
	router := a.Router()
	for _, origin := range []string{"https://imh5.myad.top", "https://new.example", "http://localhost:9000"} {
		for _, method := range []string{"OPTIONS", "POST"} {
			req := httptest.NewRequest(method, "/enterprise/im/getContacts", nil)
			req.Header.Set("Origin", origin)
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", "authorization,content-type,token,session")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Fatal("wildcard origin missing")
			}
			if w.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatal("arbitrary credentialed origins must not be enabled")
			}
			headers := strings.ToLower(w.Header().Get("Access-Control-Allow-Headers"))
			for _, key := range []string{"authorization", "content-type", "token", "session", "x-csrf-token"} {
				found := false
				for _, entry := range strings.Split(headers, ",") {
					if strings.TrimSpace(entry) == key {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing header %s", key)
				}
			}
			if method == "OPTIONS" && w.Code != 204 {
				t.Fatal("preflight failed")
			}
			if method == "POST" && !strings.Contains(w.Body.String(), "请重新登录") {
				t.Fatal("authentication bypassed")
			}
		}
	}
}
