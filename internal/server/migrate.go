package server

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"
)

//go:embed schema.sql
var legacySchema string

// Migrate is an explicit operator action. It never runs on ordinary server startup.
func (a *App) Migrate(ctx context.Context, initialize bool) error {
	var count int
	if e := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", a.cfg.Prefix+"user").Scan(&count); e != nil {
		return e
	}
	if count == 0 {
		if !initialize {
			return errors.New("数据库尚未初始化，请先使用 -init")
		}
		schema := strings.ReplaceAll(legacySchema, "`yu_", "`"+a.cfg.Prefix)
		for _, q := range strings.Split(schema, ";") {
			if strings.TrimSpace(q) == "" {
				continue
			}
			if _, e := a.db.ExecContext(ctx, q); e != nil {
				return fmt.Errorf("初始化表结构失败: %w", e)
			}
		}
	} else if initialize {
		return errors.New("数据库已有 user 表，拒绝初始化；请使用 -migrate")
	}
	if _, e := a.db.ExecContext(ctx, "ALTER TABLE "+a.t("user")+" ROW_FORMAT=DYNAMIC, MODIFY password VARCHAR(255) NOT NULL, MODIFY last_login_ip VARCHAR(45) DEFAULT NULL, MODIFY register_ip VARCHAR(45) DEFAULT NULL"); e != nil {
		return e
	}
	if e := a.EnsureAddonSchema(ctx); e != nil {
		return e
	}
	indexes := []struct{ table, name, fields string }{{"message", "imgo_chat_msg", "chat_identify,msg_id"}, {"message", "imgo_unread", "to_user,is_group,is_read,status"}, {"message", "imgo_client_id", "id,from_user"}, {"message", "imgo_file", "file_id"}, {"group_user", "imgo_membership", "group_id,user_id,status"}, {"group_user", "imgo_user_groups", "user_id,status"}, {"friend", "imgo_friends", "create_user,friend_user_id,status"}, {"file", "imgo_file_owner", "user_id,status"}, {"file", "imgo_file_src", "src"}, {"emoji", "imgo_emoji_owner", "user_id,status"}}
	for _, idx := range indexes {
		if e := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?", a.cfg.Prefix+idx.table, idx.name).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			if _, e := a.db.ExecContext(ctx, "CREATE INDEX `"+idx.name+"` ON "+a.t(idx.table)+" ("+idx.fields+")"); e != nil {
				return e
			}
		}
	}
	for _, name := range []string{"sysInfo", "chatInfo", "fileUpload", "smtp", "compass"} {
		if _, err := one(ctx, a.db, "SELECT id FROM "+a.t("config")+" WHERE name=?", name); err == sql.ErrNoRows {
			if _, e := insert(ctx, a.db, a.t("config"), M{"name": name, "value": js(defaultConfig(name)), "status": 1, "create_time": time.Now().Unix()}); e != nil {
				return e
			}
		} else if err != nil {
			return err
		}
	}
	return a.migrateLegacyMedia(ctx)
}
func (a *App) CreateAdmin(ctx context.Context, account, password string) error {
	if len(account) < 3 || len(account) > 32 {
		return errors.New("管理员账号长度须为 3–32 字节")
	}
	hash, e := hashPassword(password)
	if e != nil {
		return e
	}
	var n int
	if e = a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+a.t("user")).Scan(&n); e != nil {
		return e
	}
	if n != 0 {
		return errors.New("只允许在没有用户的数据库中创建初始管理员")
	}
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = insert(ctx, tx, a.t("user"), M{"user_id": 1, "account": account, "realname": "管理员", "password": hash, "salt": "", "role": 1, "status": 1, "create_time": time.Now().Unix(), "setting": "{}"})
	if e != nil {
		return e
	}
	if _, e = a.ensureInviteCode(ctx, tx, 1); e != nil {
		return e
	}
	return tx.Commit()
}
func (a *App) CheckSchema(ctx context.Context) error {
	var dataType string
	var length sql.NullInt64
	if e := a.db.QueryRowContext(ctx, "SELECT DATA_TYPE,CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name='password'", a.cfg.Prefix+"user").Scan(&dataType, &length); e != nil {
		return errors.New("未找到用户表，请先初始化或迁移数据库")
	}
	if length.Int64 < 60 {
		return errors.New("密码列不足以存储 bcrypt，请先备份测试库并运行 -migrate")
	}
	for _, name := range []string{"imgo_session", "imgo_chat_lock", "imgo_object", "imgo_online_sample", "imgo_bank_card", "imgo_check_in", "imgo_wallet", "imgo_withdrawal", "imgo_wallet_entry", "imgo_recharge_order", "imgo_referral", "imgo_referral_legacy_code", "imgo_referral_path"} {
		var n int
		if e := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", a.cfg.Prefix+name).Scan(&n); e != nil || n == 0 {
			return errors.New("缺少 Go 服务附加表，请运行 -migrate")
		}
	}
	for _, column := range []string{"bank_name", "branch_name"} {
		var n int
		if e := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", a.cfg.Prefix+"imgo_bank_card", column).Scan(&n); e != nil || n == 0 {
			return errors.New("绑卡表缺少银行或支行字段，请运行 -migrate")
		}
	}
	return nil
}
