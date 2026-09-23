package server

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func (a *App) manageWalletAccount(r *request) (any, error) {
	uid := r.n("user_id")
	if uid < 1 {
		return nil, r.fail("用户ID无效")
	}
	user, err := r.one("SELECT user_id,account,realname FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid)
	if err != nil {
		return nil, err
	}
	balance, err := a.walletBalance(r, uid)
	if err != nil {
		return nil, err
	}
	for key, value := range balance {
		user[key] = value
	}
	return user, nil
}

func (a *App) priorWalletCredit(r *request, userID int64, requestID string, cents int64) (M, error) {
	entry, err := r.one("SELECT entry_id,available_delta FROM "+a.t("imgo_wallet_entry")+" WHERE user_id=? AND event='credit' AND request_id=?", userID, requestID)
	if err != nil {
		return nil, err
	}
	if number(entry["available_delta"]) != cents {
		return nil, clientError{"重复入账请求的金额不一致", 409}
	}
	return M{"credited": true, "entry_id": entry["entry_id"]}, nil
}

func (a *App) manageWalletCredit(r *request) (any, error) {
	uid := r.n("user_id")
	cents, valid := parseWalletAmount(r.s("amount"))
	requestID := r.s("request_id")
	note := strings.TrimSpace(r.s("note"))
	if uid < 1 || !valid || !walletRequestID.MatchString(requestID) || len(note) < 2 || len(note) > 500 {
		return nil, r.fail("用户、金额、请求编号或入账说明无效")
	}
	if previous, err := a.priorWalletCredit(r, uid, requestID, cents); err == nil {
		return previous, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if _, err := r.one("SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid); err != nil {
		return nil, err
	}
	ctx := r.ctx()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO "+a.t("imgo_wallet")+" (user_id,available_cents,pending_cents,updated_at) VALUES (?,0,0,?)", uid, now); err != nil {
		return nil, err
	}
	wallet, err := one(ctx, tx, "SELECT available_cents,pending_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", uid)
	if err != nil {
		return nil, err
	}
	if number(wallet["available_cents"]) > 9000000000000000-cents {
		return nil, clientError{"余额超出系统支持范围", 409}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE "+a.t("imgo_wallet")+" SET available_cents=available_cents+?,updated_at=? WHERE user_id=?", cents, now, uid); err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,'credit',?,?,0,0,?,?,?)", uid, requestID, cents, r.uid(), note, now)
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			_ = tx.Rollback()
			return a.priorWalletCredit(r, uid, requestID, cents)
		}
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return M{"credited": true, "entry_id": id}, nil
}

func (a *App) manageWalletReview(r *request) (any, error) {
	id := r.n("withdrawal_id")
	status, valid := bankStatus(r.p["status"])
	remark := strings.TrimSpace(r.s("remark"))
	if id < 1 || !valid || status == 0 || len(remark) > 500 || (status == 2 && len(remark) < 2) {
		return nil, r.fail("提现处理参数无效，拒绝时须填写原因")
	}
	ctx := r.ctx()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	withdrawal, err := one(ctx, tx, "SELECT user_id,amount_cents,status FROM "+a.t("imgo_withdrawal")+" WHERE withdrawal_id=? FOR UPDATE", id)
	if err != nil {
		return nil, err
	}
	if current := number(withdrawal["status"]); current != 0 {
		if current == status {
			return M{"processed": true, "status": current}, nil
		}
		return nil, clientError{"该提现申请已经处理", 409}
	}
	uid, cents := number(withdrawal["user_id"]), number(withdrawal["amount_cents"])
	wallet, err := one(ctx, tx, "SELECT available_cents,pending_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", uid)
	if err != nil {
		return nil, err
	}
	if number(wallet["pending_cents"]) < cents {
		return nil, clientError{"待处理余额异常，已阻止操作", 409}
	}
	availableDelta, event := int64(0), "paid"
	if status == 2 {
		availableDelta, event = cents, "refund"
	}
	now := time.Now().Unix()
	result, err := tx.ExecContext(ctx, "UPDATE "+a.t("imgo_wallet")+" SET available_cents=available_cents+?,pending_cents=pending_cents-?,updated_at=? WHERE user_id=? AND pending_cents>=?", availableDelta, cents, now, uid, cents)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, clientError{"待处理余额异常，已阻止操作", 409}
	}
	result, err = tx.ExecContext(ctx, "UPDATE "+a.t("imgo_withdrawal")+" SET status=?,processed_at=?,processed_by=?,remark=? WHERE withdrawal_id=? AND status=0", status, now, r.uid(), remark, id)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, clientError{"该提现申请已经处理", 409}
	}
	requestID := "wd-" + strconv.FormatInt(id, 10)
	if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,?,?,?,?,?,?,?,?)", uid, event, requestID, availableDelta, -cents, id, r.uid(), remark, now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return M{"processed": true, "status": status}, nil
}

func (a *App) manageWallet(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "account":
		return a.manageWalletAccount(r)
	case "entries":
		if r.n("user_id") < 1 {
			return nil, r.fail("用户ID无效")
		}
		return a.walletEntries(r, r.n("user_id"))
	case "recharges":
		where, args := "1=1", []any{}
		if uid := r.n("user_id"); uid > 0 {
			where += " AND o.user_id=?"
			args = append(args, uid)
		}
		if keyword := strings.TrimSpace(r.s("keywords")); keyword != "" {
			where += " AND (u.account LIKE ? OR u.realname LIKE ?)"
			args = append(args, "%"+keyword+"%", "%"+keyword+"%")
		}
		from := a.t("imgo_recharge_order") + " o LEFT JOIN " + a.t("user") + " u ON u.user_id=o.user_id WHERE " + where
		n, err := r.one("SELECT COUNT(*) n FROM "+from, args...)
		if err != nil {
			return nil, err
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		return r.list("SELECT o.order_id,o.user_id,o.amount_cents,o.bonus_mode,o.bonus_value,o.bonus_cents,o.total_cents,o.note,o.status,o.created_at,o.created_by,u.account,u.realname FROM "+from+" ORDER BY o.order_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	case "recharge":
		return a.manageWalletRecharge(r)
	case "withdraw":
		return a.manageWalletWithdraw(r)
	case "credit":
		return a.manageWalletCredit(r)
	case "review":
		return a.manageWalletReview(r)
	case "index":
		where, args := "u.delete_time=0", []any{}
		if uid := r.n("user_id"); uid > 0 {
			where += " AND w.user_id=?"
			args = append(args, uid)
		}
		if _, provided := r.p["status"]; provided && r.s("status") != "" {
			status, valid := bankStatus(r.p["status"])
			if !valid {
				return nil, r.fail("状态无效")
			}
			where += " AND w.status=?"
			args = append(args, status)
		}
		if keyword := strings.TrimSpace(r.s("keywords")); keyword != "" {
			where += " AND (u.account LIKE ? OR u.realname LIKE ?)"
			args = append(args, "%"+keyword+"%", "%"+keyword+"%")
		}
		n, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_withdrawal")+" w JOIN "+a.t("user")+" u ON u.user_id=w.user_id WHERE "+where, args...)
		if err != nil {
			return nil, err
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		return r.list("SELECT w.withdrawal_id,w.user_id,w.amount_cents,w.status,w.bank_name,w.account_last4,w.created_at,w.processed_at,w.remark,u.account,u.realname FROM "+a.t("imgo_withdrawal")+" w JOIN "+a.t("user")+" u ON u.user_id=w.user_id WHERE "+where+" ORDER BY w.withdrawal_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	case "detail":
		if r.n("withdrawal_id") < 1 {
			return nil, r.fail("提现记录ID无效")
		}
		item, err := r.one("SELECT w.withdrawal_id,w.user_id,w.amount_cents,w.status,w.receipt_name,w.bank_name,w.branch_name,w.account_cipher,w.account_last4,w.created_at,w.processed_at,w.processed_by,w.remark,u.account,u.realname FROM "+a.t("imgo_withdrawal")+" w JOIN "+a.t("user")+" u ON u.user_id=w.user_id WHERE w.withdrawal_id=?", r.n("withdrawal_id"))
		if err != nil {
			return nil, err
		}
		account, err := decryptBankAccount(a.cfg.JWTKey, str(item["account_cipher"]))
		if err != nil {
			return nil, err
		}
		delete(item, "account_cipher")
		item["receipt_account"] = account
		return item, nil
	default:
		return nil, r.fail("未知操作")
	}
}
