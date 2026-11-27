package server

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPublishRegistrationContactIncludesWelcomeWithoutReply(t *testing.T) {
	a, mock := testApp(t)
	customer := &peer{
		id:     "customer-device",
		uid:    2,
		claims: claims{UID: 2, Exp: time.Now().Add(time.Hour).Unix()},
		send:   make(chan []byte, 2),
		done:   make(chan struct{}),
	}
	a.hub.peers[customer.id] = customer
	t.Cleanup(func() { delete(a.hub.peers, customer.id) })

	mock.ExpectQuery("SELECT user_id,account,realname,avatar,sex,motto,name_py,role,cs_uid FROM `yu_user` WHERE user_id=\\? AND status=1 AND delete_time=0").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "avatar", "sex", "motto", "name_py", "role", "cs_uid"}).
			AddRow(9, "wuhu00", "新用户", "", 1, "", "xinyonghu", 0, 2))
	mock.ExpectQuery("SELECT \\* FROM `yu_message` WHERE from_user=\\? AND to_user=\\? AND is_group=0 AND status=1 ORDER BY msg_id DESC LIMIT 1").
		WithArgs(int64(2), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"msg_id", "id", "from_user", "to_user", "chat_identify", "content", "type", "create_time", "is_group", "is_read", "file_id", "file_cate", "file_name", "file_size", "at", "extends", "pid"}).
			AddRow(81, "welcome-message", 2, 9, "2_9", "欢迎新成员", "text", 100, 0, 0, 0, 0, "", 0, "", "{}", 0))
	mock.ExpectQuery("SELECT user_id,account,realname,avatar,sex,motto,name_py,role FROM `yu_user` WHERE user_id=\\?").
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "avatar", "sex", "motto", "name_py", "role"}).
			AddRow(2, "service", "客服", "", 1, "", "kefu", 1))

	a.publishRegistrationContact(context.Background(), 9)

	select {
	case payload := <-customer.send:
		var event M
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatal(err)
		}
		contact := obj(event["data"])
		if str(event["type"]) != "appendContact" || number(contact["id"]) != 9 || str(contact["account"]) != "wuhu00" || str(contact["lastContent"]) != "欢迎新成员" {
			t.Fatalf("registered contact was not appended first: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("online customer service did not receive the registered contact")
	}
	select {
	case payload := <-customer.send:
		var event M
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatal(err)
		}
		data := obj(event["data"])
		contact := obj(data["contactInfo"])
		if str(event["type"]) != "simple" || number(data["toContactId"]) != 9 || str(data["content"]) != "欢迎新成员" {
			t.Fatalf("unexpected registration event: %#v", event)
		}
		if number(contact["id"]) != 9 || str(contact["account"]) != "wuhu00" || str(contact["lastContent"]) != "欢迎新成员" {
			t.Fatalf("registered contact missing from event: %#v", contact)
		}
	case <-time.After(time.Second):
		t.Fatal("online customer service did not receive the welcome message")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAddMutualFriendship(t *testing.T) {
	a, mock := testApp(t)
	query := regexp.QuoteMeta("INSERT INTO `yu_friend` (`create_time`,`create_user`,`friend_user_id`,`status`) VALUES (?,?,?,?)")
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(2), int64(9), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(9), int64(2), 1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	if e := a.addMutualFriendship(context.Background(), a.db, 2, 9, 100); e != nil {
		t.Fatal(e)
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}

func TestAddMutualFriendshipStopsOnFailure(t *testing.T) {
	a, mock := testApp(t)
	query := regexp.QuoteMeta("INSERT INTO `yu_friend` (`create_time`,`create_user`,`friend_user_id`,`status`) VALUES (?,?,?,?)")
	want := errors.New("friend insert failed")
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(2), int64(9), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(9), int64(2), 1).
		WillReturnError(want)

	if e := a.addMutualFriendship(context.Background(), a.db, 2, 9, 100); !errors.Is(e, want) {
		t.Fatalf("error = %v, want %v", e, want)
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}
