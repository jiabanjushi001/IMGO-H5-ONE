package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// EnsureAddonSchema upgrades extension tables and existing invite codes without
// changing referral parent IDs or the recorded hierarchy.
// Legacy table rewrites and media migrations remain explicit -migrate operations.
func (a *App) EnsureAddonSchema(ctx context.Context) error {
	conn, err := a.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var database string
	if err = conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err != nil {
		return err
	}
	lockName := fmt.Sprintf("imgo-addon-%x", sha256.Sum256([]byte(database+":"+a.cfg.Prefix)))[:60]
	var acquired int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 60)", lockName).Scan(&acquired); err != nil {
		return err
	}
	if acquired != 1 {
		return fmt.Errorf("等待数据库升级锁超时")
	}
	defer func() {
		release, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(release, "SELECT RELEASE_LOCK(?)", lockName)
	}()
	var count int
	if err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", a.cfg.Prefix+"user").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("数据库尚未初始化，请先使用 -init")
	}
	return a.ensureAddonTables(ctx, conn)
}

func (a *App) ensureAddonTables(ctx context.Context, db *sql.Conn) error {
	var count int
	for _, q := range a.addonTableDDL() {
		table := strings.Split(q, "`")[1]
		if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", table).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			if _, e := db.ExecContext(ctx, q); e != nil {
				return fmt.Errorf("自动创建附加表 %s 失败: %w", table, e)
			}
		}
	}
	if err := a.upgradeInviteCodes(ctx, db); err != nil {
		return err
	}
	users, err := rows(ctx, db, "SELECT u.user_id FROM "+a.t("user")+" u LEFT JOIN "+a.t("imgo_referral")+" r ON r.user_id=u.user_id WHERE r.user_id IS NULL")
	if err != nil {
		return err
	}
	for _, user := range users {
		if _, err := a.ensureInviteCode(ctx, db, number(user["user_id"])); err != nil {
			return fmt.Errorf("补齐用户邀请码失败: %w", err)
		}
	}
	for _, column := range []string{"bank_name", "branch_name"} {
		if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", a.cfg.Prefix+"imgo_bank_card", column).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			if _, e := db.ExecContext(ctx, "ALTER TABLE "+a.t("imgo_bank_card")+" ADD COLUMN `"+column+"` VARCHAR(120) NOT NULL DEFAULT ''"); e != nil {
				return e
			}
		}
	}
	for _, column := range []struct{ name, sqlType string }{{"source", "VARCHAR(16) NOT NULL DEFAULT 'user'"}, {"created_by", "INT NOT NULL DEFAULT 0"}, {"request_note", "VARCHAR(500) NOT NULL DEFAULT ''"}} {
		if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", a.cfg.Prefix+"imgo_withdrawal", column.name).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			if _, e := db.ExecContext(ctx, "ALTER TABLE "+a.t("imgo_withdrawal")+" ADD COLUMN `"+column.name+"` "+column.sqlType); e != nil {
				return e
			}
		}
	}
	for _, column := range []struct{ name, sqlType string }{{"last_chat_time", "BIGINT UNSIGNED NOT NULL DEFAULT 0"}, {"last_chat_ip", "VARCHAR(45) DEFAULT NULL"}, {"admin_role_id", "BIGINT NOT NULL DEFAULT 0"}} {
		if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", a.cfg.Prefix+"user", column.name).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			if _, e := db.ExecContext(ctx, "ALTER TABLE "+a.t("user")+" ADD COLUMN `"+column.name+"` "+column.sqlType); e != nil {
				return e
			}
		}
	}
	for _, column := range []struct{ name, sqlType string }{{"agent_mode", "TINYINT NOT NULL DEFAULT 0"}, {"role_code", "VARCHAR(32) NULL"}} {
		if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", a.cfg.Prefix+"imgo_admin_role", column.name).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			if _, e := db.ExecContext(ctx, "ALTER TABLE "+a.t("imgo_admin_role")+" ADD COLUMN `"+column.name+"` "+column.sqlType); e != nil {
				return e
			}
		}
	}
	if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=? AND non_unique=0", a.cfg.Prefix+"imgo_admin_role", "imgo_admin_role_code").Scan(&count); e != nil {
		return e
	}
	if count == 0 {
		if _, e := db.ExecContext(ctx, "CREATE UNIQUE INDEX `imgo_admin_role_code` ON "+a.t("imgo_admin_role")+" (`role_code`)"); e != nil {
			return e
		}
	}
	if err := a.seedAdminPermissions(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := a.seedMentorRole(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *App) addonTableDDL() []string {
	return []string{
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_online_sample") + " (sample_at BIGINT PRIMARY KEY, users INT NOT NULL, devices INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_object") + " (file_id INT PRIMARY KEY,disk VARCHAR(16) NOT NULL,object_key VARCHAR(191) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_session") + " (sid VARCHAR(64) PRIMARY KEY,user_id INT NOT NULL,expires_at BIGINT NOT NULL,INDEX(user_id),INDEX(expires_at)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_chat_lock") + " (chat_identify VARCHAR(64) PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_bank_card") + " (user_id INT PRIMARY KEY,receipt_name VARCHAR(100) NOT NULL,bank_name VARCHAR(120) NOT NULL DEFAULT '',branch_name VARCHAR(120) NOT NULL DEFAULT '',account_cipher VARCHAR(255) NOT NULL,account_last4 CHAR(4) NOT NULL,status TINYINT NOT NULL DEFAULT 0,remark VARCHAR(500) NOT NULL DEFAULT '',version INT NOT NULL DEFAULT 1,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,INDEX(status,updated_at)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_check_in") + " (user_id INT NOT NULL,sign_date DATE NOT NULL,created_at BIGINT NOT NULL,PRIMARY KEY(user_id,sign_date)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_wallet") + " (user_id INT PRIMARY KEY,available_cents BIGINT NOT NULL DEFAULT 0,pending_cents BIGINT NOT NULL DEFAULT 0,updated_at BIGINT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_withdrawal") + " (withdrawal_id BIGINT AUTO_INCREMENT PRIMARY KEY,user_id INT NOT NULL,request_id VARCHAR(80) NOT NULL,amount_cents BIGINT NOT NULL,status TINYINT NOT NULL DEFAULT 0,receipt_name VARCHAR(100) NOT NULL,bank_name VARCHAR(120) NOT NULL,branch_name VARCHAR(120) NOT NULL,account_cipher VARCHAR(255) NOT NULL,account_last4 CHAR(4) NOT NULL,created_at BIGINT NOT NULL,processed_at BIGINT NOT NULL DEFAULT 0,processed_by INT NOT NULL DEFAULT 0,remark VARCHAR(500) NOT NULL DEFAULT '',UNIQUE KEY imgo_withdraw_request(user_id,request_id),INDEX imgo_withdraw_status(status,created_at),INDEX imgo_withdraw_user(user_id,withdrawal_id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_wallet_entry") + " (entry_id BIGINT AUTO_INCREMENT PRIMARY KEY,user_id INT NOT NULL,event VARCHAR(20) NOT NULL,request_id VARCHAR(80) NOT NULL,available_delta BIGINT NOT NULL,pending_delta BIGINT NOT NULL,reference_id BIGINT NOT NULL DEFAULT 0,actor_id INT NOT NULL,note VARCHAR(500) NOT NULL DEFAULT '',created_at BIGINT NOT NULL,UNIQUE KEY imgo_wallet_event(user_id,event,request_id),INDEX imgo_wallet_user(user_id,entry_id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_recharge_order") + " (order_id BIGINT AUTO_INCREMENT PRIMARY KEY,user_id INT NOT NULL,request_id VARCHAR(80) NOT NULL,amount_cents BIGINT NOT NULL,bonus_mode VARCHAR(10) NOT NULL DEFAULT 'none',bonus_value VARCHAR(20) NOT NULL DEFAULT '',bonus_cents BIGINT NOT NULL DEFAULT 0,total_cents BIGINT NOT NULL,note VARCHAR(500) NOT NULL,status TINYINT NOT NULL DEFAULT 1,created_at BIGINT NOT NULL,created_by INT NOT NULL,UNIQUE KEY imgo_recharge_request(user_id,request_id),INDEX imgo_recharge_user(user_id,order_id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_referral") + " (user_id INT PRIMARY KEY,invite_code CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,parent_user_id INT NOT NULL DEFAULT 0,created_at BIGINT NOT NULL,UNIQUE KEY imgo_referral_code(invite_code),INDEX imgo_referral_parent(parent_user_id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_referral_legacy_code") + " (invite_code CHAR(5) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,user_id INT NOT NULL,INDEX imgo_referral_legacy_user(user_id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_referral_path") + " (ancestor_user_id INT NOT NULL,descendant_user_id INT NOT NULL,depth INT NOT NULL,PRIMARY KEY(ancestor_user_id,descendant_user_id),INDEX imgo_referral_descendant(descendant_user_id,depth)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_admin_role") + " (role_id BIGINT AUTO_INCREMENT PRIMARY KEY,name VARCHAR(64) NOT NULL,remark VARCHAR(255) NOT NULL DEFAULT '',status TINYINT NOT NULL DEFAULT 1,agent_mode TINYINT NOT NULL DEFAULT 0,role_code VARCHAR(32) NULL,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,UNIQUE KEY imgo_admin_role_name(name),UNIQUE KEY imgo_admin_role_code(role_code),INDEX imgo_admin_role_status(status)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_admin_permission") + " (permission_id BIGINT AUTO_INCREMENT PRIMARY KEY,permission_key VARCHAR(64) NOT NULL,name VARCHAR(64) NOT NULL,menu_path VARCHAR(128) NOT NULL DEFAULT '',sort INT NOT NULL DEFAULT 0,UNIQUE KEY imgo_admin_permission_key(permission_key)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_admin_role_permission") + " (role_id BIGINT NOT NULL,permission_id BIGINT NOT NULL,PRIMARY KEY(role_id,permission_id),INDEX imgo_admin_permission_role(permission_id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_agent_setting") + " (agent_user_id BIGINT PRIMARY KEY,auto_add_user JSON NULL,auto_add_group JSON NULL,updated_by BIGINT NOT NULL,updated_at BIGINT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_agent_auto_state") + " (agent_user_id BIGINT PRIMARY KEY,last_customer_user_id BIGINT NOT NULL DEFAULT 0,group_id BIGINT NOT NULL DEFAULT 0,group_num INT NOT NULL DEFAULT 1,updated_at BIGINT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS " + a.t("imgo_agent_online_sample") + " (agent_user_id BIGINT NOT NULL,sample_at BIGINT NOT NULL,users INT NOT NULL,devices INT NOT NULL,PRIMARY KEY(agent_user_id,sample_at),INDEX(sample_at)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	}
}

func (a *App) seedAdminPermissions(ctx context.Context, db DB) error {
	query := "INSERT INTO " + a.t("imgo_admin_permission") + " (permission_key,name,menu_path,sort) VALUES (?,?,?,?) " +
		"ON DUPLICATE KEY UPDATE name=VALUES(name),menu_path=VALUES(menu_path),sort=VALUES(sort)"
	for _, permission := range adminPermissionCatalog {
		if _, err := db.ExecContext(ctx, query, permission.Key, permission.Name, permission.MenuPath, permission.Sort); err != nil {
			return fmt.Errorf("补齐后台权限 %s 失败: %w", permission.Key, err)
		}
	}
	return nil
}

func (a *App) seedMentorRole(ctx context.Context, db DB) error {
	_, err := one(ctx, db, "SELECT role_id FROM "+a.t("imgo_admin_role")+" WHERE role_code=?", "mentor")
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("查询导师角色失败: %w", err)
	}
	name := "导师专员"
	for suffix := 0; ; suffix++ {
		if suffix == 1 {
			name = "导师专员（预置）"
		} else if suffix > 1 {
			name = fmt.Sprintf("导师专员（预置 %d）", suffix)
		}
		_, err = one(ctx, db, "SELECT role_id FROM "+a.t("imgo_admin_role")+" WHERE name=?", name)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return fmt.Errorf("查询导师角色名称失败: %w", err)
		}
	}
	now := time.Now().Unix()
	result, err := db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_admin_role")+" (name,remark,status,agent_mode,role_code,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", name, "", 1, 1, "mentor", now, now)
	if err != nil {
		return fmt.Errorf("创建导师角色失败: %w", err)
	}
	roleID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	permissionKeys := []string{"manage.overview", "manage.users", "manage.messages", "manage.groups", "manage.files", "manage.bank", "manage.finance"}
	args := make([]any, 0, len(permissionKeys)+1)
	args = append(args, roleID)
	for _, key := range permissionKeys {
		args = append(args, key)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(permissionKeys)), ",")
	_, err = db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_admin_role_permission")+" (role_id,permission_id) SELECT ?,permission_id FROM "+a.t("imgo_admin_permission")+" WHERE permission_key IN ("+placeholders+")", args...)
	if err != nil {
		return fmt.Errorf("写入导师角色权限失败: %w", err)
	}
	return nil
}

func (a *App) upgradeInviteCodes(ctx context.Context, db *sql.Conn) error {
	var length int64
	if err := db.QueryRowContext(ctx, "SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name='invite_code'", a.cfg.Prefix+"imgo_referral").Scan(&length); err != nil {
		return err
	}
	if length < 6 {
		if _, err := db.ExecContext(ctx, "ALTER TABLE "+a.t("imgo_referral")+" MODIFY invite_code CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"); err != nil {
			return fmt.Errorf("扩展邀请码字段失败: %w", err)
		}
	}
	referrals, err := rows(ctx, db, "SELECT user_id,invite_code FROM "+a.t("imgo_referral"))
	if err != nil {
		return err
	}
	for _, referral := range referrals {
		oldCode := str(referral["invite_code"])
		if validInviteCode(oldCode) {
			continue
		}
		if !validLegacyInviteCode(oldCode) {
			return fmt.Errorf("用户 %d 的旧邀请码格式无效", number(referral["user_id"]))
		}
		userID := number(referral["user_id"])
		if _, err := db.ExecContext(ctx, "INSERT IGNORE INTO "+a.t("imgo_referral_legacy_code")+" (invite_code,user_id) VALUES (?,?)", oldCode, userID); err != nil {
			return fmt.Errorf("保存旧邀请码映射失败: %w", err)
		}
		alias, err := one(ctx, db, "SELECT user_id FROM "+a.t("imgo_referral_legacy_code")+" WHERE invite_code=?", oldCode)
		if err != nil || number(alias["user_id"]) != userID {
			return fmt.Errorf("旧邀请码 %s 的归属冲突: %v", oldCode, err)
		}
		converted := false
		for attempt := 0; attempt < 64; attempt++ {
			code, err := newInviteCode()
			if err != nil {
				return err
			}
			result, err := db.ExecContext(ctx, "UPDATE "+a.t("imgo_referral")+" SET invite_code=? WHERE user_id=? AND invite_code=?", code, userID, oldCode)
			if err != nil {
				var duplicate *mysql.MySQLError
				if errors.As(err, &duplicate) && duplicate.Number == 1062 {
					continue
				}
				return fmt.Errorf("转换用户 %d 的邀请码失败: %w", userID, err)
			}
			affected, err := result.RowsAffected()
			if err != nil || affected != 1 {
				return fmt.Errorf("转换用户 %d 的邀请码时记录发生变化: %v", userID, err)
			}
			converted = true
			break
		}
		if !converted {
			return fmt.Errorf("用户 %d 无法生成唯一的数字邀请码", userID)
		}
	}
	return nil
}
