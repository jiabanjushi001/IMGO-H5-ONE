package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

const googleAuthPeriod int64 = 30

func (a *App) googleAuthEnabled() bool { return a.googleAuthOn.Load() }

func (a *App) refreshGoogleAuthState(ctx context.Context, db DB) error {
	row, err := one(ctx, db, "SELECT enabled FROM "+a.t("imgo_security_setting")+" WHERE id=1")
	if errors.Is(err, sql.ErrNoRows) {
		a.googleAuthOn.Store(false)
		return nil
	}
	if err != nil {
		return err
	}
	a.googleAuthOn.Store(number(row["enabled"]) == 1)
	return nil
}

// setGoogleAuthEnabled changes the global backend-login policy atomically.
// Enabling is allowed only after the acting super administrator is bound. All
// sessions belonging to unbound users are removed in the same transaction so
// they cannot continue using an already-issued token after the switch changes.
func (a *App) setGoogleAuthEnabled(ctx context.Context, actorID int64, enabled bool) ([]int64, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	disconnected := []int64{}
	if enabled {
		if _, err = one(ctx, tx, "SELECT user_id FROM "+a.t("imgo_user_totp")+" WHERE user_id=? FOR UPDATE", actorID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, clientError{"超级管理员尚未绑定谷歌验证，请先在成员列表的“更多 → 谷歌设置”完成绑定后再开启", 400}
			}
			return nil, err
		}
		active, queryErr := rows(ctx, tx, "SELECT DISTINCT s.user_id FROM "+a.t("imgo_session")+" s LEFT JOIN "+a.t("imgo_user_totp")+" t ON t.user_id=s.user_id WHERE t.user_id IS NULL")
		if queryErr != nil {
			return nil, queryErr
		}
		for _, session := range active {
			if uid := number(session["user_id"]); uid > 0 {
				disconnected = append(disconnected, uid)
			}
		}
		if _, err = tx.ExecContext(ctx, "DELETE s FROM "+a.t("imgo_session")+" s LEFT JOIN "+a.t("imgo_user_totp")+" t ON t.user_id=s.user_id WHERE t.user_id IS NULL"); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_security_setting")+" (id,enabled,updated_by,updated_at) VALUES (1,?,?,?) ON DUPLICATE KEY UPDATE enabled=VALUES(enabled),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)", enabled, actorID, time.Now().Unix()); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return disconnected, nil
}

func generateGoogleSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func normalizeGoogleSecret(secret string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
}

func googleCodeAt(secret string, step int64) (string, bool) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalizeGoogleSecret(secret))
	if err != nil || len(key) < 10 || step < 0 {
		return "", false
	}
	counter := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		counter[i] = byte(step)
		step >>= 8
	}
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter)
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (uint32(digest[offset])&0x7f)<<24 | uint32(digest[offset+1])<<16 | uint32(digest[offset+2])<<8 | uint32(digest[offset+3])
	return fmt.Sprintf("%06d", value%1000000), true
}

func matchingGoogleStep(secret, code string, now time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return 0, false
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	current := now.Unix() / googleAuthPeriod
	for _, step := range []int64{current, current - 1, current + 1} {
		expected, ok := googleCodeAt(secret, step)
		if ok && hmac.Equal([]byte(expected), []byte(code)) {
			return step, true
		}
	}
	return 0, false
}

func (a *App) encryptGoogleSecret(secret string) (string, error) {
	key := sha256.Sum256([]byte(a.cfg.JWTKey + ":google-auth"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(normalizeGoogleSecret(secret)), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (a *App) decryptGoogleSecret(ciphertext string) (string, error) {
	key := sha256.Sum256([]byte(a.cfg.JWTKey + ":google-auth"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	raw, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", errors.New("谷歌验证密钥无效")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("谷歌验证密钥无法解密，请检查 JWT_KEY")
	}
	return string(plain), nil
}

func googleAuthURI(systemName, account, secret string) string {
	issuer := strings.TrimSpace(systemName)
	if issuer == "" {
		issuer = "Imgo"
	}
	label := issuer + ":" + account
	query := url.Values{"secret": {secret}, "issuer": {issuer}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + url.PathEscape(label) + "?" + query.Encode()
}

func googleAuthQRData(uri string) (string, error) {
	png, err := qrcode.Encode(uri, qrcode.Medium, 240)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

func (a *App) authorizeGoogleAuthTarget(r *request, userID int64) error {
	if userID < 1 {
		return r.fail("成员不存在")
	}
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return err
	}
	if scope.Global || userID == r.uid() {
		return nil
	}
	return a.requireScopedUser(r.ctx(), a.db, scope, userID)
}

func (a *App) googleAuthDetail(r *request) (any, error) {
	userID := r.n("user_id")
	if err := a.authorizeGoogleAuthTarget(r, userID); err != nil {
		return nil, err
	}
	user, err := r.one("SELECT user_id,account,realname FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", userID)
	if err != nil {
		return nil, err
	}
	_, boundErr := r.one("SELECT user_id FROM "+a.t("imgo_user_totp")+" WHERE user_id=?", userID)
	bound := boundErr == nil
	if boundErr != nil && !errors.Is(boundErr, sql.ErrNoRows) {
		return nil, boundErr
	}
	if bound && r.n("reset") != 1 {
		return M{"bound": true, "user_id": userID, "account": user["account"], "realname": user["realname"]}, nil
	}
	secret, err := generateGoogleSecret()
	if err != nil {
		return nil, err
	}
	token := randomID()
	a.put("google-setup:"+token, M{"user_id": userID, "secret": secret}, 10*time.Minute)
	uri := googleAuthURI(str(a.config(r.ctx(), "sysInfo")["name"]), str(user["account"]), secret)
	qr, err := googleAuthQRData(uri)
	if err != nil {
		return nil, err
	}
	return M{"bound": bound, "setup": true, "setup_token": token, "secret": secret, "qr_data": qr, "expires_in": 600, "user_id": userID, "account": user["account"], "realname": user["realname"]}, nil
}

func (a *App) bindGoogleAuth(r *request) (any, error) {
	userID := r.n("user_id")
	if err := a.authorizeGoogleAuthTarget(r, userID); err != nil {
		return nil, err
	}
	token := strings.TrimSpace(r.s("setup_token"))
	pending, ok := a.get("google-setup:"+token, false).(M)
	if !ok || number(pending["user_id"]) != userID {
		return nil, r.fail("绑定二维码已过期，请重新生成")
	}
	secret := str(pending["secret"])
	if _, ok = matchingGoogleStep(secret, r.s("code"), time.Now()); !ok {
		return nil, r.fail("谷歌验证码错误")
	}
	ciphertext, err := a.encryptGoogleSecret(secret)
	if err != nil {
		return nil, err
	}
	_, err = a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_user_totp")+" (user_id,secret_cipher,bound_at,last_used_step) VALUES (?,?,?,0) ON DUPLICATE KEY UPDATE secret_cipher=VALUES(secret_cipher),bound_at=VALUES(bound_at),last_used_step=0", userID, ciphertext, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	a.get("google-setup:"+token, true)
	return M{"bound": true}, nil
}

func (a *App) unbindGoogleAuth(r *request) (any, error) {
	userID := r.n("user_id")
	if err := a.authorizeGoogleAuthTarget(r, userID); err != nil {
		return nil, err
	}
	if err := r.exec("DELETE FROM "+a.t("imgo_user_totp")+" WHERE user_id=?", userID); err != nil {
		return nil, err
	}
	return M{"bound": false}, nil
}

func (a *App) verifyUserGoogleAuth(ctx context.Context, userID int64, code string) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	row, err := one(ctx, tx, "SELECT secret_cipher,last_used_step FROM "+a.t("imgo_user_totp")+" WHERE user_id=? FOR UPDATE", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return clientError{"账号未绑定谷歌验证，请联系管理员", 400}
	}
	if err != nil {
		return err
	}
	secret, err := a.decryptGoogleSecret(str(row["secret_cipher"]))
	if err != nil {
		return err
	}
	step, ok := matchingGoogleStep(secret, code, time.Now())
	if !ok {
		return clientError{"谷歌验证码错误", 400}
	}
	if step <= number(row["last_used_step"]) {
		return clientError{"谷歌验证码已使用，请等待新验证码", 400}
	}
	if err = update(ctx, tx, a.t("imgo_user_totp"), M{"last_used_step": step}, "user_id=?", userID); err != nil {
		return err
	}
	return tx.Commit()
}
