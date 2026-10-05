package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func accountVerificationRequest(a *App, authenticated bool, params M) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/enterprise/im/editAccount", nil)
	isAuth := int64(0)
	if authenticated {
		isAuth = 1
	}
	return &request{
		app:  a,
		c:    c,
		user: M{"user_id": int64(7), "account": "legacy-user", "is_auth": isAuth},
		p:    params,
	}
}

func TestEditAccountVerificationDoesNotGrantIdentityVerification(t *testing.T) {
	a, mock := testApp(t)
	a.put("code:verified@example.com:4", "654321", time.Minute)
	mock.ExpectExec("UPDATE `yu_user` SET `account`=\\? WHERE user_id=\\?").
		WithArgs("verified@example.com", int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM `yu_imgo_session` WHERE user_id=\\?").
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err := a.im(accountVerificationRequest(a, false, M{
		"account": "verified@example.com",
		"newCode": "654321",
	}))
	if err != nil {
		t.Fatalf("first account verification failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEditAccountAuthenticatedUserStillRequiresCurrentAccountCode(t *testing.T) {
	a, mock := testApp(t)
	a.put("code:verified@example.com:4", "654321", time.Minute)

	_, err := a.im(accountVerificationRequest(a, true, M{
		"account": "verified@example.com",
		"code":    "",
		"newCode": "654321",
	}))
	if err == nil {
		t.Fatal("authenticated account changed without current-account verification")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
