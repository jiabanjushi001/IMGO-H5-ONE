package server

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckInSettingsRoutesAllowMentorSelfService(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/manage/config/getCheckInSettings", "/manage/config/setCheckInSettings"} {
		route, ok := a.routes[normalizedPath(path)]
		if !ok || route.public || route.super || route.permission != "" {
			t.Fatalf("%s route = %#v", path, route)
		}
	}
}

func TestMentorReadsOwnCheckInSettings(t *testing.T) {
	a, mock := testApp(t)
	r := bankRequest(a, "/manage/config/getCheckInSettings", 7, M{})
	r.user["admin_role_id"] = int64(3)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT agent_mode FROM `yu_imgo_admin_role` WHERE role_id=? AND status=1")).
		WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT check_in_setting FROM `yu_imgo_agent_setting` WHERE agent_user_id=?")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"check_in_setting"}).AddRow(`{"rewardEnabled":1,"cycleDays":14,"highDays":4,"highMinYuan":8,"highMaxYuan":12,"normalMinYuan":2,"normalMaxYuan":6}`))

	got, err := a.manageConfig(r)
	if err != nil {
		t.Fatal(err)
	}
	value := got.(M)
	if value["setting_scope"] != "mentor" || value["inherited"] != false || number(value["cycleDays"]) != 14 || number(value["highMaxYuan"]) != 12 {
		t.Fatalf("unexpected mentor settings: %#v", value)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMentorUnsavedCheckInSettingsInheritGlobal(t *testing.T) {
	a, mock := testApp(t)
	r := bankRequest(a, "/manage/config/getCheckInSettings", 7, M{})
	r.user["admin_role_id"] = int64(3)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT agent_mode FROM `yu_imgo_admin_role` WHERE role_id=? AND status=1")).
		WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT check_in_setting FROM `yu_imgo_agent_setting` WHERE agent_user_id=?")).
		WithArgs(int64(7)).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("checkIn").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(checkInTestConfig))

	got, err := a.manageConfig(r)
	if err != nil {
		t.Fatal(err)
	}
	value := got.(M)
	if value["setting_scope"] != "mentor" || value["inherited"] != true || number(value["cycleDays"]) != 10 {
		t.Fatalf("unexpected inherited settings: %#v", value)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMentorSavesOnlyOwnCheckInSettings(t *testing.T) {
	a, mock := testApp(t)
	params := M{"rewardEnabled": 1, "cycleDays": 12, "highDays": 2, "highMinYuan": 9, "highMaxYuan": 11, "normalMinYuan": 3, "normalMaxYuan": 4}
	r := bankRequest(a, "/manage/config/setCheckInSettings", 7, params)
	r.user["admin_role_id"] = int64(3)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT agent_mode FROM `yu_imgo_admin_role` WHERE role_id=? AND status=1")).
		WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `yu_imgo_agent_setting` (agent_user_id,check_in_setting,updated_by,updated_at) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE check_in_setting=VALUES(check_in_setting),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)")).
		WithArgs(int64(7), `{"cycleDays":12,"highDays":2,"highMaxYuan":11,"highMinYuan":9,"normalMaxYuan":4,"normalMinYuan":3,"rewardEnabled":1}`, int64(7), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	got, err := a.manageConfig(r)
	if err != nil {
		t.Fatal(err)
	}
	if value := got.(M); value["setting_scope"] != "mentor" || value["inherited"] != false || number(value["cycleDays"]) != 12 {
		t.Fatalf("unexpected saved settings: %#v", value)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryUserCannotManageCheckInSettings(t *testing.T) {
	a, mock := testApp(t)
	_, err := a.manageConfig(bankRequest(a, "/manage/config/getCheckInSettings", 8, M{}))
	var denied clientError
	if !errors.As(err, &denied) || denied.code != 403 {
		t.Fatalf("ordinary user was not denied: %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserCheckInUsesNearestMentorRules(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("checkIn").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(checkInTestConfig))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT check_in_setting FROM `yu_imgo_agent_setting` WHERE agent_user_id=?")).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"check_in_setting"}).AddRow(`{"rewardEnabled":1,"cycleDays":7,"highDays":1,"highMinYuan":20,"highMaxYuan":20,"normalMinYuan":2,"normalMaxYuan":2}`))

	settings, err := a.checkInSettingsForUser(context.Background(), a.db, 20)
	if err != nil {
		t.Fatal(err)
	}
	if settings.CycleDays != 7 || settings.HighMinYuan != 20 || settings.NormalMaxYuan != 2 {
		t.Fatalf("nearest mentor settings were not applied: %#v", settings)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
