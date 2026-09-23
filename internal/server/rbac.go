package server

import (
	"context"
	"database/sql"
	"errors"
	"slices"
)

type adminPermission struct {
	Key      string
	Name     string
	MenuPath string
	Sort     int
}

var adminPermissionCatalog = []adminPermission{
	{Key: "manage.overview", Name: "概况", MenuPath: "/manage/index", Sort: 10},
	{Key: "manage.settings", Name: "设置", MenuPath: "/manage/setting", Sort: 20},
	{Key: "manage.users", Name: "成员", MenuPath: "/manage/user", Sort: 30},
	{Key: "manage.messages", Name: "消息", MenuPath: "/manage/message", Sort: 40},
	{Key: "manage.groups", Name: "群聊", MenuPath: "/manage/group", Sort: 50},
	{Key: "manage.files", Name: "文件", MenuPath: "/manage/files", Sort: 60},
	{Key: "manage.bank", Name: "绑卡", MenuPath: "/manage/bank", Sort: 70},
	{Key: "manage.finance", Name: "财务", MenuPath: "/manage/finance", Sort: 80},
}

func allAdminPermissionKeys() []string {
	permissions := make([]string, 0, len(adminPermissionCatalog))
	for _, permission := range adminPermissionCatalog {
		permissions = append(permissions, permission.Key)
	}
	return permissions
}

func (a *App) menuPermissions(ctx context.Context, user M) ([]string, error) {
	if number(user["user_id"]) == 1 {
		return allAdminPermissionKeys(), nil
	}
	roleID := number(user["admin_role_id"])
	if roleID < 1 {
		return []string{}, nil
	}
	list, err := rows(ctx, a.db,
		"SELECT p.permission_key FROM "+a.t("imgo_admin_role")+" r "+
			"JOIN "+a.t("imgo_admin_role_permission")+" rp ON rp.role_id=r.role_id "+
			"JOIN "+a.t("imgo_admin_permission")+" p ON p.permission_id=rp.permission_id "+
			"WHERE r.role_id=? AND r.status=1 ORDER BY p.sort,p.permission_id", roleID)
	if err != nil {
		return nil, err
	}
	permissions := make([]string, 0, len(list))
	for _, item := range list {
		permissions = append(permissions, str(item["permission_key"]))
	}
	return permissions, nil
}

func (a *App) authorizeManage(ctx context.Context, user M, permission string) error {
	if number(user["user_id"]) == 1 {
		return nil
	}
	if permission == "" || number(user["admin_role_id"]) < 1 {
		return deny()
	}
	permissions, err := a.menuPermissions(ctx, user)
	if err != nil {
		return err
	}
	if !slices.Contains(permissions, permission) {
		return deny()
	}
	return nil
}

func (a *App) adminAccessInfo(ctx context.Context, user M) (M, error) {
	roleID := number(user["admin_role_id"])
	if number(user["user_id"]) == 1 {
		return M{"admin_role_id": int64(0), "admin_role_name": "超级管理员", "menu_permissions": allAdminPermissionKeys()}, nil
	}
	if roleID < 1 {
		return M{"admin_role_id": int64(0), "admin_role_name": "普通用户", "menu_permissions": []string{}}, nil
	}
	role, err := one(ctx, a.db, "SELECT name,status FROM "+a.t("imgo_admin_role")+" WHERE role_id=?", roleID)
	if errors.Is(err, sql.ErrNoRows) {
		return M{"admin_role_id": roleID, "admin_role_name": "普通用户", "menu_permissions": []string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	permissions := []string{}
	if number(role["status"]) == 1 {
		permissions, err = a.menuPermissions(ctx, user)
		if err != nil {
			return nil, err
		}
	}
	return M{"admin_role_id": roleID, "admin_role_name": str(role["name"]), "menu_permissions": permissions}, nil
}
