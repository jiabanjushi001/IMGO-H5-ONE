package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPasswordlessAPILoginIsDisabled(t *testing.T) {
	a, _ := testApp(t)

	if _, exists := a.routes["/common/api/login"]; exists {
		t.Fatal("passwordless API login route must not be registered")
	}
	if route, exists := a.routes["/common/pub/login"]; !exists || !route.public {
		t.Fatal("normal administrator and user login route must remain public")
	}
	if route, exists := a.routes["/common/api/createuser"]; !exists || !route.public {
		t.Fatal("external user creation API must remain available")
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/common/api/login", nil)
	a.Router().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled API login returned HTTP %d, want %d", response.Code, http.StatusNotFound)
	}
}
