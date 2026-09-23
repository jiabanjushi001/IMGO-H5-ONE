package server

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestLoginSucceedsWithDisconnectedSocket(t *testing.T) {
	a, mock := testApp(t)
	password, err := hashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE account=\\? AND delete_time=0").
		WithArgs("wuhu2").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "password", "salt", "status", "delete_time"}).
			AddRow(2, "wuhu2", "wuhu2", password, "", 1, 0))
	mock.ExpectExec("UPDATE `yu_user` SET login_count=").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_session`").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(2)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/pub/login", nil)
	r := &request{app: a, c: c, p: M{"account": "wuhu2", "password": "test-password", "client_id": "closed-socket"}}
	result, err := a.login(r)
	if err != nil {
		t.Fatal(err)
	}
	if str(obj(result)["authToken"]) == "" {
		t.Fatal("missing auth token")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
