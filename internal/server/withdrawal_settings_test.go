package server

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWithdrawalSettingsValidation(t *testing.T) {
	for _, tc := range []struct {
		value M
		valid bool
	}{
		{M{"enabled": 1, "minYuan": 100, "passwordRequired": 1}, true},
		{M{"enabled": 0, "minYuan": 1}, true},
		{M{"enabled": 1, "minYuan": 0}, false},
		{M{"enabled": 2, "minYuan": 100}, false},
		{M{"enabled": 1, "minYuan": 100, "passwordRequired": 2}, false},
	} {
		_, err := decodeWithdrawalSettings(tc.value)
		if (err == nil) != tc.valid {
			t.Fatalf("decode %#v valid=%t, err=%v", tc.value, tc.valid, err)
		}
	}
}

func TestWithdrawalSettingsForUserUsesNearestMentorOverride(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("withdrawal").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"enabled":1,"minYuan":100,"passwordRequired":0}`))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled"}).AddRow(nil))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled,withdrawal_password_required FROM `yu_imgo_agent_setting` WHERE agent_user_id=?")).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled", "withdrawal_password_required"}).AddRow(0, 1))

	settings, err := a.withdrawalSettingsForUser(context.Background(), a.db, 20)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Enabled || settings.MinYuan != 100 || !settings.PasswordRequired {
		t.Fatalf("nearest mentor toggle was not applied: %#v", settings)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithdrawalSettingsForUserPrefersPersonalOverride(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("withdrawal").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"enabled":1,"minYuan":100,"passwordRequired":0}`))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled"}).AddRow(0))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled,withdrawal_password_required FROM `yu_imgo_agent_setting` WHERE agent_user_id=?")).
		WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled", "withdrawal_password_required"}).AddRow(1, 1))

	settings, err := a.withdrawalSettingsForUser(context.Background(), a.db, 20)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Enabled || settings.MinYuan != 100 || !settings.PasswordRequired {
		t.Fatalf("personal withdrawal override was not applied: %#v", settings)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithdrawalSettingsRoutesAreSuperAdminOnly(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/manage/config/getWithdrawalSettings", "/manage/config/setWithdrawalSettings"} {
		route := a.routes[normalizedPath(path)]
		if !route.super {
			t.Fatalf("%s must be super-admin only", path)
		}
	}
	_, err := a.manageConfig(bankRequest(a, "/manage/config/getWithdrawalSettings", 2, M{}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 403 {
		t.Fatalf("non-super admin must be denied: %v", err)
	}
}

func TestSuperAdminSavesWithdrawalSettings(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT id FROM `yu_config` WHERE name=\\?").WithArgs("withdrawal").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(8))
	mock.ExpectExec("UPDATE `yu_config` SET").WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.manageConfig(bankRequest(a, "/manage/config/setWithdrawalSettings", 1, M{"enabled": 1, "minYuan": 188, "passwordRequired": 1}))
	if err != nil || number(result.(M)["minYuan"]) != 188 || number(result.(M)["passwordRequired"]) != 1 {
		t.Fatalf("save withdrawal settings failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
