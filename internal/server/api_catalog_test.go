package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestAPICatalogListsEveryRegisteredRoute(t *testing.T) {
	a, _ := testApp(t)
	document, err := os.ReadFile("../../API.txt")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(document))
	for path := range a.routes {
		if !strings.Contains(text, "| "+path+" |") {
			t.Errorf("API.txt missing %s", path)
		}
	}
}

func TestEveryPrivateAPIRejectsMissingAuthorizationWithoutRunningHandler(t *testing.T) {
	a, _ := testApp(t)
	paths := make([]string, 0, len(a.routes))
	for path, route := range a.routes {
		if !route.public {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	router := a.Router()
	for _, path := range paths {
		t.Run(strings.Trim(path, "/"), func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest("POST", path, nil)
			router.ServeHTTP(response, request)
			body := M{}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON response: %s", response.Body.String())
			}
			if number(body["code"]) == 404 || !strings.Contains(str(body["msg"]), "登录") {
				t.Fatalf("route was not protected as expected: %s", response.Body.String())
			}
		})
	}
}
