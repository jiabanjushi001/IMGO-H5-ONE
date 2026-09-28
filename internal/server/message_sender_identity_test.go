package server

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSerializedMessageSenderIncludesAccountAndNickname(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id,account,realname,avatar,sex,motto,name_py,role FROM").
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "avatar", "sex", "motto", "name_py", "role"}).
			AddRow(8, "member8", "成员八", "", 0, "", "chengyuanba", 0))

	message, err := a.serialize(context.Background(), M{
		"id":          "sender-identity",
		"msg_id":      int64(1),
		"from_user":   int64(8),
		"to_user":     int64(7),
		"is_group":    int64(0),
		"type":        "text",
		"content":     "hello",
		"create_time": int64(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	sender := obj(message["fromUser"])
	if sender["account"] != "member8" || sender["realname"] != "成员八" || sender["displayName"] != "成员八" {
		t.Fatalf("serialized sender identity = %#v", sender)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
