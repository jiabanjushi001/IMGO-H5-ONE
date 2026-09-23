package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostHonorsAuthenticationAndBusinessErrors(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{{`{"code":0,"data":{"client_id":"test"}}`, true}, {`{"code":401,"msg":"expired"}`, false}, {`<html>error</html>`, false}, {`{}`, false}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("missing auth")
			}
			if r.Method != "POST" {
				t.Error("wrong method")
			}
			w.Write([]byte(tc.body))
		}))
		_, err := post(context.Background(), server.Client(), server.URL, "/test", "test-token", map[string]any{})
		server.Close()
		if (err == nil) != tc.ok {
			t.Fatalf("ok=%v err=%v", tc.ok, err)
		}
	}
}
func TestPercentile(t *testing.T) {
	if percentile(nil, .95) != 0 {
		t.Fatal("empty")
	}
	if percentile([]float64{50, 10, 40, 20, 30}, .5) != 30 {
		t.Fatal("median")
	}
}
