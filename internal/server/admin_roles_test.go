package server

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestManageRolePermissionsReturnsAssignableCatalog(t *testing.T) {
	a, _ := testApp(t)
	result, err := a.manageRole(bankRequest(a, "/manage/role/permissions", 1, M{}))
	if err != nil {
		t.Fatal(err)
	}
	permissions := result.([]M)
	if len(permissions) != 8 || permissions[0]["permission_key"] != "manage.overview" || permissions[7]["permission_key"] != "manage.finance" {
		t.Fatalf("unexpected permissions: %#v", permissions)
	}
}

func TestManageRoleIndexIncludesBuiltInRoles(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT r.role_id,r.name,r.remark,r.status").
		WillReturnRows(sqlmock.NewRows([]string{"role_id", "name", "remark", "status", "created_at", "updated_at", "user_count", "permissions"}).
			AddRow(5, "客服", "", 1, 10, 10, 2, "manage.users"))
	mock.ExpectQuery("SELECT SUM\\(CASE WHEN user_id=1").
		WillReturnRows(sqlmock.NewRows([]string{"super_count", "ordinary_count"}).AddRow(1, 6))

	result, err := a.manageRole(bankRequest(a, "/manage/role/index", 1, M{}))
	if err != nil {
		t.Fatal(err)
	}
	roles := result.([]M)
	if len(roles) != 3 || roles[0]["name"] != "超级管理员" || roles[1]["name"] != "普通用户" || roles[2]["name"] != "客服" {
		t.Fatalf("built-in role order = %#v", roles)
	}
	if roles[0]["builtin"] != true || number(roles[0]["user_count"]) != 1 || len(roles[0]["permissions"].([]string)) != len(adminPermissionCatalog) {
		t.Fatalf("super role = %#v", roles[0])
	}
	if roles[1]["builtin"] != true || number(roles[1]["user_count"]) != 6 || len(roles[1]["permissions"].([]string)) != 0 {
		t.Fatalf("ordinary role = %#v", roles[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageRoleSaveCreatesRoleWithSelectedPermissions(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `yu_imgo_admin_role`").
		WithArgs("客服", "只管理成员", int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(5, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_admin_role_permission`").WithArgs(int64(5), "manage.users").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := a.manageRole(bankRequest(a, "/manage/role/save", 1, M{
		"name": "客服", "remark": "只管理成员", "status": 1,
		"permissions": []any{"manage.users"},
	}))
	if err != nil || number(result.(M)["role_id"]) != 5 {
		t.Fatalf("save role: result=%#v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetAdminRoleAssignsExactlyOneRole(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE role_id=? AND status=1")).WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(3))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_user` SET `admin_role_id`=?,`role`=? WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(3), int64(2), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `yu_imgo_session` WHERE user_id=?")).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := a.manageUser(bankRequest(a, "/manage/user/setRole", 1, M{"user_id": 7, "admin_role_id": 3}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAttachAdminRoleNamesDecoratesMemberRows(t *testing.T) {
	a, mock := testApp(t)
	users := []M{
		{"user_id": int64(1), "admin_role_id": int64(0)},
		{"user_id": int64(7), "admin_role_id": int64(3)},
		{"user_id": int64(8), "admin_role_id": int64(0)},
	}
	mock.ExpectQuery("SELECT role_id,name FROM `yu_imgo_admin_role` WHERE role_id IN").WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"role_id", "name"}).AddRow(3, "客服"))
	if err := a.attachAdminRoleNames(bankRequest(a, "/manage/user/index", 1, M{}), users); err != nil {
		t.Fatal(err)
	}
	if users[0]["admin_role_name"] != "超级管理员" || users[1]["admin_role_name"] != "客服" || users[2]["admin_role_name"] != "普通用户" {
		t.Fatalf("unexpected role names: %#v", users)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
