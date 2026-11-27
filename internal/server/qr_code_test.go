package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPersonalQRCodeRouteIsAuthenticatedAndUsesConfiguredBase(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.QRBaseURL = "http://192.168.22.102:8088/"
	route, ok := a.routes[normalizedPath("/enterprise/qr/my")]
	if !ok || route.public || route.super || route.permission != "" {
		t.Fatalf("personal QR route is not authenticated correctly: %#v", route)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/enterprise/qr/my", nil)
	data, err := a.qrCode(&request{app: a, c: c, user: M{"user_id": int64(7)}})
	if err != nil {
		t.Fatal(err)
	}
	qrURL := str(obj(data)["qrUrl"])
	if !strings.HasPrefix(qrURL, "http://192.168.22.102:8088/scan/u/") {
		t.Fatalf("unexpected personal QR URL: %q", qrURL)
	}
}
