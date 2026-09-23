package server

import (
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPHPLoginExpiryResponse(t *testing.T) {
	for _, kind := range []string{"missing", "malformed", "expired", "revoked", "disabled"} {
		t.Run(kind, func(t *testing.T) {
			a, mock := testApp(t)
			token := ""
			switch kind {
			case "malformed":
				token = "invalid-token"
			case "expired":
				token = signToken(a.cfg.JWTKey, 7, "session", time.Now().Add(-time.Minute))
			case "revoked", "disabled":
				token = signToken(a.cfg.JWTKey, 7, "session", time.Now().Add(time.Hour))
				sessionRows := sqlmock.NewRows([]string{"user_id"})
				if kind == "disabled" {
					sessionRows.AddRow(7)
				}
				mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WithArgs("session", int64(7), sqlmock.AnyArg()).WillReturnRows(sessionRows)
				if kind == "disabled" {
					mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
				}
			}
			req := httptest.NewRequest("POST", "/enterprise/im/getContacts", nil)
			req.Header.Set("Authorization", token)
			w := httptest.NewRecorder()
			a.Router().ServeHTTP(w, req)
			var body M
			if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
				t.Fatal(e)
			}
			if w.Code != 200 || number(body["code"]) != -1 || str(body["msg"]) == "" {
				t.Fatalf("PHP login expiry contract: HTTP %d %s", w.Code, w.Body.String())
			}
			if data, ok := body["data"].([]any); !ok || len(data) != 0 {
				t.Fatal("expected empty data array")
			}
			if e := mock.ExpectationsWereMet(); e != nil {
				t.Fatal(e)
			}
		})
	}
}
