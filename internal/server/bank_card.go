package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

// Bank account encryption is separate from legacy chat encryption. Keep JWT_KEY
// stable while records exist; rotating it requires re-encrypting saved accounts.
func bankCipher(key string) (cipher.AEAD, error) {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte("imgo-bank-account-v1"))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encryptBankAccount(key, account string) (string, error) {
	aead, err := bankCipher(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return "v1:" + base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(account), nil)), nil
}

func decryptBankAccount(key, encrypted string) (string, error) {
	if !strings.HasPrefix(encrypted, "v1:") {
		return "", errors.New("未知的银行卡密文格式")
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encrypted, "v1:"))
	if err != nil {
		return "", err
	}
	aead, err := bankCipher(key)
	if err != nil {
		return "", err
	}
	if len(data) < aead.NonceSize()+aead.Overhead() {
		return "", errors.New("银行卡密文不完整")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], nil)
	return string(plain), err
}

func validBankAccount(s string) bool {
	if len(s) < 12 || len(s) > 30 {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func validBankName(s string) bool {
	if utf8.RuneCountInString(s) < 2 || len(s) > 100 || strings.TrimSpace(s) != s {
		return false
	}
	for _, ch := range s {
		if unicode.IsControl(ch) {
			return false
		}
	}
	return true
}

func validBankInstitution(s string) bool {
	length := utf8.RuneCountInString(s)
	if length < 2 || length > 120 || strings.TrimSpace(s) != s {
		return false
	}
	for _, ch := range s {
		if unicode.IsControl(ch) {
			return false
		}
	}
	return true
}

func bankStatus(v any) (int64, bool) {
	n, err := strconv.ParseInt(str(v), 10, 64)
	return n, err == nil && n >= 0 && n <= 2
}

func bankCardSummary(card M) M {
	if card == nil {
		return M{"bound": false}
	}
	return M{
		"bound": true, "user_id": card["user_id"], "receipt_name": card["receipt_name"],
		"bank_name": card["bank_name"], "branch_name": card["branch_name"],
		"receipt_account_masked": "•••• •••• •••• " + str(card["account_last4"]),
		"status":                 card["status"], "remark": card["remark"], "version": card["version"],
		"created_at": card["created_at"], "updated_at": card["updated_at"],
	}
}

func (a *App) bankCard(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	if action(r) == "get" {
		card, err := r.one("SELECT user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at FROM "+a.t("imgo_bank_card")+" WHERE user_id=?", r.uid())
		if errors.Is(err, sql.ErrNoRows) {
			return bankCardSummary(nil), nil
		}
		return bankCardSummary(card), err
	}
	if action(r) != "save" {
		return nil, r.fail("未知操作")
	}
	name := strings.TrimSpace(r.s("name"))
	bankName := strings.TrimSpace(r.s("bank_name"))
	branchName := strings.TrimSpace(r.s("branch_name"))
	account := strings.ReplaceAll(strings.TrimSpace(r.s("card_number")), " ", "")
	// This restriction is for the user's self-service route, not management edits.
	conflict := clientError{"已提交的银行卡只能在管理员拒绝后修改，请刷新查看状态", 409}
	card, err := r.one("SELECT status FROM "+a.t("imgo_bank_card")+" WHERE user_id=?", r.uid())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil && number(card["status"]) != 2 {
		return nil, conflict
	}
	if !validBankName(name) {
		return nil, r.fail("收款姓名需为2至100字节")
	}
	if !validBankInstitution(bankName) || !validBankInstitution(branchName) {
		return nil, r.fail("请填写收款银行和支行名称（各2至120字）")
	}
	if account != "" && !validBankAccount(account) {
		return nil, r.fail("卡号需为12至30位数字")
	}
	now := time.Now().Unix()
	if errors.Is(err, sql.ErrNoRows) {
		if account == "" {
			return nil, r.fail("请填写卡号")
		}
		encrypted, err := encryptBankAccount(a.cfg.JWTKey, account)
		if err != nil {
			return nil, err
		}
		err = r.exec("INSERT INTO "+a.t("imgo_bank_card")+" (user_id,receipt_name,bank_name,branch_name,account_cipher,account_last4,status,remark,version,created_at,updated_at) VALUES (?,?,?,?,?,?,0,'',1,?,?)", r.uid(), name, bankName, branchName, encrypted, account[len(account)-4:], now, now)
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return nil, conflict
		}
		return M{"saved": true}, err
	}
	query := "UPDATE " + a.t("imgo_bank_card") + " SET receipt_name=?,bank_name=?,branch_name=?,status=0,remark='',version=version+1,updated_at=?"
	params := []any{name, bankName, branchName, now}
	if account != "" {
		encrypted, err := encryptBankAccount(a.cfg.JWTKey, account)
		if err != nil {
			return nil, err
		}
		query += ",account_cipher=?,account_last4=?"
		params = append(params, encrypted, account[len(account)-4:])
	}
	query += " WHERE user_id=? AND status=2"
	params = append(params, r.uid())
	result, err := a.db.ExecContext(r.ctx(), query, params...)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return nil, conflict
	}
	return M{"saved": true}, nil
}

func (a *App) manageBankCard(r *request) (any, error) {
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "index":
		where, args := "u.delete_time=0", []any{}
		if !scope.Global {
			predicate, params := scope.userPredicate("b")
			where += " AND " + predicate
			args = append(args, params...)
		}
		if s := strings.TrimSpace(r.s("keywords")); s != "" {
			where += " AND (u.account LIKE ? OR u.realname LIKE ? OR b.receipt_name LIKE ? OR b.bank_name LIKE ? OR b.branch_name LIKE ?)"
			for i := 0; i < 5; i++ {
				args = append(args, "%"+s+"%")
			}
		}
		if _, ok := r.p["status"]; ok && r.s("status") != "" {
			status, valid := bankStatus(r.p["status"])
			if !valid {
				return nil, r.fail("状态无效")
			}
			where += " AND b.status=?"
			args = append(args, status)
		}
		n, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_bank_card")+" b JOIN "+a.t("user")+" u ON u.user_id=b.user_id WHERE "+where, args...)
		if err != nil {
			return nil, err
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		items, err := r.list("SELECT b.user_id,b.receipt_name,b.bank_name,b.branch_name,b.account_last4,b.status,b.remark,b.version,b.created_at,b.updated_at,u.account AS login_account,u.realname AS user_name FROM "+a.t("imgo_bank_card")+" b JOIN "+a.t("user")+" u ON u.user_id=b.user_id WHERE "+where+" ORDER BY b.updated_at DESC,b.user_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
		for _, card := range items {
			card["receipt_account_masked"] = "•••• •••• •••• " + str(card["account_last4"])
			delete(card, "account_last4")
		}
		return items, err
	case "detail", "edit":
		uid := r.n("user_id")
		if uid <= 0 {
			return nil, r.fail("用户ID无效")
		}
		if !scope.Global {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
		}
		fields := "user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at"
		if action(r) == "detail" {
			fields = "user_id,receipt_name,bank_name,branch_name,account_cipher,account_last4,status,remark,version,created_at,updated_at"
		}
		card, err := r.one("SELECT "+fields+" FROM "+a.t("imgo_bank_card")+" WHERE user_id=?", uid)
		if err != nil {
			return nil, err
		}
		if action(r) == "detail" {
			account, err := decryptBankAccount(a.cfg.JWTKey, str(card["account_cipher"]))
			if err != nil {
				return nil, err
			}
			detail := bankCardSummary(card)
			detail["receipt_account"] = account
			return detail, nil
		}
		if r.n("version") != number(card["version"]) {
			return nil, clientError{"记录已被更新，请刷新后重试", 409}
		}
		name := strings.TrimSpace(r.s("receipt_name"))
		bankName := strings.TrimSpace(r.s("bank_name"))
		branchName := strings.TrimSpace(r.s("branch_name"))
		if _, provided := r.p["bank_name"]; !provided {
			bankName = str(card["bank_name"])
		}
		if _, provided := r.p["branch_name"]; !provided {
			branchName = str(card["branch_name"])
		}
		account, replacing := "", false
		if _, provided := r.p["receipt_account"]; provided {
			account = strings.ReplaceAll(strings.TrimSpace(r.s("receipt_account")), " ", "")
			if !validBankAccount(account) {
				return nil, r.fail("卡号需为12至30位数字")
			}
			replacing = true
		}
		status, valid := bankStatus(r.p["status"])
		remark := strings.TrimSpace(r.s("remark"))
		if !validBankName(name) || (bankName != "" && !validBankInstitution(bankName)) || (branchName != "" && !validBankInstitution(branchName)) || !valid || len(remark) > 500 {
			return nil, r.fail("绑卡资料无效")
		}
		query := "UPDATE " + a.t("imgo_bank_card") + " SET receipt_name=?,bank_name=?,branch_name=?,status=?,remark=?,version=version+1,updated_at=?"
		params := []any{name, bankName, branchName, status, remark, time.Now().Unix()}
		if replacing {
			encrypted, err := encryptBankAccount(a.cfg.JWTKey, account)
			if err != nil {
				return nil, err
			}
			query += ",account_cipher=?,account_last4=?"
			params = append(params, encrypted, account[len(account)-4:])
		}
		query += " WHERE user_id=? AND version=?"
		params = append(params, uid, r.n("version"))
		if !scope.Global {
			predicate, scopeArgs := scope.userPredicate("imgo_bank_scope")
			query += " AND EXISTS (SELECT 1 FROM " + a.t("user") + " imgo_bank_scope WHERE imgo_bank_scope.user_id=" + a.t("imgo_bank_card") + ".user_id AND " + predicate + ")"
			params = append(params, scopeArgs...)
		}
		result, err := a.db.ExecContext(r.ctx(), query, params...)
		if err != nil {
			return nil, err
		}
		if n, _ := result.RowsAffected(); n == 0 {
			return nil, clientError{"记录已被更新，请刷新后重试", 409}
		}
		return M{"saved": true}, nil
	}
	return nil, r.fail("未知操作")
}
