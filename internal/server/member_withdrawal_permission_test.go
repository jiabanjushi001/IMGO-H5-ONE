package server

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func memberWithdrawalRequest(a *App, mode string) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/manage/user/setWithdrawal", nil)
	return &request{
		app:  a,
		c:    c,
		p:    M{"mode": mode},
		user: M{"user_id": int64(1)},
	}
}

func TestSetMemberWithdrawalPermissionModes(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		value any
	}{
		{memberWithdrawalEnabled, 1},
		{memberWithdrawalDisabled, 0},
		{memberWithdrawalInherit, nil},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").
				WithArgs(int64(8)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(8))
			mock.ExpectExec("UPDATE `yu_user` SET withdrawal_enabled=\\? WHERE user_id=\\? AND delete_time=0").
				WithArgs(tc.value, int64(8)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			result, err := a.setMemberWithdrawalPermission(memberWithdrawalRequest(a, tc.mode), 8, "user_id=? AND delete_time=0")
			if err != nil {
				t.Fatal(err)
			}
			if result.(M)["withdrawal_enabled"] != tc.value || result.(M)["mode"] != tc.mode {
				t.Fatalf("unexpected result: %#v", result)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSetMemberWithdrawalPermissionRejectsInvalidMode(t *testing.T) {
	a, mock := testApp(t)
	if _, err := a.setMemberWithdrawalPermission(memberWithdrawalRequest(a, "other"), 8, "user_id=? AND delete_time=0"); err == nil {
		t.Fatal("invalid personal withdrawal mode accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetMemberWithdrawalPermissionRouteUsesMemberScope(t *testing.T) {
	a, _ := testApp(t)
	route := a.routes[normalizedPath("/manage/user/setWithdrawal")]
	if route.super || route.permission != "manage.users" {
		t.Fatalf("unexpected route protection: %#v", route)
	}
}
