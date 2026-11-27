package server

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestContactsKeepGroupNoticeAfterReload(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("sysInfo").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{}`))
	mock.ExpectQuery("SELECT u.user_id,u.account,u.realname").
		WithArgs(int64(7), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname"}))
	mock.ExpectQuery("SELECT from_user,COUNT").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"from_user", "n"}))
	mock.ExpectQuery("SELECT g\\.\\*,gu.role").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "name", "name_py", "owner_id", "role", "unread", "is_notice", "is_top", "setting", "notice", "joined_at",
		}).AddRow(5, "测试群", "ceshiqun", 7, 1, 0, 1, 0, `{}`, "刷新后仍显示的公告", 100))
	mock.ExpectQuery("SELECT m\\.\\* FROM `yu_message`").
		WithArgs(int64(7), int64(7), int64(7), int64(7), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"msg_id"}))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_message`").
		WithArgs("group-5", int64(7), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_friend`").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/enterprise/im/getContacts", nil)
	r := &request{app: a, c: c, p: M{}, user: M{"user_id": 7}}
	result, err := a.contacts(r, 7)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, contact := range result.([]M) {
		if str(contact["id"]) != "group-5" {
			continue
		}
		found = true
		if got := str(contact["notice"]); got != "刷新后仍显示的公告" {
			t.Fatalf("group notice = %q", got)
		}
	}
	if !found {
		t.Fatal("group contact missing")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
