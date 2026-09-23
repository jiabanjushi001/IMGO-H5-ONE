package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func settingRequest(a *App, path string, p M) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", path, nil)
	return &request{app: a, c: c, p: p, user: M{"user_id": int64(1)}}
}

func TestAgentSettingRoutesRequireSuperAdministrator(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/manage/agentsetting/detail", "/manage/agentsetting/save"} {
		route, ok := a.routes[path]
		if !ok || !route.super {
			t.Fatalf("%s must be a super administrator route", path)
		}
	}
}

func TestAgentSettingDetailReturnsIndependentInheritance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT auto_add_user,auto_add_group FROM `yu_imgo_agent_setting`").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"auto_add_user", "auto_add_group"}).AddRow(nil, `{"status":0}`))
	mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("chatInfo").WillReturnRows(sqlmock.NewRows([]string{"value"}))
	got, err := a.manageAgentSetting(settingRequest(a, "/manage/agentSetting/detail", M{"agent_user_id": 7}))
	if err != nil {
		t.Fatal(err)
	}
	v := obj(got)
	if v["inherit_auto_user"] != true || v["inherit_auto_group"] != false || number(obj(v["auto_add_group"])["status"]) != 0 {
		t.Fatalf("detail = %#v", v)
	}
	if _, ok := v["global_auto_add_user"]; !ok {
		t.Fatalf("missing global user summary: %#v", v)
	}
	if _, ok := v["global_auto_add_group"]; !ok {
		t.Fatalf("missing global group summary: %#v", v)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentSettingSavePersistsNullAndExplicitDisabledJSON(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_setting`").
		WithArgs(int64(7), nil, `{"status":0}`, int64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := a.manageAgentSetting(settingRequest(a, "/manage/agentSetting/save", M{
		"agent_user_id": 7, "inherit_auto_user": true, "inherit_auto_group": false,
		"auto_add_group": M{"status": 0},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentSettingRejectsNonMentorAndOutOfScopeUsers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		p       M
		scopeID int64
	}{
		{"customer", M{"agent_user_id": 7, "inherit_auto_user": false, "inherit_auto_group": true, "auto_add_user": M{"status": 1, "user_ids": []any{99}}}, 99},
		{"owner", M{"agent_user_id": 7, "inherit_auto_user": true, "inherit_auto_group": false, "auto_add_group": M{"status": 1, "owner_uid": 99, "userMax": 5}}, 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
			mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE").WithArgs(tc.scopeID, int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}))
			_, err := a.manageAgentSetting(settingRequest(a, "/manage/agentSetting/save", tc.p))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	_, err := a.manageAgentSetting(settingRequest(a, "/manage/agentSetting/detail", M{"agent_user_id": 7}))
	assertDenied(t, err)
}

func TestAgentSettingRejectsMalformedCustomerList(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT .* FROM `yu_user`.*`yu_imgo_admin_role`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	_, err := a.manageAgentSetting(settingRequest(a, "/manage/agentSetting/save", M{
		"agent_user_id": 7, "inherit_auto_user": false, "inherit_auto_group": true,
		"auto_add_user": M{"status": 1, "user_ids": []any{0}},
	}))
	var client clientError
	if !errors.As(err, &client) || client.code != 400 {
		t.Fatalf("invalid customer ID accepted: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationAutomationInheritanceMatrix(t *testing.T) {
	for _, tc := range []struct {
		name                string
		user, group         any
		wantUser, wantGroup bool
	}{
		{"both inherit", nil, nil, true, true},
		{"customer override", `{"status":0}`, nil, false, true},
		{"group override", nil, `{"status":0}`, true, false},
		{"both override", `{"status":0}`, `{"status":1,"owner_uid":7}`, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(9))
			mock.ExpectQuery("SELECT auto_add_user,auto_add_group FROM `yu_imgo_agent_setting`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"auto_add_user", "auto_add_group"}).AddRow(tc.user, tc.group))
			if tc.wantUser || tc.wantGroup {
				mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("chatInfo").WillReturnRows(sqlmock.NewRows([]string{"value"}))
			}
			got, err := a.registrationAutomation(context.Background(), a.db, 9)
			if err != nil {
				t.Fatal(err)
			}
			if got.AgentUserID != 9 || got.UserInherited != tc.wantUser || got.GroupInherited != tc.wantGroup {
				t.Fatalf("automation = %#v", got)
			}
			if !tc.wantUser && number(got.AutoUser["status"]) != 0 {
				t.Fatalf("user override = %#v", got.AutoUser)
			}
			if !tc.wantGroup && number(got.AutoGroup["status"]) != number(obj(tc.group)["status"]) {
				t.Fatalf("group override = %#v", got.AutoGroup)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRegistrationAutomationSelectsNearestMentorForDescendantInviter(t *testing.T) {
	for _, tc := range []struct {
		name  string
		agent int64
	}{{"grandchild invitation", 7}, {"nested mentor", 8}} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role`").WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
			mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path`").WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(tc.agent))
			mock.ExpectQuery("SELECT auto_add_user,auto_add_group FROM `yu_imgo_agent_setting`").WithArgs(tc.agent).WillReturnRows(sqlmock.NewRows([]string{"auto_add_user", "auto_add_group"}))
			mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("chatInfo").WillReturnRows(sqlmock.NewRows([]string{"value"}))
			got, err := a.registrationAutomation(context.Background(), a.db, 20)
			if err != nil || got.AgentUserID != tc.agent {
				t.Fatalf("automation = %#v, %v", got, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRegistrationAutomationLocksAndPersistsMentorState(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_auto_state`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT last_customer_user_id,group_id,group_num FROM `yu_imgo_agent_auto_state` WHERE agent_user_id=\\? FOR UPDATE").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"last_customer_user_id", "group_id", "group_num"}).AddRow(2, 40, 3))
	mock.ExpectExec("UPDATE `yu_imgo_agent_auto_state`").WithArgs(int64(3), int64(41), int64(4), sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()
	tx, err := a.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	state, err := a.loadAgentAutoState(context.Background(), tx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if number(state["user_id"]) != 2 || number(state["group_id"]) != 40 || number(state["group_num"]) != 3 {
		t.Fatalf("state = %#v", state)
	}
	state["user_id"], state["group_id"], state["group_num"] = int64(3), int64(41), int64(4)
	if err := a.saveAgentAutoState(context.Background(), tx, 7, state); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationAutomationReadsCurrentSettingsAfterWaitingForLock(t *testing.T) {
	for _, tc := range []struct {
		name      string
		user      any
		group     any
		wantUser  int64
		wantGroup int64
	}{
		{"inherit changed to override", `{"status":1,"user_ids":[7]}`, `{"status":1,"owner_uid":7}`, 7, 7},
		{"override changed to inherit", nil, nil, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectBegin()
			// The next rows represent a save that committed while this registration
			// waited for the registration lock. Locking reads must see these rows.
			mock.ExpectExec("INSERT INTO `yu_imgo_chat_lock`").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("SELECT value,status FROM `yu_config` WHERE name='chatInfo' LIMIT 1 FOR UPDATE").
				WillReturnRows(sqlmock.NewRows([]string{"value", "status"}).AddRow(`{"autoAddUser":{"status":1,"user_ids":[2]},"autoAddGroup":{"status":1,"owner_uid":1}}`, 1))
			mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role`").WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
			mock.ExpectExec("INSERT INTO `yu_imgo_agent_setting`").WithArgs(int64(7), int64(0), int64(0)).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("SELECT auto_add_user,auto_add_group FROM `yu_imgo_agent_setting` WHERE agent_user_id=\\? FOR UPDATE").WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"auto_add_user", "auto_add_group"}).AddRow(tc.user, tc.group))
			mock.ExpectRollback()
			tx, err := a.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			got, err := a.registrationAutomation(context.Background(), tx, 7)
			if err != nil {
				t.Fatal(err)
			}
			customers := ids(got.AutoUser["user_ids"])
			if got.AgentUserID != 7 || len(customers) != 1 || customers[0] != tc.wantUser || number(got.AutoGroup["owner_uid"]) != tc.wantGroup {
				t.Fatalf("stale automation selected: %#v", got)
			}
			_ = tx.Rollback()
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
