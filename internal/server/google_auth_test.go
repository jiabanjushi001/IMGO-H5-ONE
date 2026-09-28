package server

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestEnableGoogleAuthRejectsUnboundSuperAdmin(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_user_totp` WHERE user_id=? FOR UPDATE")).
		WithArgs(int64(1)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err := a.setGoogleAuthEnabled(context.Background(), 1, true)
	if err == nil || !strings.Contains(err.Error(), "超级管理员尚未绑定") {
		t.Fatalf("expected binding requirement, got %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnableGoogleAuthRevokesEveryUnboundSession(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_user_totp` WHERE user_id=? FOR UPDATE")).
		WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT s.user_id FROM `yu_imgo_session` s LEFT JOIN `yu_imgo_user_totp` t ON t.user_id=s.user_id WHERE t.user_id IS NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7).AddRow(9))
	mock.ExpectExec(regexp.QuoteMeta("DELETE s FROM `yu_imgo_session` s LEFT JOIN `yu_imgo_user_totp` t ON t.user_id=s.user_id WHERE t.user_id IS NULL")).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec("INSERT INTO `yu_imgo_security_setting`").
		WithArgs(true, int64(1), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	disconnected, err := a.setGoogleAuthEnabled(context.Background(), 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(disconnected, []int64{7, 9}) {
		t.Fatalf("disconnected = %v", disconnected)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleAuthenticatorRFCVector(t *testing.T) {
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, ok := googleCodeAt(secret, 1)
	if !ok || code != "287082" {
		t.Fatalf("code = %q, ok = %v", code, ok)
	}
	if step, ok := matchingGoogleStep(secret, code, time.Unix(59, 0)); !ok || step != 1 {
		t.Fatalf("matching step = %d, ok = %v", step, ok)
	}
}

func TestGoogleSecretEncryptionRoundTrip(t *testing.T) {
	a, _ := testApp(t)
	const secret = "JBSWY3DPEHPK3PXP"
	ciphertext, err := a.encryptGoogleSecret(secret)
	if err != nil || ciphertext == "" || strings.Contains(ciphertext, secret) {
		t.Fatalf("ciphertext = %q, err = %v", ciphertext, err)
	}
	plain, err := a.decryptGoogleSecret(ciphertext)
	if err != nil || plain != secret {
		t.Fatalf("plain = %q, err = %v", plain, err)
	}
}

func TestGoogleAuthRoutesRequireSuperAdmin(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{
		"/manage/user/googleauthdetail",
		"/manage/user/googleauthbind",
		"/manage/user/googleauthunbind",
		"/manage/config/getsecurity",
		"/manage/config/setsecurity",
	} {
		route, ok := a.routes[path]
		if !ok || !route.super || route.public {
			t.Fatalf("route %s must be private and super-admin only: %#v", path, route)
		}
	}
}

func TestBackendLoginRequiresBoundGoogleCodeWhenEnabled(t *testing.T) {
	a, mock := testApp(t)
	a.googleAuthOn.Store(true)
	password, err := hashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	secret := "JBSWY3DPEHPK3PXP"
	ciphertext, err := a.encryptGoogleSecret(secret)
	if err != nil {
		t.Fatal(err)
	}
	step := time.Now().Unix() / googleAuthPeriod
	code, ok := googleCodeAt(secret, step)
	if !ok {
		t.Fatal("could not generate code")
	}
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE account=\\? AND delete_time=0").
		WithArgs("wuhu2").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "password", "salt", "role", "status", "delete_time"}).
			AddRow(2, "wuhu2", "wuhu2", password, "", 0, 1, 0))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT secret_cipher,last_used_step FROM `yu_imgo_user_totp` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"secret_cipher", "last_used_step"}).AddRow(ciphertext, 0))
	mock.ExpectExec("UPDATE `yu_imgo_user_totp` SET `last_used_step`=\\? WHERE user_id=\\?").
		WithArgs(step, int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("UPDATE `yu_user` SET login_count=").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_session`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), int64(2)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/pub/login", nil)
	r := &request{app: a, c: c, p: M{"account": "wuhu2", "password": "test-password", "admin_login": 1, "google_code": code}}
	result, err := a.login(r)
	if err != nil || str(obj(result)["authToken"]) == "" {
		t.Fatalf("login result = %#v, err = %v", result, err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBackendLoginRejectsUnboundAccountWhenEnabled(t *testing.T) {
	a, mock := testApp(t)
	a.googleAuthOn.Store(true)
	password, err := hashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE account=\\? AND delete_time=0").
		WithArgs("wuhu2").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "password", "salt", "role", "status", "delete_time"}).
			AddRow(2, "wuhu2", "wuhu2", password, "", 0, 1, 0))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT secret_cipher,last_used_step FROM `yu_imgo_user_totp` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(2)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/pub/login", nil)
	r := &request{app: a, c: c, p: M{"account": "wuhu2", "password": "test-password", "admin_login": 1, "google_code": "123456"}}
	if _, err = a.login(r); err == nil || !strings.Contains(err.Error(), "未绑定") {
		t.Fatalf("expected unbound error, got %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
