package server

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGroupCreatePermission(t *testing.T) {
	tests := []struct {
		name       string
		user       M
		config     M
		agentMode  *int64
		wantDenied bool
	}{
		{name: "all users", user: M{"user_id": 8}, config: M{"groupCreateRole": groupCreateRoleAll}},
		{name: "super administrator", user: M{"user_id": 1}, config: M{}},
		{name: "ordinary user", user: M{"user_id": 8, "admin_role_id": 0}, config: M{}, wantDenied: true},
		{name: "mentor", user: M{"user_id": 8, "admin_role_id": 3}, config: M{}, agentMode: int64Pointer(1)},
		{name: "non mentor administrator", user: M{"user_id": 8, "admin_role_id": 4}, config: M{}, agentMode: int64Pointer(0), wantDenied: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, mock := testApp(t)
			if tt.agentMode != nil {
				mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role` WHERE role_id=\\? AND status=1").
					WithArgs(number(tt.user["admin_role_id"])).
					WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(*tt.agentMode))
			}
			err := a.requireGroupCreatePermission(context.Background(), tt.user, tt.config)
			var clientErr clientError
			if tt.wantDenied {
				if !errors.As(err, &clientErr) || clientErr.code != 403 || clientErr.message != "权限不足" {
					t.Fatalf("error = %#v, want 403 权限不足", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInactiveMentorCannotCreateGroup(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role` WHERE role_id=\\? AND status=1").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}))
	err := a.requireGroupCreatePermission(context.Background(), M{"user_id": 8, "admin_role_id": 3}, M{})
	var clientErr clientError
	if !errors.As(err, &clientErr) || clientErr.message != "权限不足" {
		t.Fatalf("error = %#v, want 权限不足", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func int64Pointer(value int64) *int64 { return &value }
