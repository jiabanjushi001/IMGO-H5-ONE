package server

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func socketBindingRequest(t *testing.T, app *App, path, clientID string) *request {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", path, nil)
	return &request{
		app:    app,
		c:      c,
		p:      M{"client_id": clientID, "group_id": "group-17"},
		user:   M{"user_id": int64(2)},
		claims: claims{UID: 2},
	}
}

func TestBindGroupIgnoresSocketReconnectRace(t *testing.T) {
	a, _ := testApp(t)
	r := socketBindingRequest(t, a, "/common/pub/bindGroup", "expired-client")
	if _, err := a.public(r); err != nil {
		t.Fatalf("bindGroup should be best effort after group creation, got %v", err)
	}
}

func TestBindUIDStillReportsDisconnectedSocket(t *testing.T) {
	a, _ := testApp(t)
	r := socketBindingRequest(t, a, "/common/pub/bindUid", "expired-client")
	if _, err := a.public(r); !errors.Is(err, errSocketDisconnected) {
		t.Fatalf("bindUid error = %v, want errSocketDisconnected", err)
	}
}
