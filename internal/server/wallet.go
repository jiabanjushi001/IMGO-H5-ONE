package server

import (
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var walletRequestID = regexp.MustCompile(`^[a-zA-Z0-9_-]{10,80}$`)

// Store money as integer cents; never round a floating-point request value.
func parseWalletAmount(value string) (int64, bool) {
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 2 || len(parts[0]) < 1 || len(parts[0]) > 9 {
		return 0, false
	}
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	cents := yuan * 100
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, false
		}
		for _, ch := range parts[1] {
			if ch < '0' || ch > '9' {
				return 0, false
			}
		}
		fraction, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, false
		}
		if len(parts[1]) == 1 {
			fraction *= 10
		}
		cents += fraction
	}
	return cents, cents > 0
}

func (a *App) walletBalance(r *request, userID int64) (M, error) {
	wallet, err := r.one("SELECT available_cents,pending_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=?", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return M{"available_cents": int64(0), "pending_cents": int64(0), "currency": "CNY"}, nil
	}
	if err != nil {
		return nil, err
	}
	return M{"available_cents": number(wallet["available_cents"]), "pending_cents": number(wallet["pending_cents"]), "currency": "CNY"}, nil
}

func (a *App) walletStatus(r *request) (M, error) { return a.walletBalance(r, r.uid()) }

func (a *App) previousWithdrawal(r *request, requestID string, cents int64) (M, error) {
	previous, err := r.one("SELECT withdrawal_id,amount_cents,status FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? AND request_id=?", r.uid(), requestID)
	if err != nil {
		return nil, err
	}
	if number(previous["amount_cents"]) != cents {
		return nil, clientError{"重复请求的金额不一致", 409}
	}
	return M{"withdrawal_id": previous["withdrawal_id"], "status": previous["status"], "amount_cents": previous["amount_cents"]}, nil
}

func (a *App) walletWithdraw(r *request) (any, error) {
	cents, valid := parseWalletAmount(r.s("amount"))
	requestID := r.s("request_id")
	if !valid || !walletRequestID.MatchString(requestID) {
		return nil, r.fail("提现金额或请求编号无效")
	}
	if previous, err := a.previousWithdrawal(r, requestID, cents); err == nil {
		return previous, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	ctx := r.ctx()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO "+a.t("imgo_wallet")+" (user_id,available_cents,pending_cents,updated_at) VALUES (?,0,0,?)", r.uid(), now); err != nil {
		return nil, err
	}
	wallet, err := one(ctx, tx, "SELECT available_cents,pending_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", r.uid())
	if err != nil {
		return nil, err
	}
	// A concurrent retry may have committed while this transaction waited for
	// the wallet lock. Resolve it before checking the newly reduced balance.
	previous, err := one(ctx, tx, "SELECT withdrawal_id,amount_cents,status FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? AND request_id=? FOR UPDATE", r.uid(), requestID)
	if err == nil {
		if number(previous["amount_cents"]) != cents {
			return nil, clientError{"重复请求的金额不一致", 409}
		}
		return M{"withdrawal_id": previous["withdrawal_id"], "status": previous["status"], "amount_cents": previous["amount_cents"]}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if number(wallet["available_cents"]) < cents {
		return nil, clientError{"可提现余额不足", 409}
	}
	card, err := one(ctx, tx, "SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4 FROM "+a.t("imgo_bank_card")+" WHERE user_id=? AND status=1", r.uid())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, clientError{"请先绑定并通过审核的银行卡", 409}
	}
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, "UPDATE "+a.t("imgo_wallet")+" SET available_cents=available_cents-?,pending_cents=pending_cents+?,updated_at=? WHERE user_id=? AND available_cents>=?", cents, cents, now, r.uid(), cents)
	if err != nil {
		return nil, err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return nil, clientError{"可提现余额不足，请刷新后重试", 409}
	}
	result, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_withdrawal")+" (user_id,request_id,amount_cents,status,receipt_name,bank_name,branch_name,account_cipher,account_last4,created_at,processed_at,processed_by,remark) VALUES (?,?,?,0,?,?,?,?,?,?,0,0,'')", r.uid(), requestID, cents, card["receipt_name"], card["bank_name"], card["branch_name"], card["account_cipher"], card["account_last4"], now)
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			_ = tx.Rollback()
			return a.previousWithdrawal(r, requestID, cents)
		}
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,?,?,?,?,?,?,?,?)", r.uid(), "withdraw", requestID, -cents, cents, id, r.uid(), "", now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return M{"withdrawal_id": id, "status": 0, "amount_cents": cents}, nil
}

func (a *App) wallet(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "status":
		return a.walletStatus(r)
	case "history":
		n, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_withdrawal")+" WHERE user_id=?", r.uid())
		if err != nil {
			return nil, err
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		return r.list("SELECT withdrawal_id,amount_cents,status,bank_name,account_last4,created_at,processed_at,remark FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? ORDER BY withdrawal_id DESC LIMIT ? OFFSET ?", r.uid(), limit, offset)
	case "entries":
		return a.walletEntries(r, r.uid())
	case "withdraw":
		return a.walletWithdraw(r)
	default:
		return nil, r.fail("未知操作")
	}
}
