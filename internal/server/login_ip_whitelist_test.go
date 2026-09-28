package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestNormalizeLoginIPRules(t *testing.T) {
	rules, err := normalizeLoginIPRules("203.0.113.8\n10.0.0.5/24, 2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(rules, ",")
	for _, expected := range []string{"203.0.113.8", "10.0.0.0/24", "2001:db8::1"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("rules %v missing %s", rules, expected)
		}
	}
	if !loginIPMatches("10.0.0.99", rules) || loginIPMatches("198.51.100.2", rules) {
		t.Fatalf("unexpected matching result for %v", rules)
	}
	if _, err = normalizeLoginIPRules("not-an-ip"); err == nil {
		t.Fatal("invalid rule must be rejected")
	}
}

func TestEnableLoginIPWhitelistRequiresCurrentIP(t *testing.T) {
	a, mock := testApp(t)
	_, err := a.setLoginIPWhitelist(context.Background(), 1, "session", "198.51.100.8", true, []string{"203.0.113.0/24"})
	if err == nil || !strings.Contains(err.Error(), "当前 IP") {
		t.Fatalf("expected current-IP protection, got %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnableLoginIPWhitelistRevokesOnlyBlockedAdminSessions(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_imgo_session` SET login_ip=?,admin_login=1 WHERE sid=? AND user_id=?")).
		WithArgs("203.0.113.8", "current", int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT sid,login_ip FROM `yu_imgo_session` WHERE admin_login=1")).
		WillReturnRows(sqlmock.NewRows([]string{"sid", "login_ip"}).
			AddRow("allowed", "203.0.113.9").AddRow("blocked", "198.51.100.9"))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `yu_imgo_session` WHERE sid=?")).
		WithArgs("blocked").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_security_setting`").
		WithArgs(true, `["203.0.113.0/24"]`, int64(1), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	revoked, err := a.setLoginIPWhitelist(context.Background(), 1, "current", "203.0.113.8", true, []string{"203.0.113.0/24"})
	if err != nil || len(revoked) != 1 || revoked[0] != "blocked" {
		t.Fatalf("revoked=%v err=%v", revoked, err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginIPWhitelistRejectsBackendLogin(t *testing.T) {
	a, mock := testApp(t)
	a.storeLoginIPWhitelist(true, []string{"203.0.113.0/24"})
	password, err := hashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE account=\\? AND delete_time=0").
		WithArgs("wuhu2").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "password", "salt", "role", "status", "delete_time"}).
			AddRow(2, "wuhu2", "wuhu2", password, "", 0, 1, 0))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/pub/login", nil)
	r := &request{app: a, c: c, p: M{"account": "wuhu2", "password": "test-password", "admin_login": 1}}
	if _, err = a.login(r); err == nil || !strings.Contains(err.Error(), "不在后台登录白名单") {
		t.Fatalf("expected backend whitelist rejection, got %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoginIPWhitelistAppliesOnlyToManageAPI(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		want int64
	}{
		{"backend", "/manage/test/ping", 403},
		{"h5", "/enterprise/test/ping", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			a.storeLoginIPWhitelist(true, []string{"203.0.113.0/24"})
			a.routes[normalizedPath(tc.path)] = endpoint{handler: func(*request) (any, error) { return M{"ok": true}, nil }}
			sid := "whitelist-" + tc.name
			token := signToken(a.cfg.JWTKey, 7, sid, time.Now().Add(time.Hour))
			mock.ExpectQuery("SELECT user_id FROM .*imgo_session").
				WithArgs(sid, int64(7), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
			mock.ExpectQuery("SELECT \\* FROM .*user.* WHERE user_id").
				WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "role", "admin_role_id", "status", "delete_time"}).AddRow(7, 0, 0, 1, 0))

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", tc.path, nil)
			req.Header.Set("Authorization", "bearer "+token)
			a.Router().ServeHTTP(w, req)
			body := M{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if number(body["code"]) != tc.want {
				t.Fatalf("response = %s", w.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
