package main

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPServerAllowsLargeVideoUpload(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	srv := newHTTPServer("127.0.0.1:0", handler)
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout < 30*time.Minute || srv.WriteTimeout < 30*time.Minute {
		t.Fatalf("request timeouts are too short for large video upload: read=%s write=%s", srv.ReadTimeout, srv.WriteTimeout)
	}
	if srv.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes = %d", srv.MaxHeaderBytes)
	}
}
