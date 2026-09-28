package server

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestOrdinaryUserProfileOmitsNetworkAuditFields(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE user_id=\\? AND status=1 AND delete_time=0").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "account", "realname", "avatar", "status", "delete_time",
			"create_time", "register_ip", "last_login_time", "last_login_ip", "last_chat_time", "last_chat_ip",
		}).AddRow(8, "member8", "成员8", "", 1, 0, 100, "1.1.1.1", 200, "2.2.2.2", 300, "3.3.3.3"))
	mock.ExpectQuery("SELECT friend_id,nickname,status FROM `yu_friend` WHERE create_user=\\? AND friend_user_id=\\?").
		WithArgs(int64(7), int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"friend_id", "nickname", "status"}))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/enterprise/im/getUserInfo", nil)
	r := &request{app: a, c: c, p: M{"user_id": int64(8)}, user: M{"user_id": int64(7), "admin_role_id": int64(0)}}
	result, err := a.im(r)
	if err != nil {
		t.Fatal(err)
	}
	profile := result.(M)
	if profile["can_view_network_info"] != false {
		t.Fatalf("ordinary profile access flag = %#v", profile["can_view_network_info"])
	}
	for _, key := range []string{"create_time", "register_ip", "reg_location", "last_login_time", "last_login_ip", "last_chat_time", "last_chat_ip", "chat_location", "location"} {
		if _, exists := profile[key]; exists {
			t.Fatalf("ordinary profile exposes %s: %#v", key, profile)
		}
	}
	if profile["account"] != "member8" || profile["realname"] != "成员8" {
		t.Fatalf("public identity fields missing: %#v", profile)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSuperAdminCanViewNetworkAuditFields(t *testing.T) {
	a, _ := testApp(t)
	allowed, err := a.canViewUserNetworkInfo(context.Background(), M{"user_id": int64(1)}, 8)
	if err != nil || !allowed {
		t.Fatalf("super admin access = %v, %v", allowed, err)
	}
}
