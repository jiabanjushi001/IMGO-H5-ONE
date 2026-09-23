package server

import (
	"context"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminPermissionCatalogIsStable(t *testing.T) {
	want := []string{
		"manage.overview",
		"manage.settings",
		"manage.users",
		"manage.messages",
		"manage.groups",
		"manage.files",
		"manage.bank",
		"manage.finance",
	}
	got := make([]string, 0, len(adminPermissionCatalog))
	for _, permission := range adminPermissionCatalog {
		got = append(got, permission.Key)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("permission catalog changed: got %v want %v", got, want)
	}
}

func TestAuthorizeManagePermissionMatrix(t *testing.T) {
	t.Run("super administrator bypasses database roles", func(t *testing.T) {
		a, mock := testApp(t)
		if err := a.authorizeManage(context.Background(), M{"user_id": 1, "admin_role_id": 0}, "manage.settings"); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ordinary user is denied", func(t *testing.T) {
		a, _ := testApp(t)
		assertDenied(t, a.authorizeManage(context.Background(), M{"user_id": 7, "admin_role_id": 0}, "manage.users"))
	})

	t.Run("enabled role with permission is allowed", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow("manage.users"))
		if err := a.authorizeManage(context.Background(), M{"user_id": 7, "admin_role_id": 3}, "manage.users"); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("role without permission is denied", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow("manage.users"))
		assertDenied(t, a.authorizeManage(context.Background(), M{"user_id": 7, "admin_role_id": 3}, "manage.groups"))
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("disabled role is denied", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"permission_key"}))
		assertDenied(t, a.authorizeManage(context.Background(), M{"user_id": 7, "admin_role_id": 3}, "manage.users"))
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestManageRoutesDeclarePermissions(t *testing.T) {
	a, _ := testApp(t)
	want := map[string]string{
		"/manage/index/overview": "manage.overview",
		"/manage/config/getinfo": "manage.settings",
		"/manage/user/index":     "manage.users",
		"/manage/message/index":  "manage.messages",
		"/manage/group/index":    "manage.groups",
		"/manage/files/index":    "manage.files",
		"/manage/bank/index":     "manage.bank",
		"/manage/wallet/index":   "manage.finance",
	}
	for path, permission := range want {
		endpoint, ok := a.routes[path]
		if !ok {
			t.Errorf("missing route %s", path)
			continue
		}
		if endpoint.permission != permission {
			t.Errorf("route %s permission = %q, want %q", path, endpoint.permission, permission)
		}
	}
	if endpoint := a.routes["/manage/user/setrole"]; !endpoint.super {
		t.Error("role assignment must remain super-admin only")
	}
	if endpoint := a.routes["/manage/role/save"]; !endpoint.super {
		t.Error("role management must remain super-admin only")
	}
}

func TestAdminAccessInfoForOrdinaryAndCustomRoles(t *testing.T) {
	t.Run("ordinary user only has chat access", func(t *testing.T) {
		a, _ := testApp(t)
		info, err := a.adminAccessInfo(context.Background(), M{"user_id": 7, "admin_role_id": 0})
		if err != nil || info["admin_role_name"] != "普通用户" || len(info["menu_permissions"].([]string)) != 0 {
			t.Fatalf("ordinary access info = %#v, %v", info, err)
		}
	})

	t.Run("custom role returns its name and permissions", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT name,status,agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"name", "status", "agent_mode"}).AddRow("客服", 1, 0))
		mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow("manage.users"))
		info, err := a.adminAccessInfo(context.Background(), M{"user_id": 7, "admin_role_id": 3})
		if err != nil || info["admin_role_name"] != "客服" || !reflect.DeepEqual(info["menu_permissions"], []string{"manage.users"}) {
			t.Fatalf("custom access info = %#v, %v", info, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func assertDenied(t *testing.T, err error) {
	t.Helper()
	ce, ok := err.(clientError)
	if !ok || ce.code != 403 || ce.message != "无权操作" {
		t.Fatalf("expected 403 无权操作, got %#v", err)
	}
}
