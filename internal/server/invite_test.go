package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInviteURL(t *testing.T) {
	for _, tc := range []struct{ name, prefix, want string }{
		{"h5", "http://127.0.0.1:8765/h5/#/pages/login/register?inviteCode=", "http://127.0.0.1:8765/h5/#/pages/login/register?inviteCode=token123"},
		{"default", "", "http://127.0.0.1:8088/index.html/#/register?inviteCode=token123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{cfg: Config{BaseURL: "http://127.0.0.1:8088", InviteURL: tc.prefix}}
			if got := a.inviteURL("token123"); got != tc.want {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
			if got := a.url("/avatar/test"); got != "http://127.0.0.1:8088/avatar/test" {
				t.Fatalf("resource URL changed: %s", got)
			}
		})
	}
}

func TestGroupInviteURL(t *testing.T) {
	a := &App{cfg: Config{BaseURL: "http://127.0.0.1:8088", H5URL: "http://127.0.0.1:8765/"}}
	got := a.groupInviteURL("group-8", "8.123.signature")
	want := "http://127.0.0.1:8765/#/pages/message/group/info?group_id=group-8&token=8.123.signature"
	if got != want {
		t.Fatalf("got %q; want %q", got, want)
	}
}

func TestScanURL(t *testing.T) {
	for _, tc := range []struct{ name, kind, qrBase, want string }{
		{"group default", "g", "", "http://127.0.0.1:8088/scan/g/signed-token"},
		{"group LAN", "g", "http://192.168.1.123:8765/", "http://192.168.1.123:8765/scan/g/signed-token"},
		{"user LAN", "u", "http://192.168.1.123:8765/", "http://192.168.1.123:8765/scan/u/signed-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{cfg: Config{BaseURL: "http://127.0.0.1:8088", QRBaseURL: tc.qrBase}}
			if got := a.scanURL(tc.kind, "signed-token"); got != tc.want {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestGroupScanRedirect(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.H5URL = "http://127.0.0.1:8765/"
	expiry := time.Now().Add(time.Hour).Unix()
	payload := fmt.Sprintf("8.%d", expiry)
	token := payload + "." + a.signLink(payload)
	for _, tc := range []struct {
		name, token string
		status      int
	}{
		{"valid", token, http.StatusFound},
		{"tampered", token + "x", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			a.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/scan/g/"+tc.token, nil))
			if w.Code != tc.status {
				t.Fatalf("status=%d, want=%d: %s", w.Code, tc.status, w.Body.String())
			}
			if tc.status == http.StatusFound {
				location := w.Header().Get("Location")
				if !strings.HasPrefix(location, "http://127.0.0.1:8765/#/pages/message/group/info?group_id=group-8&token=") || !strings.HasSuffix(location, token) {
					t.Fatalf("unexpected redirect: %s", location)
				}
			}
		})
	}
}
