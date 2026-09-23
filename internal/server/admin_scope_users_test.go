package server

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func agentMemberRequest(a *App, path string, params M) *request {
	r := bankRequest(a, path, 7, params)
	r.user["admin_role_id"] = int64(3)
	return r
}

func expectAgentMemberScope(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(1))
}

func TestAgentManageUserListScopesEmptyDirectAndAll(t *testing.T) {
	for _, tc := range []struct{ name, scope, suffix string }{
		{"empty", "", ""}, {"all", "all", ""}, {"direct", "direct", " AND scope_depth.depth=1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			where := "u.delete_time=0 AND (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id))"
			if tc.scope == "direct" {
				where += " AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_depth WHERE scope_depth.ancestor_user_id=? AND scope_depth.descendant_user_id=u.user_id" + tc.suffix + ")"
			}
			args := []driver.Value{int64(7), int64(7)}
			if tc.scope == "direct" {
				args = append(args, int64(7))
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE " + where)).WithArgs(args...).
				WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT u.* FROM `yu_user` u WHERE " + where + " ORDER BY u.user_id DESC LIMIT ? OFFSET ?")).
				WithArgs(append(args, int64(20), int64(0))...).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
			r := agentMemberRequest(a, "/manage/user/index", M{"referral_scope": tc.scope})
			result, err := a.manageUser(r)
			if err != nil || len(result.([]M)) != 0 {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentManageUserSearchKeepsExactAndFuzzyInsideScope(t *testing.T) {
	for _, tc := range []struct {
		name         string
		exact        int
		match, value string
	}{
		{"exact", 1, "u.account=?", "alice"},
		{"fuzzy after no scoped exact", 0, "u.account LIKE ? ESCAPE '!'", "%alice%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			base := "u.delete_time=0 AND (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id))"
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE "+base+" AND u.account=?")).
				WithArgs(int64(7), int64(7), "alice").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(tc.exact))
			where := base + " AND " + tc.match
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE "+where)).
				WithArgs(int64(7), int64(7), tc.value).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT u.* FROM `yu_user` u WHERE "+where+" ORDER BY u.user_id DESC LIMIT ? OFFSET ?")).
				WithArgs(int64(7), int64(7), tc.value, int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
			_, err := a.manageUser(agentMemberRequest(a, "/manage/user/index", M{"keywords": " alice "}))
			if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentManageUserRejectsOutOfScopeReadAndWrites(t *testing.T) {
	for _, action := range []string{"detail", "checkInHistory", "setRemark", "setInviteCode", "edit", "setStatus", "editPassword", "del"} {
		t.Run(action, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").
				WithArgs(int64(99), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}))
			_, err := a.manageUser(agentMemberRequest(a, "/manage/user/"+action, M{"user_id": 99, "password": "test-password", "invite_code": "123456"}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentManageUserAddBindsNewMemberInSameTransaction(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `yu_user`").WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectQuery("SELECT invite_code FROM `yu_imgo_referral` WHERE user_id=").WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"invite_code"}))
	mock.ExpectExec("INSERT INTO `yu_imgo_referral`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `yu_imgo_referral` SET parent_user_id=").WithArgs(int64(7), int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_referral_path` \\(ancestor_user_id,descendant_user_id,depth\\) VALUES").
		WithArgs(int64(7), int64(42)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_referral_path` \\(ancestor_user_id,descendant_user_id,depth\\) SELECT").
		WithArgs(int64(42), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := a.manageUser(agentMemberRequest(a, "/manage/user/add", M{"account": "newmember", "password": "test-password"}))
	if err != nil || number(result.(M)["user_id"]) != 42 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
