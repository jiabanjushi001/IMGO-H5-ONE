package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

const (
	identityPending  int64 = 0
	identityApproved int64 = 1
	identityRejected int64 = 2
)

func identityCipher(key string) (cipher.AEAD, error) {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte("imgo-identity-number-v1"))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encryptIdentityNumber(key, number string) (string, error) {
	aead, err := identityCipher(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return "idv1:" + base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(number), nil)), nil
}

func decryptIdentityNumber(key, encrypted string) (string, error) {
	if !strings.HasPrefix(encrypted, "idv1:") {
		return "", errors.New("未知的身份证密文格式")
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encrypted, "idv1:"))
	if err != nil {
		return "", err
	}
	aead, err := identityCipher(key)
	if err != nil {
		return "", err
	}
	if len(data) < aead.NonceSize()+aead.Overhead() {
		return "", errors.New("身份证密文不完整")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], nil)
	return string(plain), err
}

func exposeIdentityNumber(row M, key string) error {
	fullNumber, err := decryptIdentityNumber(key, str(row["id_number_cipher"]))
	if err != nil {
		return err
	}
	delete(row, "id_number_cipher")
	row["id_number"] = fullNumber
	return nil
}

func identityNumberHash(key, number string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte("imgo-identity-number-index-v1:"))
	mac.Write([]byte(number))
	return hex.EncodeToString(mac.Sum(nil))
}

func validIdentityName(value string) bool {
	if value != strings.TrimSpace(value) || utf8.RuneCountInString(value) < 2 || utf8.RuneCountInString(value) > 50 {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func normalizeIdentityNumber(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func validIdentityNumber(value string) bool {
	value = normalizeIdentityNumber(value)
	if len(value) != 18 {
		return false
	}
	for i := 0; i < 17; i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	if (value[17] < '0' || value[17] > '9') && value[17] != 'X' {
		return false
	}
	birth, err := time.Parse("20060102", value[6:14])
	if err != nil || birth.Year() < 1900 || birth.After(time.Now()) {
		return false
	}
	weights := [...]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checks := "10X98765432"
	sum := 0
	for i, weight := range weights {
		sum += int(value[i]-'0') * weight
	}
	return value[17] == checks[sum%11]
}

func maskIdentityNumber(value string) string {
	value = normalizeIdentityNumber(value)
	if len(value) != 18 {
		return ""
	}
	return value[:6] + "********" + value[14:]
}

func identityStatus(value any) (int64, bool) {
	status, err := strconv.ParseInt(str(value), 10, 64)
	return status, err == nil && status >= identityPending && status <= identityRejected
}

func (a *App) identityFile(ctx context.Context, uid, fileID int64) (M, error) {
	if fileID <= 0 {
		return nil, clientError{"请上传身份证正反面照片", 400}
	}
	file, err := one(ctx, a.db, "SELECT file_id,user_id,cate,src FROM "+a.t("file")+" WHERE file_id=? AND status=1 AND COALESCE(delete_time,0)=0", fileID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (number(file["user_id"]) != uid || number(file["cate"]) != 2) {
		return nil, clientError{"身份证照片无效，请重新上传", 400}
	}
	return file, err
}

func (a *App) identityVerification(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "get":
		row, err := r.one("SELECT v.user_id,v.real_name,v.id_number_cipher,v.id_number_masked,v.front_file_id,v.back_file_id,v.status,v.remark,v.version,v.submitted_at,v.reviewed_at,v.created_at,v.updated_at,ff.src AS front_image,bf.src AS back_image FROM "+a.t("imgo_identity_verification")+" v JOIN "+a.t("file")+" ff ON ff.file_id=v.front_file_id JOIN "+a.t("file")+" bf ON bf.file_id=v.back_file_id WHERE v.user_id=?", r.uid())
		if errors.Is(err, sql.ErrNoRows) {
			return M{"submitted": false, "status": int64(-1), "is_auth": false}, nil
		}
		if err != nil {
			return nil, err
		}
		if err = exposeIdentityNumber(row, a.cfg.JWTKey); err != nil {
			return nil, err
		}
		row["submitted"] = true
		row["is_auth"] = number(row["status"]) == identityApproved
		row["front_image"] = a.mediaPath(str(row["front_image"]))
		row["back_image"] = a.mediaPath(str(row["back_image"]))
		return row, nil
	case "save":
		name := strings.TrimSpace(r.s("real_name"))
		numberValue := normalizeIdentityNumber(r.s("id_number"))
		frontID, backID := r.n("front_file_id"), r.n("back_file_id")
		if !validIdentityName(name) {
			return nil, r.fail("真实姓名需为2至50个字符")
		}
		if !validIdentityNumber(numberValue) {
			return nil, r.fail("请输入有效的18位身份证号")
		}
		if frontID == backID {
			return nil, r.fail("身份证正反面不能使用同一张照片")
		}
		if _, err := a.identityFile(r.ctx(), r.uid(), frontID); err != nil {
			return nil, err
		}
		if _, err := a.identityFile(r.ctx(), r.uid(), backID); err != nil {
			return nil, err
		}
		existing, err := r.one("SELECT status FROM "+a.t("imgo_identity_verification")+" WHERE user_id=?", r.uid())
		isNew := errors.Is(err, sql.ErrNoRows)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil && number(existing["status"]) != identityRejected {
			return nil, clientError{"实名认证已提交，当前状态不允许修改", 409}
		}
		ciphertext, err := encryptIdentityNumber(a.cfg.JWTKey, numberValue)
		if err != nil {
			return nil, err
		}
		now := time.Now().Unix()
		tx, err := a.db.BeginTx(r.ctx(), nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if isNew {
			_, err = tx.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_identity_verification")+" (user_id,real_name,id_number_cipher,id_number_hash,id_number_masked,front_file_id,back_file_id,status,remark,version,submitted_at,reviewed_at,reviewed_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,0,'',1,?,0,0,?,?)", r.uid(), name, ciphertext, identityNumberHash(a.cfg.JWTKey, numberValue), maskIdentityNumber(numberValue), frontID, backID, now, now, now)
		} else {
			result, updateErr := tx.ExecContext(r.ctx(), "UPDATE "+a.t("imgo_identity_verification")+" SET real_name=?,id_number_cipher=?,id_number_hash=?,id_number_masked=?,front_file_id=?,back_file_id=?,status=0,remark='',version=version+1,submitted_at=?,reviewed_at=0,reviewed_by=0,updated_at=? WHERE user_id=? AND status=2", name, ciphertext, identityNumberHash(a.cfg.JWTKey, numberValue), maskIdentityNumber(numberValue), frontID, backID, now, now, r.uid())
			err = updateErr
			if err == nil {
				if affected, _ := result.RowsAffected(); affected == 0 {
					return nil, clientError{"实名认证状态已变化，请刷新后重试", 409}
				}
			}
		}
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return nil, clientError{"该身份证号已被其他账号使用", 409}
		}
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(r.ctx(), "UPDATE "+a.t("user")+" SET is_auth=0 WHERE user_id=?", r.uid()); err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return M{"submitted": true, "status": identityPending, "is_auth": false}, nil
	}
	return nil, r.fail("未知操作")
}

func (a *App) manageIdentityVerification(r *request) (any, error) {
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	r.c.Header("Cache-Control", "no-store")
	switch action(r) {
	case "index":
		where, args := "u.delete_time=0", []any{}
		if !scope.Global {
			predicate, values := scope.userPredicate("v")
			where += " AND " + predicate
			args = append(args, values...)
		}
		if keyword := strings.TrimSpace(r.s("keywords")); keyword != "" {
			where += " AND (u.account LIKE ? OR u.realname LIKE ? OR v.real_name LIKE ?"
			args = append(args, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
			if validIdentityNumber(keyword) {
				where += " OR v.id_number_hash=?"
				args = append(args, identityNumberHash(a.cfg.JWTKey, normalizeIdentityNumber(keyword)))
			}
			where += ")"
		}
		if _, ok := r.p["status"]; ok && r.s("status") != "" {
			status, valid := identityStatus(r.p["status"])
			if !valid {
				return nil, r.fail("状态无效")
			}
			where += " AND v.status=?"
			args = append(args, status)
		}
		count, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_identity_verification")+" v JOIN "+a.t("user")+" u ON u.user_id=v.user_id WHERE "+where, args...)
		if err != nil {
			return nil, err
		}
		r.count = number(count["n"])
		limit, offset := r.pagination()
		return r.list("SELECT v.user_id,v.real_name,v.id_number_masked,v.status,v.remark,v.version,v.submitted_at,v.reviewed_at,v.updated_at,u.account AS login_account,u.realname AS user_name FROM "+a.t("imgo_identity_verification")+" v JOIN "+a.t("user")+" u ON u.user_id=v.user_id WHERE "+where+" ORDER BY (v.status=0) DESC,v.updated_at DESC,v.user_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	case "detail":
		uid := r.n("user_id")
		if uid <= 0 {
			return nil, r.fail("用户ID无效")
		}
		if !scope.Global {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
		}
		row, err := r.one("SELECT v.user_id,v.real_name,v.id_number_cipher,v.id_number_masked,v.front_file_id,v.back_file_id,v.status,v.remark,v.version,v.submitted_at,v.reviewed_at,v.reviewed_by,v.created_at,v.updated_at,ff.src AS front_image,bf.src AS back_image,u.account AS login_account,u.realname AS user_name FROM "+a.t("imgo_identity_verification")+" v JOIN "+a.t("user")+" u ON u.user_id=v.user_id JOIN "+a.t("file")+" ff ON ff.file_id=v.front_file_id JOIN "+a.t("file")+" bf ON bf.file_id=v.back_file_id WHERE v.user_id=?", uid)
		if err != nil {
			return nil, err
		}
		if err = exposeIdentityNumber(row, a.cfg.JWTKey); err != nil {
			return nil, err
		}
		row["front_image"] = a.mediaPath(str(row["front_image"]))
		row["back_image"] = a.mediaPath(str(row["back_image"]))
		return row, nil
	case "review":
		uid, version := r.n("user_id"), r.n("version")
		status, valid := identityStatus(r.p["status"])
		remark := strings.TrimSpace(r.s("remark"))
		if uid <= 0 || version <= 0 || !valid || status == identityPending || len(remark) > 500 {
			return nil, r.fail("审核参数无效")
		}
		if status == identityRejected && remark == "" {
			return nil, r.fail("拒绝时请填写原因")
		}
		if !scope.Global {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
		}
		now := time.Now().Unix()
		tx, err := a.db.BeginTx(r.ctx(), nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(r.ctx(), "UPDATE "+a.t("imgo_identity_verification")+" SET status=?,remark=?,version=version+1,reviewed_at=?,reviewed_by=?,updated_at=? WHERE user_id=? AND version=? AND status=0", status, remark, now, r.uid(), now, uid, version)
		if err != nil {
			return nil, err
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil, clientError{"记录已被审核或更新，请刷新后重试", 409}
		}
		isAuth := int64(0)
		if status == identityApproved {
			isAuth = 1
		}
		if _, err = tx.ExecContext(r.ctx(), "UPDATE "+a.t("user")+" SET is_auth=? WHERE user_id=?", isAuth, uid); err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return M{"saved": true, "status": status}, nil
	}
	return nil, r.fail("未知操作")
}

func (a *App) canReadIdentityFile(ctx context.Context, viewerID int64, file M) bool {
	record, err := one(ctx, a.db, "SELECT user_id FROM "+a.t("imgo_identity_verification")+" WHERE front_file_id=? OR back_file_id=? LIMIT 1", file["file_id"], file["file_id"])
	if err != nil {
		return false
	}
	ownerID := number(record["user_id"])
	if ownerID == viewerID {
		return true
	}
	viewer, err := one(ctx, a.db, "SELECT user_id,admin_role_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", viewerID)
	if err != nil || a.authorizeManage(ctx, viewer, "manage.users") != nil {
		return false
	}
	scope, err := a.adminScope(ctx, viewer)
	if err != nil {
		return false
	}
	if scope.Global {
		return true
	}
	return a.requireScopedUser(ctx, a.db, scope, ownerID) == nil
}
