package server

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// These administrator actions are atomic: an order and its wallet ledger entries
// either commit together with the balance change, or none of them commit.
func (a *App) manageWalletRecharge(r *request, scope adminScope) (any, error) {
	uid, requestID := r.n("user_id"), r.s("request_id")
	principal, valid := parseWalletAmount(r.s("amount"))
	mode, value := r.s("bonus_mode"), strings.TrimSpace(r.s("bonus_value"))
	bonus, bonusValid := walletRechargeBonus(principal, mode, value)
	note := strings.TrimSpace(r.s("note"))
	if uid < 1 || !valid || !bonusValid || !walletRequestID.MatchString(requestID) || len(note) < 2 || len(note) > 500 {
		return nil, r.fail("用户、金额、赠送规则、请求编号或备注无效")
	}
	total := principal + bonus
	if total > 9000000000000000 {
		return nil, r.fail("充值总金额超出系统支持范围")
	}
	ctx := r.ctx()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if !scope.Global {
		if err := a.requireScopedUser(ctx, tx, scope, uid); err != nil {
			return nil, err
		}
	} else if _, err = one(ctx, tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO "+a.t("imgo_wallet")+" (user_id,available_cents,pending_cents,updated_at) VALUES (?,0,0,?)", uid, now); err != nil {
		return nil, err
	}
	wallet, err := one(ctx, tx, "SELECT available_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", uid)
	if err != nil {
		return nil, err
	}
	if !scope.Global {
		if err := a.requireScopedUser(ctx, tx, scope, uid); err != nil {
			return nil, err
		}
	}
	// Locking the wallet serializes retries even if they arrive concurrently.
	previous, err := one(ctx, tx, "SELECT order_id,amount_cents,bonus_mode,bonus_value,bonus_cents,total_cents,note,status FROM "+a.t("imgo_recharge_order")+" WHERE user_id=? AND request_id=?", uid, requestID)
	if err == nil {
		if number(previous["amount_cents"]) != principal || str(previous["bonus_mode"]) != mode || str(previous["bonus_value"]) != value || number(previous["bonus_cents"]) != bonus || str(previous["note"]) != note {
			return nil, clientError{"重复充值请求的内容不一致", 409}
		}
		return M{"order_id": previous["order_id"], "total_cents": previous["total_cents"], "bonus_cents": previous["bonus_cents"], "status": previous["status"]}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if number(wallet["available_cents"]) > 9000000000000000-total {
		return nil, clientError{"余额超出系统支持范围", 409}
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_recharge_order")+" (user_id,request_id,amount_cents,bonus_mode,bonus_value,bonus_cents,total_cents,note,status,created_at,created_by) VALUES (?,?,?,?,?,?,?,?,1,?,?)", uid, requestID, principal, mode, value, bonus, total, note, now, r.uid())
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	result, err = tx.ExecContext(ctx, "UPDATE "+a.t("imgo_wallet")+" SET available_cents=available_cents+?,updated_at=? WHERE user_id=?", total, now, uid)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, clientError{"充值入账失败", 409}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,'recharge',?,?,0,?,?,?,?)", uid, requestID, principal, id, r.uid(), note, now); err != nil {
		return nil, err
	}
	if bonus > 0 {
		if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,'recharge_bonus',?,?,0,?,?,?,?)", uid, requestID, bonus, id, r.uid(), note, now); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return M{"order_id": id, "total_cents": total, "bonus_cents": bonus, "status": 1}, nil
}

func (a *App) manageWalletWithdraw(r *request, scope adminScope) (any, error) {
	uid, requestID := r.n("user_id"), r.s("request_id")
	amount, valid := parseWalletAmount(r.s("amount"))
	note := strings.TrimSpace(r.s("note"))
	if uid < 1 || !valid || !walletRequestID.MatchString(requestID) || len(note) > 500 {
		return nil, r.fail("用户、提现金额、请求编号或备注无效")
	}
	ctx := r.ctx()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if !scope.Global {
		if err := a.requireScopedUser(ctx, tx, scope, uid); err != nil {
			return nil, err
		}
	} else if _, err = one(ctx, tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO "+a.t("imgo_wallet")+" (user_id,available_cents,pending_cents,updated_at) VALUES (?,0,0,?)", uid, now); err != nil {
		return nil, err
	}
	wallet, err := one(ctx, tx, "SELECT available_cents,pending_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", uid)
	if err != nil {
		return nil, err
	}
	if !scope.Global {
		if err := a.requireScopedUser(ctx, tx, scope, uid); err != nil {
			return nil, err
		}
	}
	previous, err := one(ctx, tx, "SELECT withdrawal_id,amount_cents,status,request_note FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? AND request_id=?", uid, requestID)
	if err == nil {
		if number(previous["amount_cents"]) != amount || str(previous["request_note"]) != note {
			return nil, clientError{"重复提现请求的内容不一致", 409}
		}
		return M{"withdrawal_id": previous["withdrawal_id"], "amount_cents": previous["amount_cents"], "status": previous["status"]}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if number(wallet["available_cents"]) < amount || number(wallet["pending_cents"]) > 9000000000000000-amount {
		return nil, clientError{"可提现余额不足或冻结余额超出系统支持范围", 409}
	}
	card, err := one(ctx, tx, "SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4 FROM "+a.t("imgo_bank_card")+" WHERE user_id=? AND status=1", uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, clientError{"该用户尚未绑定并通过审核的银行卡", 409}
	}
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, "UPDATE "+a.t("imgo_wallet")+" SET available_cents=available_cents-?,pending_cents=pending_cents+?,updated_at=? WHERE user_id=? AND available_cents>=?", amount, amount, now, uid, amount)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, clientError{"可提现余额不足，请刷新后重试", 409}
	}
	result, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_withdrawal")+" (user_id,request_id,amount_cents,status,receipt_name,bank_name,branch_name,account_cipher,account_last4,created_at,processed_at,processed_by,remark,source,created_by,request_note) VALUES (?,?,?,0,?,?,?,?,?,?,0,0,'','admin',?,?)", uid, requestID, amount, card["receipt_name"], card["bank_name"], card["branch_name"], card["account_cipher"], card["account_last4"], now, r.uid(), note)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,'withdraw',?,?,?,?,?,?,?)", uid, requestID, -amount, amount, id, r.uid(), note, now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return M{"withdrawal_id": id, "amount_cents": amount, "status": 0}, nil
}
