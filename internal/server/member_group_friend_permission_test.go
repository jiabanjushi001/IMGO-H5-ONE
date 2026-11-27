package server

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func groupFriendPermissionRequest(a *App, value int64) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/manage/user/setGroupFriendAdd", nil)
	return &request{
		app:  a,
		c:    c,
		p:    M{"group_friend_add": value},
		user: M{"user_id": int64(1)},
	}
}

func TestSetGroupFriendAddPermissionPersistsMemberFlag(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(8))
	mock.ExpectExec("UPDATE `yu_user` SET `group_friend_add`=\\? WHERE user_id=\\? AND delete_time=0").
		WithArgs(int64(1), int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := a.setGroupFriendAddPermission(groupFriendPermissionRequest(a, 1), 8, "user_id=? AND delete_time=0")
	if err != nil {
		t.Fatal(err)
	}
	if number(result.(M)["group_friend_add"]) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetGroupFriendAddPermissionRejectsInvalidValue(t *testing.T) {
	a, mock := testApp(t)
	if _, err := a.setGroupFriendAddPermission(groupFriendPermissionRequest(a, 2), 8, "user_id=? AND delete_time=0"); err == nil {
		t.Fatal("invalid group friend permission accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
