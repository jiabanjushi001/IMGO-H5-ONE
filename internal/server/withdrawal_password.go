package server

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"
)

const (
	withdrawalPasswordMaxAttempts = 5
	withdrawalPasswordLockSeconds = int64(5 * 60)
)

var withdrawalPasswordPattern = regexp.MustCompile(`^[0-9]{6}$`)

func validWithdrawalPassword(password string) bool {
	return withdrawalPasswordPattern.MatchString(password)
}

func (a *App) withdrawalPasswordSet(ctx context.Context, db DB, userID int64) (bool, error) {
	_, err := one(ctx, db, "SELECT user_id FROM "+a.t("imgo_withdrawal_password")+" WHERE user_id=?", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (a *App) setWithdrawalPassword(r *request) (any, error) {
	password := r.s("password")
	if !validWithdrawalPassword(password) {
		return nil, clientError{"提现密码必须是6位数字", 400}
	}
	if password != r.s("password_confirmation") {
		return nil, clientError{"两次输入的提现密码不一致", 400}
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	result, err := a.db.ExecContext(r.ctx(), "INSERT IGNORE INTO "+a.t("imgo_withdrawal_password")+" (user_id,password_hash,failed_attempts,locked_until,updated_at) VALUES (?,?,0,0,?)", r.uid(), hash, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, clientError{"提现密码已设置，请勿重复设置", 409}
	}
	return M{"withdrawal_password_set": true}, nil
}

func (a *App) verifyWithdrawalPassword(ctx context.Context, db DB, userID int64, password string) error {
	row, err := one(ctx, db, "SELECT password_hash,failed_attempts,locked_until FROM "+a.t("imgo_withdrawal_password")+" WHERE user_id=?", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return clientError{"请先设置6位提现密码", 409}
	}
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	if lockedUntil := number(row["locked_until"]); lockedUntil > now {
		return clientError{"提现密码错误次数过多，请5分钟后重试", 429}
	}
	if !validWithdrawalPassword(password) || !checkPassword(str(row["password_hash"]), "", password) {
		attempts := number(row["failed_attempts"]) + 1
		lockedUntil := int64(0)
		message := "提现密码错误"
		code := 400
		if attempts >= withdrawalPasswordMaxAttempts {
			attempts = 0
			lockedUntil = now + withdrawalPasswordLockSeconds
			message = "提现密码错误次数过多，请5分钟后重试"
			code = 429
		} else {
			message += "，还可尝试" + str(withdrawalPasswordMaxAttempts-attempts) + "次"
		}
		if _, updateErr := db.ExecContext(ctx, "UPDATE "+a.t("imgo_withdrawal_password")+" SET failed_attempts=?,locked_until=?,updated_at=? WHERE user_id=?", attempts, lockedUntil, now, userID); updateErr != nil {
			return updateErr
		}
		return clientError{message, code}
	}
	if number(row["failed_attempts"]) != 0 || number(row["locked_until"]) != 0 {
		if _, err = db.ExecContext(ctx, "UPDATE "+a.t("imgo_withdrawal_password")+" SET failed_attempts=0,locked_until=0,updated_at=? WHERE user_id=?", now, userID); err != nil {
			return err
		}
	}
	return nil
}
