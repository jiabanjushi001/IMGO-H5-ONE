package server

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAgentNewGroupMemberPickerOnlyListsDescendants(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT user_id,account,realname,avatar,name_py FROM `yu_user` u.*scope_path").
		WithArgs(int64(7), int64(0), int64(7), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "avatar", "name_py"}).AddRow(8, "child8", "直属成员", "", "zhishu"))

	result, err := a.group(agentMemberRequest(a, "/enterprise/group/getAllUser", M{}))
	if err != nil {
		t.Fatal(err)
	}
	users := result.([]M)
	if len(users) != 1 || number(users[0]["user_id"]) != 8 {
		t.Fatalf("member picker result = %#v", result)
	}
	if users[0]["member_label"] != "直属成员（child8）" {
		t.Fatalf("member picker label = %#v", users[0]["member_label"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentCreateGroupRejectsUsersOutsideOwnTeam(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	scope := adminScope{AgentUserID: 7, referralTable: a.t("imgo_referral_path")}
	expectResourceUser(mock, 8, true)
	expectResourceUser(mock, 99, false)
	err = a.requireCreateGroupUsers(context.Background(), tx, scope, 7, []int64{7, 8, 99})
	assertDenied(t, err)
	mock.ExpectRollback()
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentCreateGroupAllowsOwnDescendants(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	scope := adminScope{AgentUserID: 7, referralTable: a.t("imgo_referral_path")}
	expectResourceUser(mock, 8, true)
	expectResourceUser(mock, 9, true)
	if err = a.requireCreateGroupUsers(context.Background(), tx, scope, 7, []int64{7, 8, 9}); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentCanAddDescendantToOwnGroup(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(7))
	expectResourceUser(mock, 13, true)
	mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "name", "setting"}).AddRow(9, 7, "导师的群", `{}`))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT group_id,owner_id FROM `yu_group`.*FOR UPDATE").WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(9, 7))
	expectLockedResourceUser(mock, 13, true)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_group_user`").WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("chatInfo").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"groupUserMax":100}`))
	mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs(int64(13)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(13))
	mock.ExpectQuery("SELECT id FROM `yu_group_user`").WithArgs(int64(9), int64(13)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT INTO `yu_group_user`").
		WithArgs(sqlmock.AnyArg(), int64(9), int64(7), int64(3), int64(1), int64(13)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7).AddRow(13))

	_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/addGroupUser", M{"group_id": 9, "user_ids": []any{13}}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
