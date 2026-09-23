package server

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

func (a *App) manageRole(r *request) (any, error) {
	switch action(r) {
	case "permissions":
		result := make([]M, 0, len(adminPermissionCatalog))
		for _, permission := range adminPermissionCatalog {
			result = append(result, M{
				"permission_key": permission.Key,
				"name":           permission.Name,
				"menu_path":      permission.MenuPath,
				"sort":           permission.Sort,
			})
		}
		return result, nil
	case "index":
		list, err := r.list("SELECT r.role_id,r.name,r.remark,r.status,r.created_at,r.updated_at," +
			"COUNT(DISTINCT u.user_id) user_count,GROUP_CONCAT(DISTINCT p.permission_key ORDER BY p.sort SEPARATOR ',') permissions " +
			"FROM " + a.t("imgo_admin_role") + " r " +
			"LEFT JOIN " + a.t("user") + " u ON u.admin_role_id=r.role_id AND u.delete_time=0 " +
			"LEFT JOIN " + a.t("imgo_admin_role_permission") + " rp ON rp.role_id=r.role_id " +
			"LEFT JOIN " + a.t("imgo_admin_permission") + " p ON p.permission_id=rp.permission_id " +
			"GROUP BY r.role_id,r.name,r.remark,r.status,r.created_at,r.updated_at ORDER BY r.role_id")
		if err != nil {
			return nil, err
		}
		for _, role := range list {
			role["permissions"] = splitPermissionKeys(str(role["permissions"]))
		}
		counts, err := r.one("SELECT SUM(CASE WHEN user_id=1 THEN 1 ELSE 0 END) super_count," +
			"SUM(CASE WHEN user_id<>1 AND COALESCE(admin_role_id,0)=0 THEN 1 ELSE 0 END) ordinary_count " +
			"FROM " + a.t("user") + " WHERE delete_time=0")
		if err != nil {
			return nil, err
		}
		builtins := []M{
			{"role_id": int64(-1), "name": "超级管理员", "remark": "拥有后台全部权限", "status": int64(1), "user_count": number(counts["super_count"]), "permissions": allAdminPermissionKeys(), "builtin": true},
			{"role_id": int64(0), "name": "普通用户", "remark": "仅使用聊天功能", "status": int64(1), "user_count": number(counts["ordinary_count"]), "permissions": []string{}, "builtin": true},
		}
		return append(builtins, list...), nil
	case "detail":
		role, err := r.one("SELECT role_id,name,remark,status,created_at,updated_at FROM "+a.t("imgo_admin_role")+" WHERE role_id=?", r.n("role_id"))
		if err != nil {
			return nil, err
		}
		permissions, err := rows(r.ctx(), a.db, "SELECT p.permission_key FROM "+a.t("imgo_admin_role_permission")+" rp JOIN "+a.t("imgo_admin_permission")+" p ON p.permission_id=rp.permission_id WHERE rp.role_id=? ORDER BY p.sort,p.permission_id", r.n("role_id"))
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(permissions))
		for _, permission := range permissions {
			keys = append(keys, str(permission["permission_key"]))
		}
		role["permissions"] = keys
		return role, nil
	case "save":
		return a.saveAdminRole(r)
	case "setstatus":
		status := r.n("status")
		if status != 0 && status != 1 {
			return nil, r.fail("角色状态无效")
		}
		result, err := a.db.ExecContext(r.ctx(), "UPDATE "+a.t("imgo_admin_role")+" SET status=?,updated_at=? WHERE role_id=?", status, time.Now().Unix(), r.n("role_id"))
		if err != nil {
			return nil, err
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil, sql.ErrNoRows
		}
		return nil, nil
	case "del":
		return nil, a.deleteAdminRole(r)
	}
	return nil, r.fail("未知操作")
}

func (a *App) attachAdminRoleNames(r *request, users []M) error {
	roleIDs := map[int64]bool{}
	for _, user := range users {
		if number(user["user_id"]) == 1 {
			user["admin_role_name"] = "超级管理员"
			continue
		}
		roleID := number(user["admin_role_id"])
		if roleID < 1 {
			user["admin_role_name"] = "普通用户"
			continue
		}
		user["admin_role_name"] = "角色已删除"
		roleIDs[roleID] = true
	}
	if len(roleIDs) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(roleIDs))
	for roleID := range roleIDs {
		ids = append(ids, roleID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	args := make([]any, len(ids))
	for i, roleID := range ids {
		args[i] = roleID
	}
	roles, err := rows(r.ctx(), a.db, "SELECT role_id,name FROM "+a.t("imgo_admin_role")+" WHERE role_id IN ("+marks(len(ids))+")", args...)
	if err != nil {
		return err
	}
	names := map[int64]string{}
	for _, role := range roles {
		names[number(role["role_id"])] = str(role["name"])
	}
	for _, user := range users {
		if name := names[number(user["admin_role_id"])]; name != "" {
			user["admin_role_name"] = name
		}
	}
	return nil
}

func (a *App) saveAdminRole(r *request) (any, error) {
	name := strings.TrimSpace(r.s("name"))
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return nil, r.fail("角色名称须为1-64个字")
	}
	remark := strings.TrimSpace(r.s("remark"))
	if utf8.RuneCountInString(remark) > 255 {
		return nil, r.fail("角色备注最多255个字")
	}
	status := r.n("status")
	if status != 0 && status != 1 {
		return nil, r.fail("角色状态无效")
	}
	permissions, err := requestedPermissionKeys(r.p["permissions"])
	if err != nil {
		return nil, err
	}
	tx, err := a.db.BeginTx(r.ctx(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	roleID := r.n("role_id")
	now := time.Now().Unix()
	if roleID == 0 {
		result, execErr := tx.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_admin_role")+" (name,remark,status,created_at,updated_at) VALUES (?,?,?,?,?)", name, remark, status, now, now)
		if execErr != nil {
			return nil, roleWriteError(execErr)
		}
		roleID, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
	} else {
		result, execErr := tx.ExecContext(r.ctx(), "UPDATE "+a.t("imgo_admin_role")+" SET name=?,remark=?,status=?,updated_at=? WHERE role_id=?", name, remark, status, now, roleID)
		if execErr != nil {
			return nil, roleWriteError(execErr)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil, sql.ErrNoRows
		}
		if _, err = tx.ExecContext(r.ctx(), "DELETE FROM "+a.t("imgo_admin_role_permission")+" WHERE role_id=?", roleID); err != nil {
			return nil, err
		}
	}
	for _, permission := range permissions {
		result, execErr := tx.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_admin_role_permission")+" (role_id,permission_id) SELECT ?,permission_id FROM "+a.t("imgo_admin_permission")+" WHERE permission_key=?", roleID, permission)
		if execErr != nil {
			return nil, execErr
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return nil, r.fail("菜单权限无效")
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return M{"role_id": roleID}, nil
}

func (a *App) deleteAdminRole(r *request) error {
	roleID := r.n("role_id")
	if roleID < 1 {
		return r.fail("角色ID无效")
	}
	tx, err := a.db.BeginTx(r.ctx(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int64
	if err = tx.QueryRowContext(r.ctx(), "SELECT COUNT(*) FROM "+a.t("user")+" WHERE admin_role_id=? AND delete_time=0", roleID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return clientError{"该角色已绑定用户，请先转移用户", 409}
	}
	if _, err = tx.ExecContext(r.ctx(), "DELETE FROM "+a.t("imgo_admin_role_permission")+" WHERE role_id=?", roleID); err != nil {
		return err
	}
	result, err := tx.ExecContext(r.ctx(), "DELETE FROM "+a.t("imgo_admin_role")+" WHERE role_id=?", roleID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func requestedPermissionKeys(value any) ([]string, error) {
	requested := []string{}
	switch permissions := value.(type) {
	case nil:
	case []any:
		for _, permission := range permissions {
			requested = append(requested, str(permission))
		}
	case []string:
		requested = append(requested, permissions...)
	case string:
		requested = splitPermissionKeys(permissions)
	default:
		return nil, clientError{"菜单权限格式无效", 400}
	}
	allowed := map[string]bool{}
	for _, permission := range adminPermissionCatalog {
		allowed[permission.Key] = true
	}
	unique := make([]string, 0, len(requested))
	seen := map[string]bool{}
	for _, permission := range requested {
		permission = strings.TrimSpace(permission)
		if !allowed[permission] {
			return nil, clientError{"菜单权限无效", 400}
		}
		if !seen[permission] {
			seen[permission] = true
			unique = append(unique, permission)
		}
	}
	return unique, nil
}

func splitPermissionKeys(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func roleWriteError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return clientError{"角色名称已存在", 400}
	}
	return err
}
