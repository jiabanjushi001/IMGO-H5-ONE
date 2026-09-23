package server

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/speps/go-hashids"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

func (a *App) hashID(id int64) string {
	d := hashids.NewData()
	d.Salt = "raingads"
	d.MinLength = 12
	h, _ := hashids.NewWithData(d)
	s, _ := h.EncodeInt64([]int64{id})
	return s
}
func (a *App) decodeID(s string) int64 {
	d := hashids.NewData()
	d.Salt = "raingads"
	d.MinLength = 12
	h, _ := hashids.NewWithData(d)
	ids, e := h.DecodeInt64WithError(s)
	if e != nil || len(ids) != 1 {
		return 0
	}
	return ids[0]
}
func (a *App) url(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "https://") || strings.HasPrefix(p, "http://") {
		return p
	}
	return a.cfg.BaseURL + "/" + strings.TrimLeft(p, "/")
}
func (a *App) safeUser(u M) M {
	v := pick(u, "user_id", "realname", "avatar", "sex", "motto", "name_py", "role")
	v["id"] = u["user_id"]
	v["displayName"] = u["realname"]
	v["avatar"] = a.userAvatar(u)
	return v
}
func (a *App) userAvatar(u M) string {
	if s := str(u["avatar"]); s != "" {
		return a.mediaPath(s)
	}
	return a.mediaPath("/avatar/" + url.PathEscape(str(u["realname"])) + "/120/" + str(u["user_id"]))
}
func (a *App) public(r *request) (any, error) {
	switch action(r) {
	case "getsysteminfo":
		s := a.config(r.ctx(), "sysInfo")
		chat := a.config(r.ctx(), "chatInfo")
		if _, _, err := a.authenticate(r.ctx(), r.c.GetHeader("Authorization")); err != nil {
			delete(chat, "stunPass")
		}
		return M{"sysInfo": s, "chatInfo": chat, "compass": a.config(r.ctx(), "compass"), "fileUpload": pick(a.config(r.ctx(), "fileUpload"), "size", "preview", "fileExt")}, nil
	case "checkversion":
		return a.checkVersion(r), nil
	case "sendcode":
		return a.sendCode(r)
	case "register":
		return a.registerUser(r)
	case "login":
		return a.login(r)
	case "logout":
		a.setMediaCookie(r.c, "", -1)
		e := r.exec("DELETE FROM "+a.t("imgo_session")+" WHERE sid=?", r.claims.SID)
		a.hub.disconnectSession(r.claims.SID)
		return nil, e
	case "binduid", "bindgroup":
		return nil, a.hub.bind(r.s("client_id"), r.uid(), r.claims)
	case "offline":
		a.hub.disconnectClient(r.s("client_id"), r.uid())
		return nil, nil
	case "avatar":
		return a.userAvatar(r.user), nil
	}
	return nil, r.fail("未知操作")
}
func (a *App) login(r *request) (any, error) {
	if r.s("captcha") != "" && !a.validCaptcha(r.c, r.s("captcha")) {
		return nil, r.fail("图形验证码无效")
	}
	if number(a.get("login-fail:"+r.s("account"), false)) >= 5 {
		return nil, r.fail("密码错误过多，请 5 分钟后重试")
	}
	if !a.allow("login-rate:"+a.clientIP(r.c), 100*time.Millisecond) {
		return nil, r.fail("请求过于频繁")
	}
	var u M
	var e error
	if tok := r.s("token"); tok != "" {
		uid := number(a.get("sso:"+tok, true))
		if uid == 0 {
			return nil, r.fail("登录链接已失效")
		}
		u, e = r.one("SELECT * FROM "+a.t("user")+" WHERE user_id=?", uid)
	} else {
		u, e = r.one("SELECT * FROM "+a.t("user")+" WHERE account=? AND delete_time=0", r.s("account"))
		if e != nil || u == nil {
			return nil, r.fail("账号或密码错误")
		}
		ok := false
		if r.s("code") != "" {
			ok = a.verifyCode(r.s("account"), "1", r.s("code"))
		} else {
			ok = checkPassword(str(u["password"]), str(u["salt"]), r.s("password"))
		}
		if !ok {
			a.recordFailure("login-fail:" + r.s("account"))
			return nil, r.fail("账号或密码错误")
		}
		if !strings.HasPrefix(str(u["password"]), "$2") && r.s("password") != "" {
			hash, err := hashPassword(r.s("password"))
			if err == nil {
				if e = update(r.ctx(), a.db, a.t("user"), M{"password": hash, "salt": ""}, "user_id=?", u["user_id"]); e != nil {
					return nil, e
				}
			}
		}
	}
	if e != nil {
		return nil, e
	}
	if number(u["status"]) != 1 || number(u["delete_time"]) != 0 {
		return nil, r.fail("账号不可用")
	}
	if e = r.exec("UPDATE "+a.t("user")+" SET login_count=login_count+1,last_login_time=?,last_login_ip=? WHERE user_id=?", time.Now().Unix(), a.clientIP(r.c), u["user_id"]); e != nil {
		return nil, e
	}
	a.get("login-fail:"+r.s("account"), true)
	sid := randomID()
	expiry := time.Now().Add(24 * time.Hour)
	_, e = insert(r.ctx(), a.db, a.t("imgo_session"), M{"sid": sid, "user_id": u["user_id"], "expires_at": expiry.Unix()})
	if e != nil {
		return nil, e
	}
	token := signToken(a.cfg.JWTKey, number(u["user_id"]), sid, expiry)
	a.setMediaCookie(r.c, token, 86400)
	v := pick(u, "user_id", "account", "realname", "email", "sex", "role", "admin_role_id", "motto", "remark", "name_py", "cs_uid", "setting", "friend_limit", "group_limit", "is_auth", "status", "create_time")
	access, err := a.adminAccessInfo(r.ctx(), u)
	if err != nil {
		return nil, err
	}
	for key, value := range access {
		v[key] = value
	}
	v["avatar"] = a.userAvatar(u)
	v["id"] = u["user_id"]
	v["displayName"] = u["realname"]
	v["qrUrl"] = a.scanURL("u", a.hashID(number(u["user_id"])))
	setting := obj(u["setting"])
	for _, k := range []string{"hideMessageName", "hideMessageTime", "avatarCricle", "isVoice"} {
		if s, ok := setting[k].(string); ok {
			setting[k] = s == "true"
		}
	}
	v["setting"] = setting
	if r.s("client_id") != "" {
		// A previously opened socket may close while credentials are being checked.
		// Login remains valid; the client can bind a new socket with this token.
		if e = a.hub.bind(r.s("client_id"), number(u["user_id"]), claims{number(u["user_id"]), sid, expiry.Unix()}); e != nil && !errors.Is(e, errSocketDisconnected) {
			return nil, e
		}
	}
	return M{"sessionId": sid, "authToken": "bearer " + token, "userInfo": v}, nil
}
func (a *App) createUser(ctx context.Context, db DB, p M, ip string) (int64, error) {
	account := str(p["account"])
	if len(account) < 3 || len(account) > 32 || strings.ContainsAny(account, "\r\n\x00") {
		return 0, clientError{"账号长度须为 3–32 字节", 400}
	}
	name := str(p["realname"])
	if name == "" {
		name = account
	}
	if len([]rune(name)) > 100 {
		return 0, clientError{"昵称过长", 400}
	}
	if err := a.moderate(ctx, name, "nickname_detection"); err != nil {
		return 0, err
	}
	hash, e := hashPassword(str(p["password"]))
	if e != nil {
		return 0, clientError{e.Error(), 400}
	}
	userID, err := insert(ctx, db, a.t("user"), M{"account": account, "realname": name, "password": hash, "salt": "", "name_py": namePinyin(name), "email": str(p["email"]), "role": 0, "register_ip": ip, "create_time": time.Now().Unix(), "status": 1, "setting": "{}"})
	if err != nil {
		return 0, err
	}
	_, err = a.ensureInviteCode(ctx, db, userID)
	return userID, err
}
func (a *App) registerUser(r *request) (any, error) {
	s := a.config(r.ctx(), "sysInfo")
	if number(s["regtype"]) == 0 {
		return nil, r.fail("注册已关闭")
	}
	if number(s["regtype"]) == 2 && strings.TrimSpace(r.s("inviteCode")) == "" {
		return nil, r.fail("请输入邀请码")
	}
	if number(s["regauth"]) != 0 && !a.verifyCode(r.s("account"), "2", r.s("code")) {
		return nil, r.fail("验证码无效")
	}
	interval := number(s["registerInterval"])
	if interval < 5 {
		interval = 5
	}
	if !a.allow("register:"+a.clientIP(r.c), time.Duration(interval)*time.Second) {
		return nil, r.fail("注册过于频繁")
	}
	uid, e := a.createRegisteredUser(r.ctx(), r.p, a.clientIP(r.c))
	if e != nil {
		return nil, e
	}
	return M{"user_id": uid}, nil
}
func (a *App) verifyCode(account, kind, code string) bool {
	if len(code) != 6 {
		return false
	}
	key := "code:" + account + ":" + kind
	v := a.get(key, false)
	if v == nil {
		return false
	}
	if number(a.get("attempt:"+key, false)) >= 5 {
		a.get(key, true)
		return false
	}
	if !hmac.Equal([]byte(str(v)), []byte(code)) {
		a.recordFailure("attempt:" + key)
		return false
	}
	a.get(key, true)
	return true
}
func (a *App) sendCode(r *request) (any, error) {
	account := r.s("account")
	_, mailError := mail.ParseAddress(account)
	kind := r.s("type")
	if kind == "" {
		kind = "1"
	}
	if kind != "1" && kind != "2" && kind != "3" && kind != "4" {
		return nil, r.fail("验证码类型错误")
	}
	if !a.allow("code-ip:"+a.clientIP(r.c), 10*time.Second) || !a.allow("code-send:"+account, time.Minute) {
		return nil, r.fail("验证码发送过于频繁")
	}
	n, e := rand.Int(rand.Reader, big.NewInt(1000000))
	if e != nil {
		return nil, e
	}
	code := fmt.Sprintf("%06d", n.Int64())
	if mailError == nil {
		e = a.sendMail(account, "Imgo 验证码", "验证码："+code+"，5 分钟内有效。")
	} else {
		e = a.sendSMS(r.ctx(), account, kind, code)
	}
	if e != nil {
		return nil, e
	}
	a.put("code:"+account+":"+kind, code, 5*time.Minute)
	return nil, nil
}
func (a *App) sendMail(to, subject, body string) error {
	smtpHost, smtpUser, smtpPass, smtpFrom := a.cfg.SMTPHost, a.cfg.SMTPUser, a.cfg.SMTPPass, a.cfg.SMTPFrom
	security := "tls"
	if smtpHost == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conf := a.config(ctx, "smtp")
		if str(conf["host"]) != "" {
			smtpHost = net.JoinHostPort(str(conf["host"]), str(conf["port"]))
			smtpUser = str(conf["addr"])
			smtpFrom = smtpUser
			smtpPass = str(conf["pass"])
			security = str(conf["security"])
		}
	}
	if smtpHost == "" {
		return clientError{"尚未配置 SMTP 邮件服务", 503}
	}
	addr, e := mail.ParseAddress(to)
	if e != nil {
		return clientError{"邮箱格式错误", 400}
	}
	from, e := mail.ParseAddress(smtpFrom)
	if e != nil {
		return errors.New("SMTP_FROM 无效")
	}
	host, _, e := net.SplitHostPort(smtpHost)
	if e != nil {
		return e
	}
	var conn net.Conn
	if security == "ssl" || strings.HasSuffix(smtpHost, ":465") {
		conn, e = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", smtpHost, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, e = net.DialTimeout("tcp", smtpHost, 10*time.Second)
	}
	if e != nil {
		return e
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	client, e := smtp.NewClient(conn, host)
	if e != nil {
		return e
	}
	defer client.Close()
	if _, implicitTLS := conn.(*tls.Conn); implicitTLS {
		if smtpUser != "" {
			e = client.Auth(smtp.PlainAuth("", smtpUser, smtpPass, host))
		}
	} else {
		e = startSMTP(client, host, smtpUser, smtpPass)
	}
	if e != nil {
		return e
	}
	if e = client.Mail(from.Address); e != nil {
		return e
	}
	if e = client.Rcpt(addr.Address); e != nil {
		return e
	}
	w, e := client.Data()
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", from.Address, addr.Address, subject, body)
	if e != nil {
		return e
	}
	if e = w.Close(); e != nil {
		return e
	}
	return client.Quit()
}
func (a *App) api(r *request) (any, error) {
	ts := r.c.GetHeader("X-Im-TimeStamp")
	delta := time.Now().Unix() - number(ts)
	if a.cfg.APIID == "" || a.cfg.APISecret == "" || delta > 60 || delta < -60 || r.c.GetHeader("X-Im-AppId") != a.cfg.APIID {
		return nil, deny()
	}
	sum := fmt.Sprintf("%x", md5.Sum([]byte(a.cfg.APIID+ts+a.cfg.APISecret)))
	if !hmac.Equal([]byte(sum), []byte(r.c.GetHeader("X-Im-Sign"))) {
		return nil, deny()
	}
	switch action(r) {
	case "createuser":
		if r.s("password") == "" {
			r.p["password"] = randomID()[:30]
		}
		uid, e := a.createRegisteredUser(r.ctx(), r.p, a.clientIP(r.c))
		return M{"open_id": a.hashID(uid), "user_id": uid}, e
	case "login":
		var u M
		var e error
		if id := a.decodeID(r.s("open_id")); id > 0 {
			u, e = r.one("SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", id)
		} else {
			u, e = r.one("SELECT user_id FROM "+a.t("user")+" WHERE account=? AND status=1 AND delete_time=0", r.s("account"))
		}
		if e != nil {
			return nil, e
		}
		tok := randomID()
		a.put("sso:"+tok, u["user_id"], 5*time.Minute)
		return M{"token": tok, "url": a.url("/#/login?token=" + tok)}, nil
	}
	return nil, r.fail("未知操作")
}
func (a *App) revoke(ctx context.Context, uid int64) error {
	_, e := a.db.ExecContext(ctx, "DELETE FROM "+a.t("imgo_session")+" WHERE user_id=?", uid)
	a.hub.disconnectUser(uid)
	return e
}
func (a *App) signLink(value string) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.JWTKey))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

var _ = sql.ErrNoRows

func (a *App) recordFailure(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	v := a.cache[key]
	if time.Now().After(v.Expiry) {
		v.Value = int64(0)
	}
	v.Value = number(v.Value) + 1
	v.Expiry = time.Now().Add(5 * time.Minute)
	a.cache[key] = v
}
func (a *App) checkVersion(r *request) M {
	platform := "andriod"
	if r.n("platform") != 1101 {
		platform = "ios"
	}
	config := a.config(r.ctx(), "appVersion")
	v := obj(config[platform])
	release := number(v["release"])
	version := str(v["version"])
	if version == "" {
		version = "5.5.2"
	}
	download := ""
	if release > r.n("release") {
		download = str(v["url"])
		if download == "" {
			download = a.url("/downloadApp/" + platform)
		}
	}
	updateType := str(v["update_type"])
	if updateType == "" || r.n("setupPage") != 0 {
		updateType = "solicit"
	}
	return M{"versionName": version, "versionCode": release, "updateType": updateType, "versionInfo": str(v["update_info"]), "downloadUrl": download}
}
