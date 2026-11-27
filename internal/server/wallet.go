package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// Attach wallet amounts for the visible member page in one query. The balance
// follows the user-facing definition: available funds plus administrator-frozen
// withdrawals; ordinary pending withdrawals stay separate.
func (a *App) memberWalletSummaries(r *request, users []M) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[int64]M, len(users))
	args := make([]any, 0, len(users)+1)
	args = append(args, withdrawalStatusFrozen)
	for _, user := range users {
		user["available_cents"] = int64(0)
		user["pending_cents"] = int64(0)
		user["frozen_cents"] = int64(0)
		user["balance_cents"] = int64(0)
		userID := number(user["user_id"])
		byID[userID] = user
		args = append(args, userID)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(users)), ",")
	query := "SELECT w.user_id,w.available_cents,w.pending_cents,COALESCE(SUM(d.amount_cents),0) frozen_cents FROM " + a.t("imgo_wallet") + " w LEFT JOIN " + a.t("imgo_withdrawal") + " d ON d.user_id=w.user_id AND d.status=? WHERE w.user_id IN (" + placeholders + ") GROUP BY w.user_id,w.available_cents,w.pending_cents"
	summaries, err := r.list(query, args...)
	if err != nil {
		return err
	}
	for _, summary := range summaries {
		if user := byID[number(summary["user_id"])]; user != nil {
			available := number(summary["available_cents"])
			frozen := number(summary["frozen_cents"])
			user["available_cents"] = available
			user["pending_cents"] = number(summary["pending_cents"])
			user["frozen_cents"] = frozen
			user["balance_cents"] = available + frozen
		}
	}
	return nil
}

func (a *App) walletStatus(r *request) (M, error) {
	balance, err := a.walletBalance(r, r.uid())
	if err != nil {
		return nil, err
	}
	frozen, err := r.one(
		"SELECT COALESCE(SUM(amount_cents),0) frozen_cents FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? AND status=?",
		r.uid(), withdrawalStatusFrozen,
	)
	if err != nil {
		return nil, err
	}
	balance["frozen_cents"] = number(frozen["frozen_cents"])
	settings, err := a.withdrawalSettingsForUser(r.ctx(), a.db, r.uid())
	if err != nil {
		return nil, err
	}
	balance["withdrawal_enabled"] = settings.Enabled
	balance["minimum_withdrawal_cents"] = settings.MinYuan * 100
	balance["withdrawal_password_required"] = settings.PasswordRequired
	balance["withdrawal_password_set"] = false
	if settings.PasswordRequired {
		set, err := a.withdrawalPasswordSet(r.ctx(), a.db, r.uid())
		if err != nil {
			return nil, err
		}
		balance["withdrawal_password_set"] = set
	}
	return balance, nil
}

func (a *App) previousWithdrawal(r *request, requestID string, cents int64, paymentType string) (M, error) {
	previous, err := r.one("SELECT withdrawal_id,amount_cents,status,payment_type FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? AND request_id=?", r.uid(), requestID)
	if err != nil {
		return nil, err
	}
	if number(previous["amount_cents"]) != cents || str(previous["payment_type"]) != paymentType {
		return nil, clientError{"重复请求的金额或收款方式不一致", 409}
	}
	return M{"withdrawal_id": previous["withdrawal_id"], "status": previous["status"], "amount_cents": previous["amount_cents"], "payment_type": previous["payment_type"]}, nil
}

func (a *App) walletWithdraw(r *request) (any, error) {
	cents, valid := parseWalletAmount(r.s("amount"))
	requestID := r.s("request_id")
	paymentType := strings.ToLower(strings.TrimSpace(r.s("payment_type")))
	if paymentType == "" {
		paymentType = paymentTypeBank
	}
	if !valid || !walletRequestID.MatchString(requestID) || !validPaymentType(paymentType) {
		return nil, r.fail("提现金额或请求编号无效")
	}
	if previous, err := a.previousWithdrawal(r, requestID, cents, paymentType); err == nil {
		return previous, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	settings, err := a.withdrawalSettingsForUser(r.ctx(), a.db, r.uid())
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, clientError{"提现功能暂未开放", 403}
	}
	if cents < settings.MinYuan*100 {
		return nil, clientError{fmt.Sprintf("最低提现金额为¥%d.00", settings.MinYuan), 400}
	}
	if settings.PasswordRequired {
		if err = a.verifyWithdrawalPassword(r.ctx(), a.db, r.uid(), r.s("withdrawal_password")); err != nil {
			return nil, err
		}
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
	previous, err := one(ctx, tx, "SELECT withdrawal_id,amount_cents,status,payment_type FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? AND request_id=? FOR UPDATE", r.uid(), requestID)
	if err == nil {
		if number(previous["amount_cents"]) != cents || str(previous["payment_type"]) != paymentType {
			return nil, clientError{"重复请求的金额或收款方式不一致", 409}
		}
		return M{"withdrawal_id": previous["withdrawal_id"], "status": previous["status"], "amount_cents": previous["amount_cents"], "payment_type": previous["payment_type"]}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if number(wallet["available_cents"]) < cents {
		return nil, clientError{"可提现余额不足", 409}
	}
	methods, err := a.withdrawalMethodsForUser(ctx, tx, r.uid())
	if err != nil {
		return nil, err
	}
	if !withdrawalMethodEnabled(methods, paymentType) {
		return nil, clientError{"该提现方式已被导师关闭", 403}
	}
	payment := M{"bank_name": "", "branch_name": "", "account_cipher": "", "account_last4": "", "qr_file_id": int64(0)}
	if paymentType == paymentTypeBank {
		payment, err = one(ctx, tx, "SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4,0 AS qr_file_id FROM "+a.t("imgo_bank_card")+" WHERE user_id=? AND status=1 FOR UPDATE", r.uid())
		if errors.Is(err, sql.ErrNoRows) {
			return nil, clientError{"请先绑定并通过审核的银行卡", 409}
		}
	} else {
		payment, err = one(ctx, tx, "SELECT receipt_name,account_cipher,account_last4,qr_file_id FROM "+a.t("imgo_payment_method")+" WHERE user_id=? AND payment_type=? AND status=1 FOR UPDATE", r.uid(), paymentType)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, clientError{"请先绑定并通过审核的" + paymentQRCodeLabel(paymentType), 409}
		}
		if err == nil && number(payment["qr_file_id"]) <= 0 {
			return nil, clientError{"请先上传并通过审核的" + paymentQRCodeLabel(paymentType), 409}
		}
		payment["bank_name"] = paymentQRCodeLabel(paymentType)
		payment["branch_name"] = ""
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
	result, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_withdrawal")+" (user_id,request_id,amount_cents,status,payment_type,receipt_name,bank_name,branch_name,account_cipher,account_last4,payment_qr_file_id,created_at,processed_at,processed_by,remark) VALUES (?,?,?,0,?,?,?,?,?,?,?,?,0,0,'')", r.uid(), requestID, cents, paymentType, payment["receipt_name"], payment["bank_name"], payment["branch_name"], payment["account_cipher"], payment["account_last4"], payment["qr_file_id"], now)
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			_ = tx.Rollback()
			return a.previousWithdrawal(r, requestID, cents, paymentType)
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
	a.queueSystemAlert(systemAlertEvent{Type: "user_withdraw", OccurredAt: time.Now(), IP: a.clientIP(r.c), Actor: systemAlertUserIdentity(r.user), ActorID: r.uid(), Target: systemAlertUserIdentity(r.user), TargetID: r.uid(), AmountCents: cents, Detail: fmt.Sprintf("提现单 #%d（用户端提交）", id)})
	a.notifyMentorWithdrawal(r.ctx(), r.uid(), id, cents, r.user)
	return M{"withdrawal_id": id, "status": 0, "amount_cents": cents, "payment_type": paymentType}, nil
}

// A withdrawal is already committed when this runs. Notification lookup is
// intentionally best-effort and must never roll back or duplicate the order.
func (a *App) notifyMentorWithdrawal(ctx context.Context, userID, withdrawalID, amountCents int64, user M) {
	mentorID, err := a.nearestAgent(ctx, a.db, userID)
	if err != nil {
		a.log.Warn("withdrawal mentor notification lookup failed", "user_id", userID, "error", err)
		return
	}
	if mentorID < 1 {
		return
	}
	a.hub.send([]int64{mentorID}, "withdrawalSubmitted", M{
		"withdrawal_id": withdrawalID,
		"user_id":       userID,
		"account":       str(user["account"]),
		"realname":      str(user["realname"]),
		"amount_cents":  amountCents,
	})
}

func (a *App) wallet(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "status":
		return a.walletStatus(r)
	case "setwithdrawalpassword":
		return a.setWithdrawalPassword(r)
	case "history":
		n, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_withdrawal")+" WHERE user_id=?", r.uid())
		if err != nil {
			return nil, err
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		return r.list("SELECT withdrawal_id,amount_cents,status,payment_type,bank_name,account_last4,created_at,processed_at,remark FROM "+a.t("imgo_withdrawal")+" WHERE user_id=? ORDER BY withdrawal_id DESC LIMIT ? OFFSET ?", r.uid(), limit, offset)
	case "entries":
		return a.walletEntries(r, r.uid())
	case "withdraw":
		return a.walletWithdraw(r)
	default:
		return nil, r.fail("未知操作")
	}
}
