package server

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestQuickReplyRoutesAreAvailableToAuthenticatedChatUsers(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/enterprise/im/getquickreplies", "/enterprise/im/savequickreplies"} {
		route, ok := a.routes[path]
		if !ok || route.public || route.super || route.permission != "" {
			t.Fatalf("%s route = %#v", path, route)
		}
	}
}

func TestQuickReplyItemsNormalizesAndValidates(t *testing.T) {
	items, err := quickReplyItems([]any{"  欢迎咨询  ", "稍后回复", "欢迎咨询", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0] != "欢迎咨询" || items[1] != "稍后回复" {
		t.Fatalf("items = %#v", items)
	}
	if _, err = quickReplyItems([]any{strings.Repeat("好", maxQuickReplyLength+1)}); err == nil {
		t.Fatal("expected overlong reply to fail")
	}
}

func TestQuickRepliesAreStoredByCurrentMentor(t *testing.T) {
	a, mock := testApp(t)
	r := settingRequest(a, "/enterprise/im/saveQuickReplies", M{"items": []any{"欢迎咨询", "稍后回复"}})
	r.user = M{"user_id": int64(7), "admin_role_id": int64(3)}
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_setting`").
		WithArgs(int64(7), `["欢迎咨询","稍后回复"]`, int64(7), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	got, err := a.saveQuickReplies(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(obj(got)["items"].([]string)) != 2 {
		t.Fatalf("save result = %#v", got)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestQuickRepliesReadOnlyCurrentMentor(t *testing.T) {
	a, mock := testApp(t)
	r := settingRequest(a, "/enterprise/im/getQuickReplies", nil)
	r.user = M{"user_id": int64(9), "admin_role_id": int64(3)}
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(9))
	mock.ExpectQuery("SELECT quick_replies FROM `yu_imgo_agent_setting`").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"quick_replies"}).AddRow(`["九号导师用语"]`))
	got, err := a.manageQuickReplies(r)
	if err != nil {
		t.Fatal(err)
	}
	items := obj(got)["items"].([]string)
	if len(items) != 1 || items[0] != "九号导师用语" {
		t.Fatalf("items = %#v", items)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTeamAdminUsesMentorQuickRepliesWithoutEditPermission(t *testing.T) {
	a, mock := testApp(t)
	r := settingRequest(a, "/enterprise/im/getQuickReplies", nil)
	r.user = M{"user_id": int64(20), "admin_role_id": int64(4)}
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(20)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path`").WithArgs(int64(20)).
		WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT quick_replies FROM `yu_imgo_agent_setting`").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"quick_replies"}).AddRow(`["导师七的用语"]`))
	got, err := a.manageQuickReplies(r)
	if err != nil {
		t.Fatal(err)
	}
	data := obj(got)
	if number(data["owner_user_id"]) != 7 || data["can_manage"] != false || data["items"].([]string)[0] != "导师七的用语" {
		t.Fatalf("shared quick replies = %#v", data)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSuperAdminCannotModifyMentorQuickReplies(t *testing.T) {
	a, mock := testApp(t)
	r := settingRequest(a, "/enterprise/im/saveQuickReplies", M{"items": []any{"不可代改"}})
	_, err := a.saveQuickReplies(r)
	assertDenied(t, err)
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
