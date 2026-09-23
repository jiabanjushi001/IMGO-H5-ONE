package server

import (
	"bytes"
	"crypto/aes"
	"crypto/hmac"
	"crypto/md5" // Legacy PHP hashes only; new passwords use bcrypt.
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"path"
	"strings"
	"time"
)

func randomID() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func legacyPassword(p, s string) string { return fmt.Sprintf("%x", md5.Sum([]byte(s+p+s))) }
func hashPassword(p string) (string, error) {
	if len(p) < 6 || len(p) > 30 {
		return "", errors.New("密码长度须为 6–30 字节")
	}
	b, e := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), e
}
func checkPassword(h, s, p string) bool {
	if strings.HasPrefix(h, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(h), []byte(p)) == nil
	}
	return len(h) == 32 && subtle.ConstantTimeCompare([]byte(h), []byte(legacyPassword(p, s))) == 1
}

type claims struct {
	UID int64  `json:"sub"`
	SID string `json:"jti"`
	Exp int64  `json:"exp"`
}

func signToken(key string, uid int64, sid string, expiry time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, _ := json.Marshal(claims{uid, sid, expiry.Unix()})
	s := header + "." + base64.RawURLEncoding.EncodeToString(body)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(s))
	return s + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func parseToken(key, token string) (claims, error) {
	var c claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return c, errors.New("登录已失效")
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(parts[0] + "." + parts[1]))
	sig, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil || !hmac.Equal(sig, h.Sum(nil)) {
		return c, errors.New("登录已失效")
	}
	header, e := base64.RawURLEncoding.DecodeString(parts[0])
	var algorithm struct {
		Alg string `json:"alg"`
	}
	if e != nil || json.Unmarshal(header, &algorithm) != nil || algorithm.Alg != "HS256" {
		return c, errors.New("非法令牌算法")
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || json.Unmarshal(b, &c) != nil || c.UID < 1 || c.SID == "" || c.Exp <= time.Now().Unix() {
		return c, errors.New("登录已失效")
	}
	return c, nil
}
func cryptKey(key string) []byte { b := make([]byte, 16); copy(b, key); return b }
func encryptContent(key, s string) (string, error) {
	if key == "" || s == "" {
		return s, nil
	}
	block, e := aes.NewCipher(cryptKey(key))
	if e != nil {
		return "", e
	}
	b := []byte(s)
	pad := 16 - len(b)%16
	b = append(b, bytes.Repeat([]byte{byte(pad)}, pad)...)
	for i := 0; i < len(b); i += 16 {
		block.Encrypt(b[i:i+16], b[i:i+16])
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
func decryptContent(key, s string) (string, error) {
	if key == "" || s == "" {
		return s, nil
	}
	b, e := base64.StdEncoding.DecodeString(s)
	if e != nil || len(b) == 0 || len(b)%16 != 0 {
		return "", errors.New("聊天密文无效，请检查 CHAT_KEY")
	}
	block, _ := aes.NewCipher(cryptKey(key))
	for i := 0; i < len(b); i += 16 {
		block.Decrypt(b[i:i+16], b[i:i+16])
	}
	p := int(b[len(b)-1])
	if p < 1 || p > 16 || !bytes.Equal(b[len(b)-p:], bytes.Repeat([]byte{byte(p)}, p)) {
		return "", errors.New("CHAT_KEY 与旧数据库不匹配")
	}
	return string(b[:len(b)-p]), nil
}
func safeAsset(s string) bool {
	s = strings.ToLower(s)
	for _, part := range strings.Split(strings.ReplaceAll(s, "\\", "/"), "/") {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	for _, ext := range []string{".php", ".phtml", ".phar", ".env", ".sql", ".sh", ".go", ".log", ".ini", ".toml", ".yaml", ".yml", ".bak"} {
		if strings.Contains(s, ext) {
			return false
		}
	}
	return !strings.Contains(s, "..") && path.Ext(s) != ""
}

func (a *App) validGroupToken(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && number(parts[0]) > 0 && number(parts[1]) > time.Now().Unix() && hmac.Equal([]byte(parts[2]), []byte(a.signLink(parts[0]+"."+parts[1])))
}
