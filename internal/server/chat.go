package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func chatKey(uid, to int64, group int64) string {
	if group == 1 {
		return fmt.Sprintf("group-%d", to)
	}
	if uid > to {
		uid, to = to, uid
	}
	return fmt.Sprintf("%d-%d", uid, to)
}
func target(p M) (int64, int64) {
	s := str(p["toContactId"])
	if strings.HasPrefix(s, "group-") {
		return number(strings.TrimPrefix(s, "group-")), 1
	}
	to := number(s)
	g := number(p["is_group"])
	if to == -1 {
		g = 3
	}
	return to, g
}
func (a *App) member(ctx context.Context, db DB, gid, uid int64) (M, error) {
	m, e := one(ctx, db, "SELECT gu.*,g.setting AS group_setting,g.owner_id FROM "+a.t("group_user")+" gu JOIN "+a.t("group")+" g ON g.group_id=gu.group_id WHERE gu.group_id=? AND gu.user_id=? AND gu.status=1 AND g.status=1 AND COALESCE(g.delete_time,0)=0", gid, uid)
	if e != nil {
		return nil, deny()
	}
	return m, nil
}
func (a *App) canChat(r *request, to, group int64, write bool) error {
	if group == 1 {
		m, e := a.member(r.ctx(), a.db, to, r.uid())
		if e != nil {
			return e
		}
		if write {
			setting := obj(m["group_setting"])
			role := number(m["role"])
			ns := number(setting["nospeak"])
			if number(m["no_speak_time"]) > time.Now().Unix() || (ns == 1 && role == 3) || (ns == 2 && role != 1) {
				return r.fail("当前处于禁言状态")
			}
			if number(a.config(r.ctx(), "chatInfo")["groupChat"]) == 0 {
				return r.fail("群聊已关闭")
			}
		}
		return nil
	}
	if group == 2 && !write && r.s("toContactId") == "admin_notice" {
		return nil
	}
	if group == 3 && to == -1 {
		return nil
	}
	if group != 0 || to < 1 {
		return r.fail("联系人无效")
	}
	u, e := r.one("SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", to)
	if e != nil || u == nil {
		return r.fail("联系人不存在")
	}
	if write {
		if number(a.config(r.ctx(), "chatInfo")["simpleChat"]) == 0 {
			return r.fail("私聊已关闭")
		}
		if number(a.config(r.ctx(), "sysInfo")["runMode"]) == 2 && number(r.user["role"]) == 0 && to != number(r.user["cs_uid"]) {
			if _, e = r.one("SELECT friend_id FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=? AND status=1", r.uid(), to); e != nil {
				return r.fail("请先添加好友")
			}
		}
	}
	return nil
}
func (a *App) im(r *request) (any, error) {
	switch action(r) {
	case "sendmessage":
		return a.sendMessage(r, r.p)
	case "getmessagelist":
		return a.messageList(r, false)
	case "getcontacts":
		return a.contacts(r, r.uid())
	case "getcontactinfo":
		id := r.s("id")
		if id == "" {
			id = r.s("toContactId")
		}
		return a.contact(r, id)
	case "getuserinfo":
		uid := r.n("user_id")
		if uid == 0 {
			uid = r.uid()
		}
		u, e := r.one("SELECT * FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", uid)
		if e != nil {
			return nil, e
		}
		v := a.safeUser(u)
		v["account"] = u["account"]
		v["create_time"] = u["create_time"]
		v["register_ip"] = u["register_ip"]
		v["reg_location"] = a.location(u["register_ip"])
		v["last_login_time"] = u["last_login_time"]
		v["last_login_ip"] = u["last_login_ip"]
		v["last_chat_time"] = u["last_chat_time"]
		v["last_chat_ip"] = u["last_chat_ip"]
		v["chat_location"] = a.location(u["last_chat_ip"])
		f, _ := r.one("SELECT friend_id,nickname,status FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=?", r.uid(), uid)
		v["friendInfo"] = f
		v["friend"] = f
		v["location"] = a.location(u["last_login_ip"])
		return v, nil
	case "searchuser", "userlist":
		return a.searchUsers(r)
	case "setting":
		setting := obj(r.p["setting"])
		if len(setting) == 0 {
			setting = pick(r.p, "hideMessageName", "hideMessageTime", "avatarCricle", "isVoice", "sendKey")
		}
		return nil, update(r.ctx(), a.db, a.t("user"), M{"setting": js(setting)}, "user_id=?", r.uid())
	case "updateuserinfo":
		p := pick(r.p, "realname", "email", "motto", "sex")
		for _, field := range []string{"realname", "motto"} {
			if str(p[field]) != "" {
				service := "nickname_detection"
				if field == "motto" {
					service = "comment_detection"
				}
				if e := a.moderate(r.ctx(), str(p[field]), service); e != nil {
					return nil, e
				}
			}
		}
		if name, ok := p["realname"]; ok {
			p["name_py"] = namePinyin(str(name))
		}
		if name := str(p["realname"]); len([]rune(name)) > 100 {
			return nil, r.fail("昵称过长")
		}
		return nil, update(r.ctx(), a.db, a.t("user"), p, "user_id=?", r.uid())
	case "editpassword":
		if !checkPassword(str(r.user["password"]), str(r.user["salt"]), r.s("originalPassword")) && !a.verifyCode(str(r.user["account"]), "3", r.s("code")) {
			return nil, r.fail("原密码或验证码错误")
		}
		password := r.s("password")
		if password == "" {
			password = r.s("newPassword")
		}
		hash, e := hashPassword(password)
		if e != nil {
			return nil, r.fail(e.Error())
		}
		if e = update(r.ctx(), a.db, a.t("user"), M{"password": hash, "salt": ""}, "user_id=?", r.uid()); e != nil {
			return nil, e
		}
		return nil, a.revoke(r.ctx(), r.uid())
	case "editaccount":
		if !a.verifyCode(str(r.user["account"]), "4", r.s("code")) || !a.verifyCode(r.s("account"), "4", r.s("newCode")) {
			return nil, r.fail("验证码错误")
		}
		if len(r.s("account")) < 3 || len(r.s("account")) > 32 {
			return nil, r.fail("账号格式错误")
		}
		if e := update(r.ctx(), a.db, a.t("user"), M{"account": r.s("account")}, "user_id=?", r.uid()); e != nil {
			return nil, e
		}
		return nil, a.revoke(r.ctx(), r.uid())
	case "setmsgisread":
		return nil, a.readMessages(r)
	case "undomessage", "removemessage", "delmessage":
		return nil, a.changeMessage(r)
	case "isnotice", "setchattop", "delchat":
		return nil, a.contactSetting(r)
	case "forwardmessage":
		return a.forward(r)
	case "sendtomsg":
		return a.rtc(r)
	case "readatmsg":
		to, g := target(r.p)
		if g != 1 {
			return nil, deny()
		}
		if e := a.canChat(r, to, g, false); e != nil {
			return nil, e
		}
		return nil, r.exec("UPDATE "+a.t("message")+" SET `at`=TRIM(BOTH ',' FROM REPLACE(CONCAT(',',COALESCE(`at`,''),','),?,',')) WHERE chat_identify=?", fmt.Sprintf(",%d,", r.uid()), chatKey(r.uid(), to, g))
	case "getadminnotice":
		m, e := r.one("SELECT * FROM " + a.t("message") + " WHERE chat_identify='admin_notice' AND status=1 ORDER BY msg_id DESC LIMIT 1")
		if e == sql.ErrNoRows {
			return M{}, nil
		}
		if e != nil {
			return nil, e
		}
		v := obj(m["extends"])
		v["create_time"] = m["create_time"]
		return v, nil
	}
	return nil, r.fail("未知操作")
}
func (a *App) sendMessage(r *request, p M) (any, error) { return a.sendMessageWithPublish(r, p, true) }
func (a *App) sendMessageWithPublish(r *request, p M, publish bool) (any, error) {
	to, g := target(p)
	if e := a.canChat(r, to, g, true); e != nil {
		return nil, e
	}
	typ := str(p["type"])
	if typ == "" {
		typ = "text"
	}
	allowed := map[string]bool{"text": true, "image": true, "file": true, "voice": true, "video": true, "webrtc": true}
	if !allowed[typ] {
		return nil, r.fail("消息类型无效")
	}
	content := str(p["content"])
	if typ == "text" {
		if e := a.moderate(r.ctx(), content, "chat_detection"); e != nil {
			return nil, e
		}
	}
	if typ == "text" {
		content = sanitizeText(content)
	}
	if content == "" || len([]rune(content)) > 16384 || (typ == "text" && len([]rune(content)) > 2048) {
		return nil, r.fail("消息内容为空或过长")
	}
	interval := number(a.config(r.ctx(), "chatInfo")["sendInterval"])
	if interval > 0 && !a.allow(fmt.Sprintf("send:%d", r.uid()), time.Duration(interval)*time.Second) {
		return nil, r.fail("发送过于频繁")
	}
	id := str(p["id"])
	if id == "" {
		id = randomID()[:32]
	}
	if len(id) > 36 {
		return nil, r.fail("消息 id 过长")
	}
	key := chatKey(r.uid(), to, g)
	enc, e := encryptContent(a.cfg.ChatKey, content)
	if e != nil {
		return nil, e
	}
	m := M{"id": id, "from_user": r.uid(), "to_user": to, "content": enc, "chat_identify": key, "type": typ, "is_group": g, "is_read": 0, "is_last": 1, "create_time": time.Now().Unix(), "status": 1, "at": "", "pid": 0, "file_id": 0, "file_cate": 0, "file_size": 0, "file_name": "", "extends": js(a.mediaExtensions(typ, obj(p["extends"])))}
	ats := []string{}
	mentions := ids(p["at"])
	if g == 1 {
		for _, v := range targetValues(p["at"]) {
			if str(v) == "0" {
				users, err := r.list("SELECT user_id FROM "+a.t("group_user")+" WHERE group_id=? AND user_id<>? AND status=1", to, r.uid())
				if err != nil {
					return nil, err
				}
				mentions = nil
				for _, u := range users {
					mentions = append(mentions, number(u["user_id"]))
				}
				break
			}
		}
	}
	for _, n := range mentions {
		ats = append(ats, fmt.Sprint(n))
	}
	m["at"] = strings.Join(ats, ",")
	if number(p["file_id"]) > 0 {
		f, err := r.one("SELECT * FROM "+a.t("file")+" WHERE file_id=? AND status=1", p["file_id"])
		if err != nil || !a.canReadFile(r.ctx(), r.uid(), f) {
			return nil, deny()
		}
		m["file_id"] = f["file_id"]
		m["file_cate"] = f["cate"]
		m["file_size"] = f["size"]
		m["file_name"] = str(f["name"]) + "." + str(f["ext"])
		m["content"], _ = encryptContent(a.cfg.ChatKey, a.mediaPath(str(f["src"])))
	} else if typ != "text" && typ != "webrtc" {
		return nil, r.fail("附件必须先上传")
	}
	if pid := number(p["pid"]); pid > 0 {
		if _, e = r.one("SELECT msg_id FROM "+a.t("message")+" WHERE msg_id=? AND chat_identify=? AND status=1", pid, key); e != nil {
			return nil, deny()
		}
		m["pid"] = pid
	}
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_chat_lock")+" (chat_identify) VALUES (?) ON DUPLICATE KEY UPDATE chat_identify=VALUES(chat_identify)", key); e != nil {
		return nil, e
	}
	if _, e = one(r.ctx(), tx, "SELECT chat_identify FROM "+a.t("imgo_chat_lock")+" WHERE chat_identify=? FOR UPDATE", key); e != nil {
		return nil, e
	}
	if existing, err := one(r.ctx(), tx, "SELECT * FROM "+a.t("message")+" WHERE id=? AND from_user=? AND chat_identify=?", id, r.uid(), key); err == nil {
		return a.serialize(r.ctx(), existing)
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	if _, e = tx.ExecContext(r.ctx(), "UPDATE "+a.t("message")+" SET is_last=0 WHERE chat_identify=? AND is_last=1", key); e != nil {
		return nil, e
	}
	mid, e := insert(r.ctx(), tx, a.t("message"), m)
	if e != nil {
		return nil, e
	}
	m["msg_id"] = mid
	if g == 1 {
		if _, e = tx.ExecContext(r.ctx(), "UPDATE "+a.t("group_user")+" SET unread=unread+1 WHERE group_id=? AND user_id<>? AND status=1", to, r.uid()); e != nil {
			return nil, e
		}
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	if chatTime, chatErr := a.recordChatIP(r.ctx(), r.uid(), a.clientIP(r.c)); chatErr == nil {
		a.hub.broadcast("userIPChanged", M{"id": r.uid(), "time": chatTime})
	} else {
		a.log.Warn("record chat IP failed", "user_id", r.uid(), "error", chatErr)
	}
	data, e := a.serialize(r.ctx(), m)
	if e != nil {
		return nil, e
	}
	if !publish {
		return data, nil
	}
	data["toUser"] = data["toContactId"]
	data["contactInfo"], _ = a.contact(r, str(data["toContactId"]))
	if g == 1 {
		a.groupEvent(r.ctx(), to, "group", data)
	} else if g == 0 {
		peer := M{}
		for k, v := range data {
			peer[k] = v
		}
		peer["toContactId"] = r.uid()
		peer["contactInfo"] = a.safeUser(r.user)
		obj(peer["contactInfo"])["lastContent"] = data["content"]
		obj(peer["contactInfo"])["lastSendTime"] = data["sendTime"]
		a.hub.send([]int64{to}, "simple", peer)
		a.hub.send([]int64{r.uid()}, "simple", data)
	} else {
		a.hub.send([]int64{r.uid()}, "simple", data)
	}
	return data, nil
}
func (a *App) serialize(ctx context.Context, m M) (M, error) {
	content, e := decryptContent(a.cfg.ChatKey, str(m["content"]))
	if e != nil {
		return nil, e
	}
	g := number(m["is_group"])
	var to any = m["to_user"]
	if g == 1 {
		to = "group-" + str(m["to_user"])
	}
	u, e := one(ctx, a.db, "SELECT user_id,realname,avatar,sex,motto,name_py,role FROM "+a.t("user")+" WHERE user_id=?", m["from_user"])
	if e == sql.ErrNoRows {
		u = M{"user_id": m["from_user"], "realname": "已注销用户"}
	} else if e != nil {
		return nil, e
	}
	if str(m["type"]) == "text" {
		content = sanitizeText(content)
	}
	download := ""
	if number(m["file_id"]) > 0 {
		download = a.mediaPath("/filedown/" + a.hashID(number(m["file_id"])))
		content = a.mediaPath(content)
	}
	role := int64(3)
	if g == 1 {
		if member, err := a.member(ctx, a.db, number(m["to_user"]), number(m["from_user"])); err == nil {
			role = number(member["role"])
		}
	}
	if kind := str(m["type"]); kind == "image" || kind == "file" || kind == "video" || kind == "voice" {
		content = a.mediaPath(content)
	}
	at := []string{}
	if str(m["at"]) != "" {
		at = strings.Split(str(m["at"]), ",")
	}
	extensions := obj(m["extends"])
	if str(m["type"]) == "video" {
		extensions["poster"] = a.videoPoster(str(extensions["poster"]))
	}
	return M{"role": role, "toUser": to, "msg_id": m["msg_id"], "id": m["id"], "status": "succeed", "type": m["type"], "sendTime": number(m["create_time"]) * 1000, "content": content, "preview": content, "download": download, "is_read": m["is_read"], "is_group": g, "at": at, "toContactId": to, "to_user": m["to_user"], "from_user": m["from_user"], "fromUser": a.safeUser(u), "file_id": m["file_id"], "file_cate": m["file_cate"], "fileName": m["file_name"], "fileSize": m["file_size"], "extUrl": "", "extends": extensions, "pid": m["pid"]}, nil
}
func (a *App) messageList(r *request, admin bool) (any, error) {
	to, g := target(r.p)
	key := chatKey(r.uid(), to, g)
	if r.s("toContactId") == "admin_notice" {
		key = "admin_notice"
		g = 2
	}
	where := "status=1"
	args := []any{}
	if admin {
		scope, err := a.adminScope(r.ctx(), r.user)
		if err != nil {
			return nil, err
		}
		if !scope.Global {
			predicate, params := a.messageScopePredicate(scope, a.t("message"))
			where += " AND " + predicate
			args = append(args, params...)
		}
	}
	if !admin {
		if e := a.canChat(r, to, g, false); e != nil {
			return nil, e
		}
		where += " AND chat_identify=? AND NOT FIND_IN_SET(?,COALESCE(del_user,''))"
		args = append(args, key, r.uid())
		if g == 1 {
			member, e := a.member(r.ctx(), a.db, to, r.uid())
			if e != nil {
				return nil, e
			}
			if v, ok := obj(member["group_setting"])["history"]; ok && number(v) == 0 {
				where += " AND create_time>=?"
				args = append(args, member["create_time"])
			}
		}
	} else {
		if r.n("user_id") > 0 {
			where += " AND chat_identify=?"
			args = append(args, chatKey(r.n("user_id"), to, g))
		}
		if r.n("is_group") > 0 {
			where += " AND is_group=?"
			args = append(args, r.n("is_group")-1)
		}
	}
	if t := r.s("type"); t != "" {
		if t == "all" {
			where += " AND type<>'event'"
		} else {
			where += " AND type=?"
			args = append(args, t)
		}
	}
	keyword := r.s("keywords")
	if keyword != "" {
		where += " AND type='text'"
		if a.cfg.ChatKey == "" {
			where += " AND content LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
	}
	if r.n("last_id") > 0 {
		where += " AND msg_id<?"
		args = append(args, r.n("last_id"))
	}
	if r.n("is_at") > 0 {
		where += " AND (FIND_IN_SET(?,`at`) OR FIND_IN_SET('0',`at`))"
		args = append(args, r.uid())
	}
	var list []M
	var e error
	if keyword != "" && a.cfg.ChatKey != "" {
		list, e = a.searchEncrypted(r, where, args, keyword)
		if e != nil {
			return nil, e
		}
	} else {
		count, e := r.one("SELECT COUNT(*) AS n FROM "+a.t("message")+" WHERE "+where, args...)
		if e != nil {
			return nil, e
		}
		r.count = number(count["n"])
		limit, offset := r.pagination()
		list, e = r.list("SELECT * FROM "+a.t("message")+" WHERE "+where+" ORDER BY msg_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
		if e != nil {
			return nil, e
		}

	}
	out := []M{}
	for _, m := range list {
		v, e := a.serialize(r.ctx(), m)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	if r.s("type") == "" && !admin {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
		if e = a.readMessages(r); e != nil {
			return nil, e
		}
	}
	return out, nil
}
func (a *App) readMessages(r *request) error {
	to, g := target(r.p)
	if r.s("toContactId") == "admin_notice" {
		return nil
	}
	if e := a.canChat(r, to, g, false); e != nil {
		return e
	}
	if g == 1 {
		return r.exec("UPDATE "+a.t("group_user")+" SET unread=0 WHERE group_id=? AND user_id=?", to, r.uid())
	}
	if g == 3 {
		return nil
	}
	e := r.exec("UPDATE "+a.t("message")+" SET is_read=1 WHERE chat_identify=? AND to_user=? AND is_group=0", chatKey(r.uid(), to, g), r.uid())
	if e == nil {
		a.hub.send([]int64{to}, "readAll", M{"toContactId": r.uid()})
	}
	return e
}
func (a *App) accessibleMessage(r *request) (M, error) {
	m, e := r.one("SELECT * FROM "+a.t("message")+" WHERE id=? AND status=1", r.s("id"))
	if r.n("msg_id") > 0 {
		m, e = r.one("SELECT * FROM "+a.t("message")+" WHERE msg_id=? AND status=1", r.n("msg_id"))
	}
	if e != nil {
		return nil, e
	}
	g := number(m["is_group"])
	if g == 1 {
		if _, e = a.member(r.ctx(), a.db, number(m["to_user"]), r.uid()); e != nil {
			return nil, e
		}
	} else if number(m["from_user"]) != r.uid() && (g != 0 || number(m["to_user"]) != r.uid()) {
		return nil, deny()
	}
	return m, nil
}
func (a *App) changeMessage(r *request) error {
	m, e := a.accessibleMessage(r)
	if e != nil {
		return e
	}
	kind := action(r)
	if kind == "removemessage" {
		return r.exec("UPDATE "+a.t("message")+" SET del_user=CONCAT_WS(',',NULLIF(del_user,''),?) WHERE msg_id=? AND NOT FIND_IN_SET(?,COALESCE(del_user,''))", r.uid(), m["msg_id"], r.uid())
	}
	own := number(m["from_user"]) == r.uid()
	if !own {
		if kind != "undomessage" || number(m["is_group"]) != 1 {
			return deny()
		}
		member, err := a.member(r.ctx(), a.db, number(m["to_user"]), r.uid())
		if err != nil || number(member["role"]) > 2 {
			return deny()
		}
	}
	if kind == "undomessage" {
		ttl := number(a.config(r.ctx(), "chatInfo")["redoTime"])
		if ttl <= 0 {
			ttl = 120
		}
		if own && time.Now().Unix()-number(m["create_time"]) > ttl {
			return r.fail("已超过撤回时限")
		}
		content, _ := encryptContent(a.cfg.ChatKey, "撤回了一条消息")
		e = update(r.ctx(), a.db, a.t("message"), M{"type": "event", "content": content, "is_undo": 1, "at": ""}, "msg_id=?", m["msg_id"])
	} else {
		if number(a.config(r.ctx(), "chatInfo")["dbDelMsg"]) == 0 {
			return deny()
		}
		e = update(r.ctx(), a.db, a.t("message"), M{"status": 0}, "msg_id=?", m["msg_id"])
	}
	if e != nil {
		return e
	}
	event := "undoMessage"
	if kind == "delmessage" {
		event = "delMessage"
	}
	data := M{"id": m["id"], "content": "撤回了一条消息", "toContactId": m["to_user"]}
	a.messageEvent(r.ctx(), m, event, data)
	return nil
}
func (a *App) messageEvent(ctx context.Context, m M, event string, data any) {
	if number(m["is_group"]) == 1 {
		a.groupEvent(ctx, number(m["to_user"]), event, data)
	} else {
		a.hub.send([]int64{number(m["from_user"]), number(m["to_user"])}, event, data)
	}
}
func (a *App) forward(r *request) (any, error) {
	users := targetValues(r.p["user_ids"])
	if len(users) == 0 || len(users) > 5 {
		return nil, r.fail("每次最多转发给 5 个联系人")
	}
	m, e := a.accessibleMessage(r)
	if e != nil {
		return nil, e
	}
	if str(m["type"]) == "event" || str(m["type"]) == "webrtc" || number(m["is_undo"]) != 0 {
		return nil, r.fail("此消息不能转发")
	}
	content, e := decryptContent(a.cfg.ChatKey, str(m["content"]))
	if e != nil {
		return nil, e
	}
	out := []any{}
	for _, target := range users {
		p := M{"toContactId": target, "type": m["type"], "content": content, "file_id": m["file_id"]}
		v, e := a.sendMessage(r, p)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}
func targetValues(v any) []string {
	out := []string{}
	switch x := v.(type) {
	case []any:
		for _, n := range x {
			out = append(out, str(n))
		}
	case []string:
		out = x
	case string:
		out = strings.Split(x, ",")
	}
	return out
}
