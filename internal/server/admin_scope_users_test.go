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

func expectMemberDecorators(mock sqlmock.Sqlmock, userIDs []int64) {
	args := make([]driver.Value, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}
	checkInArgs := append([]driver.Value{sqlmock.AnyArg(), sqlmock.AnyArg()}, args...)
	mock.ExpectQuery("SELECT user_id,COUNT\\(\\*\\) total_days").WithArgs(checkInArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "total_days", "last_date", "signed_today", "today_reward_cents", "total_reward_cents"}))
	walletArgs := append([]driver.Value{withdrawalStatusFrozen}, args...)
	mock.ExpectQuery("SELECT w.user_id,w.available_cents,w.pending_cents").WithArgs(walletArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "available_cents", "pending_cents", "frozen_cents"}))
	mock.ExpectQuery("SELECT user_id,invite_code FROM `yu_imgo_referral`").WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "invite_code"}))
	mock.ExpectQuery("SELECT p.ancestor_user_id,COUNT\\(\\*\\) team_count").WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id", "team_count", "direct_count"}))
	mock.ExpectQuery("SELECT p.descendant_user_id,parent.user_id parent_user_id,parent.account parent_account,parent.realname parent_name").WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "parent_user_id", "parent_account", "parent_name"}).AddRow(8, 7, "mentor", "导师七"))
	mock.ExpectQuery("SELECT role_id,name,agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"role_id", "name", "agent_mode"}).AddRow(3, "导师专员", 1))
	mock.ExpectQuery("SELECT p.descendant_user_id,mentor.user_id mentor_user_id").WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "mentor_user_id", "mentor_account", "mentor_name"}).AddRow(8, 7, "mentor", "导师七"))
}

func TestAgentManageUserListScopesEmptyDirectAndAll(t *testing.T) {
	for _, tc := range []struct{ name, scope, suffix string }{
		{"empty", "", ""}, {"all", "all", ""}, {"direct", "direct", " AND scope_depth.depth=1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			where := "u.delete_time=0 AND (u.user_id=? OR (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id)))"
			if tc.scope == "direct" {
				where = "u.delete_time=0 AND (u.user_id=? OR (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id) AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_depth WHERE scope_depth.ancestor_user_id=? AND scope_depth.descendant_user_id=u.user_id" + tc.suffix + ")))"
			}
			args := []driver.Value{int64(7), int64(7), int64(7)}
			if tc.scope == "direct" {
				args = append(args, int64(7))
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE " + where)).WithArgs(args...).
				WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT u.* FROM `yu_user` u WHERE " + where + " ORDER BY (u.user_id=?) DESC,u.user_id DESC LIMIT ? OFFSET ?")).
				WithArgs(append(args, int64(7), int64(20), int64(0))...).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
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

func TestAgentManageUserListMarksSelfAndReturnsItFirst(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	where := "u.delete_time=0 AND (u.user_id=? OR (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id)))"
	args := []driver.Value{int64(7), int64(7), int64(7)}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE " + where)).WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT u.* FROM `yu_user` u WHERE " + where + " ORDER BY (u.user_id=?) DESC,u.user_id DESC LIMIT ? OFFSET ?")).
		WithArgs(append(args, int64(7), int64(20), int64(0))...).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "admin_role_id"}).AddRow(7, "mentor", 3).AddRow(8, "child", 0))
	expectMemberDecorators(mock, []int64{7, 8})
	got, err := a.manageUser(agentMemberRequest(a, "/manage/user/index", M{"referral_scope": "all"}))
	if err != nil {
		t.Fatal(err)
	}
	list := got.([]M)
	if len(list) != 2 || number(list[0]["user_id"]) != 7 || number(list[0]["is_self"]) != 1 || number(list[1]["is_self"]) != 0 {
		t.Fatalf("self row not first/marked: %#v", list)
	}
	if str(list[0]["mentor_account"]) != "mentor" || number(list[0]["mentor_user_id"]) != 7 || str(list[1]["mentor_account"]) != "mentor" {
		t.Fatalf("mentor ownership missing: %#v", list)
	}
	if number(list[0]["parent_user_id"]) != 0 || number(list[1]["parent_user_id"]) != 7 || str(list[1]["parent_account"]) != "mentor" || str(list[1]["parent_name"]) != "导师七" {
		t.Fatalf("direct parent missing: %#v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageUserListCanShowOnlyMentorAccounts(t *testing.T) {
	a, mock := testApp(t)
	where := "delete_time=0 AND EXISTS (SELECT 1 FROM `yu_imgo_admin_role` mentor_filter WHERE mentor_filter.role_id=admin_role_id AND mentor_filter.status=1 AND mentor_filter.agent_mode=1)"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` WHERE " + where)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `yu_user` WHERE "+where+" ORDER BY user_id DESC LIMIT ? OFFSET ?")).
		WithArgs(int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	result, err := a.manageUser(bankRequest(a, "/manage/user/index", 1, M{"mentor_only": 1}))
	if err != nil || len(result.([]M)) != 0 {
		t.Fatalf("mentor-only result=%#v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
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
			base := "u.delete_time=0 AND (u.user_id=? OR (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id)))"
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE "+base+" AND u.account=?")).
				WithArgs(int64(7), int64(7), int64(7), "alice").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(tc.exact))
			where := base + " AND " + tc.match
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` u WHERE "+where)).
				WithArgs(int64(7), int64(7), int64(7), tc.value).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT u.* FROM `yu_user` u WHERE "+where+" ORDER BY (u.user_id=?) DESC,u.user_id DESC LIMIT ? OFFSET ?")).
				WithArgs(int64(7), int64(7), int64(7), tc.value, int64(7), int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
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

func TestManageUserEditPasswordClearsLoginFailureLock(t *testing.T) {
	a, mock := testApp(t)
	const account = "15177301318"
	const userID = int64(4768)
	for range 5 {
		a.recordFailure("login-fail:" + account)
	}
	if got := number(a.get("login-fail:"+account, false)); got != 5 {
		t.Fatalf("login failure count = %d, want 5", got)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT account FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"account"}).AddRow(account))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_user` SET `password`=?,`salt`=? WHERE user_id=? AND delete_time=0")).
		WithArgs(sqlmock.AnyArg(), "", userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `yu_imgo_session` WHERE user_id=?")).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := a.manageUser(bankRequest(a, "/manage/user/editPassword", 1, M{"user_id": userID, "password": "123123"})); err != nil {
		t.Fatal(err)
	}
	if got := a.get("login-fail:"+account, false); got != nil {
		t.Fatalf("login failure lock remains after administrator password reset: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
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

func expectAgentInviteCodePrechecks(mock sqlmock.Sqlmock, userID int64) {
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").
		WithArgs(userID, int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_user` WHERE user_id=? AND role=0 AND user_id<>1 AND delete_time=0")).
		WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(userID))
}

func TestAgentManageUserSetInviteCodeRejectsMemberPromotedBeforeLock(t *testing.T) {
	a, mock := testApp(t)
	expectAgentInviteCodePrechecks(mock, 12)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_user` WHERE user_id=? AND delete_time=0 AND role=0 AND user_id<>1 FOR UPDATE")).
		WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectRollback()
	_, err := a.manageUser(agentMemberRequest(a, "/manage/user/setInviteCode", M{"user_id": 12, "invite_code": "654321"}))
	assertDenied(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentManageUserSetInviteCodeUpdatesOnlyAfterLockedScopeCheck(t *testing.T) {
	a, mock := testApp(t)
	expectAgentInviteCodePrechecks(mock, 12)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_user` WHERE user_id=? AND delete_time=0 AND role=0 AND user_id<>1 FOR UPDATE")).
		WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").
		WithArgs(int64(12), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT invite_code FROM `yu_imgo_referral` WHERE user_id=?")).
		WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"invite_code"}).AddRow("123456"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_referral` WHERE invite_code=?")).
		WithArgs("654321").WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_imgo_referral` SET invite_code=? WHERE user_id=?")).
		WithArgs("654321", int64(12)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := a.manageUser(agentMemberRequest(a, "/manage/user/setInviteCode", M{"user_id": 12, "invite_code": "654321"}))
	if err != nil || str(result.(M)["invite_code"]) != "654321" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
