package server

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const (
	paymentTypeBank   = "bank"
	paymentTypeAlipay = "alipay"
	paymentTypeWechat = "wechat"
)

func validPaymentType(value string) bool {
	return value == paymentTypeBank || value == paymentTypeAlipay || value == paymentTypeWechat
}

func paymentQRCodeLabel(typeName string) string {
	if typeName == paymentTypeAlipay {
		return "支付宝收款码"
	}
	return "微信收款码"
}

func (a *App) paymentQRCodeFile(ctx context.Context, uid, fileID int64, typeName string) (M, error) {
	label := paymentQRCodeLabel(typeName)
	if fileID <= 0 {
		return nil, clientError{"请上传" + label, 400}
	}
	file, err := one(ctx, a.db, "SELECT file_id,user_id,cate,src FROM "+a.t("file")+" WHERE file_id=? AND status=1 AND COALESCE(delete_time,0)=0", fileID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (number(file["user_id"]) != uid || number(file["cate"]) != 2) {
		return nil, clientError{label + "图片无效，请重新上传", 400}
	}
	return file, err
}

func (a *App) paymentMethods(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "get":
		methods, err := a.withdrawalMethodsForUser(r.ctx(), a.db, r.uid())
		if err != nil {
			return nil, err
		}
		rows, err := r.list("SELECT p.payment_type,p.receipt_name,p.account_cipher,p.account_last4,p.qr_file_id,p.status,p.remark,p.version,p.created_at,p.updated_at,f.src AS qr_image FROM "+a.t("imgo_payment_method")+" p LEFT JOIN "+a.t("file")+" f ON f.file_id=p.qr_file_id AND f.status=1 AND COALESCE(f.delete_time,0)=0 WHERE p.user_id=?", r.uid())
		if err != nil {
			return nil, err
		}
		result := M{
			paymentTypeAlipay: M{"bound": false, "payment_type": paymentTypeAlipay},
			paymentTypeWechat: M{"bound": false, "payment_type": paymentTypeWechat},
			"enabled_methods": methods,
		}
		for _, row := range rows {
			typeName := str(row["payment_type"])
			if (typeName == paymentTypeAlipay || typeName == paymentTypeWechat) && str(row["qr_image"]) != "" {
				row["qr_image"] = a.mediaPath(str(row["qr_image"]))
			}
			delete(row, "account_cipher")
			row["bound"] = true
			result[typeName] = row
		}
		return result, nil
	case "save":
		typeName := strings.ToLower(strings.TrimSpace(r.s("payment_type")))
		name := strings.TrimSpace(r.s("receipt_name"))
		if typeName != paymentTypeAlipay && typeName != paymentTypeWechat {
			return nil, r.fail("收款方式无效")
		}
		if !validBankName(name) {
			return nil, r.fail("收款姓名需为2至100字节")
		}
		conflict := clientError{"已提交的收款方式只能在管理员拒绝后修改，请刷新查看状态", 409}
		existing, err := r.one("SELECT status,qr_file_id FROM "+a.t("imgo_payment_method")+" WHERE user_id=? AND payment_type=?", r.uid(), typeName)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		legacyAlipay := err == nil && typeName == paymentTypeAlipay && number(existing["qr_file_id"]) == 0
		if err == nil && number(existing["status"]) != 2 && !legacyAlipay {
			return nil, conflict
		}
		accountCipher, accountLast4, qrFileID := "", "", r.n("qr_file_id")
		if _, err := a.paymentQRCodeFile(r.ctx(), r.uid(), qrFileID, typeName); err != nil {
			return nil, err
		}
		now := time.Now().Unix()
		if errors.Is(err, sql.ErrNoRows) {
			_, err = a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_payment_method")+" (user_id,payment_type,receipt_name,account_cipher,account_last4,qr_file_id,status,remark,version,created_at,updated_at) VALUES (?,?,?,?,?,?,0,'',1,?,?)", r.uid(), typeName, name, accountCipher, accountLast4, qrFileID, now, now)
			var duplicate *mysql.MySQLError
			if errors.As(err, &duplicate) && duplicate.Number == 1062 {
				return nil, conflict
			}
			if err == nil {
				a.notifyMentorPaymentMethod(r.ctx(), r.uid(), typeName, r.user)
			}
			return M{"saved": err == nil, "payment_type": typeName, "status": 0}, err
		}
		where := "status=2"
		if typeName == paymentTypeAlipay {
			where = "(status=2 OR qr_file_id=0)"
		}
		result, err := a.db.ExecContext(r.ctx(), "UPDATE "+a.t("imgo_payment_method")+" SET receipt_name=?,account_cipher=?,account_last4=?,qr_file_id=?,status=0,remark='',version=version+1,updated_at=? WHERE user_id=? AND payment_type=? AND "+where, name, accountCipher, accountLast4, qrFileID, now, r.uid(), typeName)
		if err != nil {
			return nil, err
		}
		if changed, _ := result.RowsAffected(); changed == 0 {
			return nil, conflict
		}
		a.notifyMentorPaymentMethod(r.ctx(), r.uid(), typeName, r.user)
		return M{"saved": true, "payment_type": typeName, "status": 0}, nil
	default:
		return nil, r.fail("未知操作")
	}
}

// The payment method row is committed before this notification runs. A failed
// mentor lookup must never turn a successful submission into a client error.
func (a *App) notifyMentorPaymentMethod(ctx context.Context, userID int64, typeName string, user M) {
	mentorID, err := a.nearestAgent(ctx, a.db, userID)
	if err != nil {
		a.log.Warn("payment method mentor notification lookup failed", "user_id", userID, "payment_type", typeName, "error", err)
		return
	}
	if mentorID < 1 {
		return
	}
	a.hub.send([]int64{mentorID}, "paymentMethodSubmitted", M{
		"user_id":       userID,
		"account":       str(user["account"]),
		"realname":      str(user["realname"]),
		"payment_type":  typeName,
		"payment_label": paymentQRCodeLabel(typeName),
	})
}

func (a *App) managePaymentMethod(r *request, scope adminScope, typeName string) (any, error) {
	if typeName != paymentTypeAlipay && typeName != paymentTypeWechat {
		return nil, r.fail("收款类型无效")
	}
	uid := r.n("user_id")
	if uid <= 0 {
		return nil, r.fail("用户ID无效")
	}
	if !scope.Global {
		if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
			return nil, err
		}
	}
	fields := "p.user_id,p.payment_type,p.receipt_name,p.account_cipher,p.account_last4,p.qr_file_id,p.status,p.remark,p.version,p.created_at,p.updated_at,f.src AS qr_image"
	record, err := r.one("SELECT "+fields+" FROM "+a.t("imgo_payment_method")+" p LEFT JOIN "+a.t("file")+" f ON f.file_id=p.qr_file_id AND f.status=1 AND COALESCE(f.delete_time,0)=0 WHERE p.user_id=? AND p.payment_type=?", uid, typeName)
	if err != nil {
		return nil, err
	}
	if action(r) == "detail" {
		if str(record["qr_image"]) != "" {
			record["qr_image"] = a.mediaPath(str(record["qr_image"]))
		}
		delete(record, "account_cipher")
		return record, nil
	}
	if action(r) != "edit" {
		return nil, r.fail("未知操作")
	}
	if r.n("version") != number(record["version"]) {
		return nil, clientError{"记录已被更新，请刷新后重试", 409}
	}
	name := strings.TrimSpace(r.s("receipt_name"))
	status, valid := bankStatus(r.p["status"])
	remark := strings.TrimSpace(r.s("remark"))
	if !validBankName(name) || !valid || len(remark) > 500 {
		return nil, r.fail("收款资料无效")
	}
	query := "UPDATE " + a.t("imgo_payment_method") + " SET receipt_name=?,status=?,remark=?,version=version+1,updated_at=?"
	params := []any{name, status, remark, time.Now().Unix()}
	query += " WHERE user_id=? AND payment_type=? AND version=?"
	params = append(params, uid, typeName, r.n("version"))
	if !scope.Global {
		predicate, scopeArgs := scope.userPredicate("imgo_payment_scope")
		query += " AND EXISTS (SELECT 1 FROM " + a.t("user") + " imgo_payment_scope WHERE imgo_payment_scope.user_id=" + a.t("imgo_payment_method") + ".user_id AND " + predicate + ")"
		params = append(params, scopeArgs...)
	}
	result, err := a.db.ExecContext(r.ctx(), query, params...)
	if err != nil {
		return nil, err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return nil, clientError{"记录已被更新，请刷新后重试", 409}
	}
	return M{"saved": true}, nil
}

func (a *App) canReadPaymentFile(ctx context.Context, viewerID int64, file M) bool {
	record, err := one(ctx, a.db, "SELECT user_id FROM "+a.t("imgo_payment_method")+" WHERE qr_file_id=? UNION SELECT user_id FROM "+a.t("imgo_withdrawal")+" WHERE payment_qr_file_id=? LIMIT 1", file["file_id"], file["file_id"])
	if err != nil {
		return false
	}
	ownerID := number(record["user_id"])
	if ownerID == viewerID {
		return true
	}
	viewer, err := one(ctx, a.db, "SELECT user_id,admin_role_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", viewerID)
	if err != nil {
		return false
	}
	if a.authorizeManage(ctx, viewer, "manage.finance") != nil && a.authorizeManage(ctx, viewer, "manage.bank") != nil {
		return false
	}
	scope, err := a.adminScope(ctx, viewer)
	if err != nil {
		return false
	}
	return scope.Global || a.requireScopedUser(ctx, a.db, scope, ownerID) == nil
}
