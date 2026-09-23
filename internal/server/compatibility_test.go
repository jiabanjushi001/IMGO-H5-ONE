package server

import (
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func testApp(t *testing.T) (*App, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, e := sqlmock.New()
	if e != nil {
		t.Fatal(e)
	}
	a := &App{db: db, cfg: Config{Prefix: "yu_", JWTKey: strings.Repeat("s", 32), PublicDir: t.TempDir()}, cache: map[string]cacheItem{}, routes: map[string]endpoint{}, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	a.hub = newHub(a)
	a.register()
	t.Cleanup(func() { a.hub.close(); db.Close() })
	return a, mock
}
func TestAllLegacyRoutesRegistered(t *testing.T) {
	a, _ := testApp(t)
	b, e := os.ReadFile("../../docs/legacy-routes.json")
	if e != nil {
		t.Fatal(e)
	}
	var catalog struct {
		Controllers []string `json:"controller_routes"`
		Frontend    []string `json:"frontend_routes"`
	}
	if e = json.Unmarshal(b, &catalog); e != nil {
		t.Fatal(e)
	}
	for _, p := range append(catalog.Controllers, catalog.Frontend...) {
		if _, ok := a.routes[normalizedPath(p)]; !ok {
			t.Errorf("missing legacy route %s", p)
		}
	}
}
func TestNestedFormCompatibility(t *testing.T) {
	v := url.Values{"setting[isVoice]": {"true"}, "user_ids[0]": {"7"}, "user_ids[1]": {"9"}, "messages[0][id]": {"uuid"}, "messages[0][fromUser][id]": {"2"}, "at[]": {"1", "2"}}
	m := decodeForm(v)
	if str(obj(m["setting"])["isVoice"]) != "true" {
		t.Fatal(m)
	}
	if len(ids(m["user_ids"])) != 2 {
		t.Fatal(m)
	}
	list, ok := m["messages"].([]any)
	if !ok || str(obj(list[0])["id"]) != "uuid" {
		t.Fatal(m)
	}
	if len(ids(m["at"])) != 2 {
		t.Fatal(m)
	}
}
func TestPrivateEndpointsRejectAnonymous(t *testing.T) {
	a, _ := testApp(t)
	router := a.Router()
	for _, path := range []string{"/manage/User/index", "/enterprise/im/sendMessage", "/common/pub/bindUid", "/enterprise/Group/setManager", "/common/upload/uploadFile", "/index.php?s=/manage/User/index"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, strings.NewReader(`{"user_id":1}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		var body M
		if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
			t.Fatal(path, w.Body.String())
		}
		if w.Code != 200 || number(body["code"]) != -1 {
			t.Fatalf("%s: %s", path, w.Body.String())
		}
	}
}
func TestOrdinaryUserCannotReadAdminMessages(t *testing.T) {
	a, mock := testApp(t)
	sid := "test-session"
	mock.ExpectQuery("SELECT user_id FROM .*imgo_session").WithArgs(sid, int64(7), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT \\* FROM .*user.* WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "role", "status"}).AddRow(7, 0, 1))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/manage/Message/index", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer "+signToken(a.cfg.JWTKey, 7, sid, time.Now().Add(time.Hour)))
	a.Router().ServeHTTP(w, req)
	var b M
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if number(b["code"]) != 403 {
		t.Fatal(w.Body.String())
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}
func TestNonMemberCannotSendGroupMessage(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT gu.*").WithArgs(int64(9), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	r := &request{app: a, c: &gin.Context{}, p: M{}, user: M{"user_id": int64(7)}}
	r.c.Request = httptest.NewRequest("POST", "/enterprise/im/sendMessage", nil)
	_, e := a.sendMessage(r, M{"toContactId": "group-9", "content": "x", "fromUser": M{"id": 1}})
	if e == nil {
		t.Fatal("non-member allowed to send")
	}
	if e = mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}
func TestWebsocketCannotRebindOtherUser(t *testing.T) {
	a, _ := testApp(t)
	a.hub.peers["client"] = &peer{id: "client", uid: 7}
	if e := a.hub.bind("client", 8, claims{UID: 8}); e == nil {
		t.Fatal("client hijacked")
	}
	delete(a.hub.peers, "client")
}
func TestInstallerClosed(t *testing.T) {
	a, _ := testApp(t)
	w := httptest.NewRecorder()
	a.Router().ServeHTTP(w, httptest.NewRequest("GET", "/index/install/install", nil))
	var b M
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if number(b["code"]) != 410 {
		t.Fatal(w.Body.String())
	}
}
func TestGroupTokenTampering(t *testing.T) {
	a, _ := testApp(t)
	payload := "7." + str(time.Now().Add(time.Hour).Unix())
	token := payload + "." + a.signLink(payload)
	if !a.validGroupToken(token) || a.validGroupToken("8"+token[1:]) {
		t.Fatal("group token signature verification")
	}
}
