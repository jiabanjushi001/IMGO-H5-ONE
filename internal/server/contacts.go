package server

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (a *App) searchUsers(r *request) (any, error) {
	where := "status=1 AND delete_time=0 AND user_id<>?"
	args := []any{r.uid()}
	if action(r) == "searchuser" {
		where += " AND account=?"
		account := r.s("account")
		if account == "" {
			account = r.s("keywords")
		}
		args = append(args, account)
	} else if k := r.s("keywords"); k != "" {
		where += " AND (account LIKE ? OR realname LIKE ? OR name_py LIKE ?)"
		args = append(args, "%"+k+"%", "%"+k+"%", "%"+k+"%")
	}
	n, e := r.one("SELECT COUNT(*) n FROM "+a.t("user")+" WHERE "+where, args...)
	if e != nil {
		return nil, e
	}
	r.count = number(n["n"])
	limit, offset := r.pagination()
	list, e := r.list("SELECT user_id,realname,avatar,sex,motto,name_py,role FROM "+a.t("user")+" WHERE "+where+" ORDER BY user_id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if e != nil {
		return nil, e
	}
	out := []M{}
	for _, u := range list {
		out = append(out, a.safeUser(u))
	}
	return out, nil
}
func (a *App) contact(r *request, id string) (M, error) {
	group := strings.HasPrefix(id, "group-")
	uid := number(id)
	key := chatKey(r.uid(), uid, 0)
	joinedAfter := int64(0)
	v := M{"id": uid, "lastContent": "", "unread": 0, "lastSendTime": 0, "is_group": 0, "is_notice": 1, "is_top": 0, "is_online": 0, "is_at": 0, "setting": M{}, "type": "text", "location": "", "index": "#"}
	if group {
		gid := number(strings.TrimPrefix(id, "group-"))
		m, e := a.member(r.ctx(), a.db, gid, r.uid())
		if e != nil {
			return nil, e
		}
		g, e := r.one("SELECT * FROM "+a.t("group")+" WHERE group_id=?", gid)
		if e != nil {
			return nil, e
		}
		v["id"] = id
		v["is_group"] = 1
		v["displayName"] = g["name"]
		v["realname"] = g["name"]
		v["name_py"] = g["name_py"]
		v["avatar"] = a.groupAvatarURL(gid)
		v["setting"] = obj(g["setting"])
		if history, ok := obj(g["setting"])["history"]; ok && number(history) == 0 {
			joinedAfter = number(m["create_time"])
		}
		n, err := r.one("SELECT COUNT(*) n FROM "+a.t("message")+" WHERE chat_identify=? AND status=1 AND (FIND_IN_SET(?,`at`) OR FIND_IN_SET('0',`at`)) AND create_time>=?", id, r.uid(), joinedAfter)
		if err != nil {
			return nil, err
		}
		v["is_at"] = n["n"]
		v["role"] = m["role"]
		v["owner_id"] = g["owner_id"]
		v["unread"] = m["unread"]
		v["is_notice"] = m["is_notice"]
		v["is_top"] = m["is_top"]
		v["index"] = "[2]群聊"
		key = id
	} else if id == "admin_notice" || id == "-1" {
		v["id"] = id
		v["user_id"] = id
		v["displayName"] = "文件传输助手"
		v["realname"] = "文件传输助手"
		v["is_group"] = 3
		v["index"] = "[1]系统消息"
		v["avatar"] = a.mediaPath("/avatar/助手/120/1")
		if id == "admin_notice" {
			v["displayName"] = "系统公告"
			v["realname"] = "系统公告"
			v["is_group"] = 2
			key = id
		}
	} else {
		u, e := r.one("SELECT user_id,realname,avatar,name_py,last_login_ip FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", uid)
		if e != nil {
			return nil, e
		}
		for k, val := range a.safeUser(u) {
			v[k] = val
		}
		v["is_online"] = a.hub.online(uid)
		v["location"] = a.location(u["last_login_ip"])
		v["index"] = nameIndex(str(u["name_py"]))
		f, err := r.one("SELECT * FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=? AND status=1", r.uid(), uid)
		if err == nil {
			v["is_top"] = f["is_top"]
			v["is_notice"] = f["is_notice"]
			if str(f["nickname"]) != "" {
				v["displayName"] = f["nickname"]
			}
		} else if err != sql.ErrNoRows {
			return nil, err
		}
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("message")+" WHERE to_user=? AND from_user=? AND is_group=0 AND is_read=0 AND status=1", r.uid(), uid)
		if e != nil {
			return nil, e
		}
		v["unread"] = n["n"]
	}
	last, e := r.one("SELECT content,create_time,type FROM "+a.t("message")+" WHERE chat_identify=? AND status=1 AND NOT FIND_IN_SET(?,COALESCE(del_user,'')) AND create_time>=? ORDER BY msg_id DESC LIMIT 1", key, r.uid(), joinedAfter)
	if e == nil {
		v["lastContent"], e = decryptContent(a.cfg.ChatKey, str(last["content"]))
		if e != nil {
			return nil, e
		}
		v["lastSendTime"] = number(last["create_time"]) * 1000
		v["type"] = last["type"]
	} else if e != sql.ErrNoRows {
		return nil, e
	}
	return v, nil
}
func (a *App) contacts(r *request, uid int64) (any, error) {
	where := "u.status=1 AND u.delete_time=0 AND u.user_id<>?"
	args := []any{uid, uid}
	if number(a.config(r.ctx(), "sysInfo")["runMode"]) == 2 {
		where += " AND (f.status=1 OR u.user_id=?)"
		args = append(args, r.user["cs_uid"])
	}
	users, e := r.list("SELECT u.user_id,u.realname,u.avatar,u.name_py,u.last_login_ip,f.nickname,f.is_notice,f.is_top FROM "+a.t("user")+" u LEFT JOIN "+a.t("friend")+" f ON f.friend_user_id=u.user_id AND f.create_user=? WHERE "+where+" ORDER BY u.user_id", args...)
	if e != nil {
		return nil, e
	}
	unread, e := r.list("SELECT from_user,COUNT(*) n FROM "+a.t("message")+" WHERE to_user=? AND is_group=0 AND is_read=0 AND status=1 GROUP BY from_user", uid)
	if e != nil {
		return nil, e
	}
	unreadMap := map[int64]int64{}
	for _, m := range unread {
		unreadMap[number(m["from_user"])] = number(m["n"])
	}
	groups, e := r.list("SELECT g.*,gu.role,gu.unread,gu.is_notice,gu.is_top,gu.create_time AS joined_at FROM "+a.t("group")+" g JOIN "+a.t("group_user")+" gu ON gu.group_id=g.group_id WHERE gu.user_id=? AND gu.status=1 AND g.status=1 AND COALESCE(g.delete_time,0)=0", uid)
	if e != nil {
		return nil, e
	}
	latest, e := r.list("SELECT m.* FROM "+a.t("message")+" m JOIN (SELECT MAX(msg_id) AS last_id FROM "+a.t("message")+" WHERE status=1 AND NOT FIND_IN_SET(?,COALESCE(del_user,'')) AND ((is_group=0 AND (from_user=? OR to_user=?)) OR (is_group=3 AND from_user=?) OR chat_identify='admin_notice' OR (is_group=1 AND to_user IN (SELECT group_id FROM "+a.t("group_user")+" WHERE user_id=? AND status=1))) GROUP BY chat_identify) latest ON latest.last_id=m.msg_id", uid, uid, uid, uid, uid)
	if e != nil {
		return nil, e
	}
	lastMap := map[string]M{}
	for _, m := range latest {
		lastMap[str(m["chat_identify"])] = m
	}
	makeContact := func(id any, name any, avatar string, key string) (M, error) {
		v := M{"id": id, "user_id": id, "displayName": name, "realname": name, "avatar": avatar, "lastContent": "", "lastSendTime": 0, "unread": 0, "is_group": 0, "is_top": 0, "is_notice": 1, "is_online": 0, "is_at": 0, "setting": M{}, "type": "text", "index": "#", "location": ""}
		if m := lastMap[key]; m != nil {
			content, err := decryptContent(a.cfg.ChatKey, str(m["content"]))
			if err != nil {
				return nil, err
			}
			v["lastContent"] = sanitizeText(content)
			v["lastSendTime"] = number(m["create_time"]) * 1000
			v["type"] = m["type"]
		}
		return v, nil
	}
	out := []M{}
	for _, u := range users {
		id := number(u["user_id"])
		v, e := makeContact(id, u["realname"], a.userAvatar(u), chatKey(uid, id, 0))
		if e != nil {
			return nil, e
		}
		v["name_py"] = u["name_py"]
		v["index"] = nameIndex(str(u["name_py"]))
		v["location"] = a.location(u["last_login_ip"])
		v["unread"] = unreadMap[id]
		v["is_online"] = a.hub.online(id)
		if u["is_notice"] != nil {
			v["is_notice"] = u["is_notice"]
		}
		if u["is_top"] != nil {
			v["is_top"] = u["is_top"]
		}
		if str(u["nickname"]) != "" {
			v["displayName"] = u["nickname"]
		}
		out = append(out, v)
	}
	for _, g := range groups {
		id := "group-" + str(g["group_id"])
		v, e := makeContact(id, g["name"], a.groupAvatarURL(number(g["group_id"])), id)
		if e != nil {
			return nil, e
		}
		v["is_group"] = 1
		v["name_py"] = g["name_py"]
		v["owner_id"] = g["owner_id"]
		v["role"] = g["role"]
		v["unread"] = g["unread"]
		v["is_notice"] = g["is_notice"]
		v["is_top"] = g["is_top"]
		v["setting"] = obj(g["setting"])
		after := int64(0)
		if history, ok := obj(g["setting"])["history"]; ok && number(history) == 0 {
			after = number(g["joined_at"])
		}
		n, err := r.one("SELECT COUNT(*) n FROM "+a.t("message")+" WHERE chat_identify=? AND status=1 AND (FIND_IN_SET(?,`at`) OR FIND_IN_SET('0',`at`)) AND create_time>=?", id, uid, after)
		if err != nil {
			return nil, err
		}
		v["is_at"] = n["n"]
		v["index"] = "[2]群聊"
		if history, ok := obj(g["setting"])["history"]; ok && number(history) == 0 {
			if m := lastMap[id]; m != nil && number(m["create_time"]) < number(g["joined_at"]) {
				v["lastContent"] = ""
				v["lastSendTime"] = 0
			}
		}
		out = append(out, v)
	}
	for _, special := range []struct {
		id, name string
		group    int
	}{{"admin_notice", "系统公告", 2}, {"-1", "文件传输助手", 3}} {
		key := special.id
		if special.group == 3 {
			key = chatKey(uid, -1, 3)
		}
		v, e := makeContact(special.id, special.name, a.mediaPath("/avatar/助手/120/1"), key)
		if e != nil {
			return nil, e
		}
		if special.group == 3 && number(v["lastSendTime"]) == 0 {
			v["lastSendTime"] = time.Now().UnixMilli()
			v["lastContent"] = "传输你的文件"
		}
		v["is_group"] = special.group
		v["index"] = "[1]系统消息"
		out = append(out, v)
	}
	n, e := r.one("SELECT COUNT(*) n FROM "+a.t("friend")+" WHERE friend_user_id=? AND status=2 AND is_invite=1", uid)
	if e != nil {
		return nil, e
	}
	r.count = number(n["n"])
	return out, nil
}

func (a *App) contactSetting(r *request) error {
	id := r.s("id")
	if id == "" {
		id = r.s("toContactId")
	}
	p := M{"toContactId": id, "is_group": r.n("is_group")}
	to, g := target(p)
	if e := a.canChat(r, to, g, false); e != nil {
		return e
	}
	if action(r) == "delchat" {
		key := chatKey(r.uid(), to, g)
		return r.exec("UPDATE "+a.t("message")+" SET del_user=CONCAT_WS(',',NULLIF(del_user,''),?) WHERE chat_identify=? AND NOT FIND_IN_SET(?,COALESCE(del_user,''))", r.uid(), key, r.uid())
	}
	col := "is_notice"
	event := "setIsNotice"
	if action(r) == "setchattop" {
		col = "is_top"
		event = "setChatTop"
	}
	flag := number(r.p[col])
	if flag != 0 && flag != 1 {
		return r.fail("参数无效")
	}
	var e error
	if g == 1 {
		e = update(r.ctx(), a.db, a.t("group_user"), M{col: flag}, "group_id=? AND user_id=?", to, r.uid())
	} else {
		_, e = r.one("SELECT friend_id FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=?", r.uid(), to)
		if e == sql.ErrNoRows {
			_, e = insert(r.ctx(), a.db, a.t("friend"), M{"create_user": r.uid(), "friend_user_id": to, "status": 1, "create_time": time.Now().Unix(), col: flag})
		} else if e == nil {
			e = update(r.ctx(), a.db, a.t("friend"), M{col: flag}, "create_user=? AND friend_user_id=?", r.uid(), to)
		}
	}
	if e == nil {
		a.hub.send([]int64{r.uid()}, event, M{"id": id, "is_group": g, col: flag})
	}
	return e
}
func (a *App) friend(r *request) (any, error) {
	switch action(r) {
	case "getapplymsg":
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("friend")+" WHERE friend_user_id=? AND status=2 AND is_invite=1", r.uid())
		if e != nil {
			return nil, e
		}
		return n["n"], nil
	case "index":
		col := "friend_user_id"
		if r.n("is_mine") == 1 {
			col = "create_user"
		}
		where := col + "=? AND is_invite=1"
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("friend")+" WHERE "+where, r.uid())
		if e != nil {
			return nil, e
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		list, e := r.list("SELECT * FROM "+a.t("friend")+" WHERE "+where+" ORDER BY friend_id DESC LIMIT ? OFFSET ?", r.uid(), limit, offset)
		if e != nil {
			return nil, e
		}
		for _, f := range list {
			for field, key := range map[string]string{"create_user": "create_user_info", "friend_user_id": "user_id_info"} {
				u, e := r.one("SELECT user_id,realname,avatar FROM "+a.t("user")+" WHERE user_id=?", f[field])
				if e == nil {
					f[key] = a.safeUser(u)
				}
			}
			f["is_group"] = 0
		}
		return list, nil
	case "add":
		uid := r.n("user_id")
		if uid <= 0 || uid == r.uid() {
			return nil, r.fail("不能添加自己")
		}
		if _, e := r.one("SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", uid); e != nil {
			return nil, e
		}
		limit := number(r.user["friend_limit"])
		if limit != 0 {
			n, e := r.one("SELECT COUNT(*) n FROM "+a.t("friend")+" WHERE create_user=? AND status=1", r.uid())
			if e != nil {
				return nil, e
			}
			if limit < 0 || number(n["n"]) >= limit {
				return nil, r.fail("已达到好友上限")
			}
		}
		tx, e := a.db.BeginTx(r.ctx(), nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		if _, e = one(r.ctx(), tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? FOR UPDATE", r.uid()); e != nil {
			return nil, e
		}
		f, e := one(r.ctx(), tx, "SELECT friend_id,status FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=?", r.uid(), uid)
		if e == nil && number(f["status"]) == 1 {
			return nil, r.fail("已经是好友")
		}
		p := M{"create_user": r.uid(), "friend_user_id": uid, "status": 2, "is_invite": 1, "remark": r.s("remark"), "create_time": time.Now().Unix()}
		var id int64
		if e == sql.ErrNoRows {
			id, e = insert(r.ctx(), tx, a.t("friend"), p)
		} else if e == nil {
			id = number(f["friend_id"])
			e = update(r.ctx(), tx, a.t("friend"), p, "friend_id=?", id)
		}
		if e != nil {
			return nil, e
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
		a.hub.send([]int64{uid}, "friendApply", M{"friend_id": id, "user_id": r.uid()})
		return M{"friend_id": id}, nil
	case "update":
		f, e := r.one("SELECT * FROM "+a.t("friend")+" WHERE friend_id=? AND friend_user_id=? AND status=2", r.n("friend_id"), r.uid())
		if e != nil {
			return nil, e
		}
		status := r.n("status")
		if status != 0 && status != 1 {
			return nil, r.fail("状态无效")
		}
		tx, e := a.db.BeginTx(r.ctx(), nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		if e = update(r.ctx(), tx, a.t("friend"), M{"status": status, "update_time": time.Now().Unix()}, "friend_id=? AND status=2", f["friend_id"]); e != nil {
			return nil, e
		}
		if status == 1 {
			reverse, err := one(r.ctx(), tx, "SELECT friend_id FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=?", r.uid(), f["create_user"])
			if err == sql.ErrNoRows {
				_, e = insert(r.ctx(), tx, a.t("friend"), M{"create_user": r.uid(), "friend_user_id": f["create_user"], "status": 1, "create_time": time.Now().Unix()})
			} else if err == nil {
				e = update(r.ctx(), tx, a.t("friend"), M{"status": 1}, "friend_id=?", reverse["friend_id"])
			} else {
				e = err
			}
			if e != nil {
				return nil, e
			}
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
		if status == 1 {
			contact, e := a.contact(r, str(f["create_user"]))
			if e != nil {
				return nil, e
			}
			a.hub.send([]int64{r.uid()}, "appendContact", contact)
			a.hub.send([]int64{number(f["create_user"])}, "appendContact", a.safeUser(r.user))
		}
		return nil, nil
	case "del":
		uid := r.n("id")
		if uid == 0 {
			uid = r.n("user_id")
		}
		return nil, r.exec("DELETE FROM "+a.t("friend")+" WHERE (create_user=? AND friend_user_id=?) OR (create_user=? AND friend_user_id=?)", r.uid(), uid, uid, r.uid())
	case "setnickname":
		if len([]rune(r.s("nickname"))) > 100 {
			return nil, r.fail("备注过长")
		}
		if uid := r.n("user_id"); uid > 0 {
			return nil, update(r.ctx(), a.db, a.t("friend"), M{"nickname": r.s("nickname")}, "friend_user_id=? AND create_user=? AND status=1", uid, r.uid())
		}
		return nil, update(r.ctx(), a.db, a.t("friend"), M{"nickname": r.s("nickname")}, "friend_id=? AND create_user=? AND status=1", r.n("friend_id"), r.uid())
	}
	return nil, r.fail("未知操作")
}
func (a *App) rtc(r *request) (any, error) {
	to, g := target(r.p)
	if g != 0 {
		return nil, r.fail("音视频通话只支持私聊")
	}
	if e := a.canChat(r, to, g, true); e != nil {
		return nil, e
	}
	event := r.s("event")
	if event == "" {
		event = "calling"
	}
	allowed := map[string]bool{"calling": true, "acceptRtc": true, "hangup": true, "offer": true, "answer": true, "candidate": true, "iceCandidate": true}
	if !allowed[event] {
		return nil, r.fail("通话事件无效")
	}
	ext := pick(r.p, "type", "status", "event", "callTime", "sdp", "iceCandidate", "code", "isMobile")
	ext["event"] = event
	if number(ext["code"]) == 0 {
		ext["code"] = 901
	}
	if event == "calling" {
		ext["status"] = 3
	}
	content := "语音通话"
	if r.n("type") == 1 {
		content = "视频通话"
	}
	switch number(ext["code"]) {
	case 902:
		content = "已取消"
	case 903:
		content = "已拒绝"
	case 905:
		content = "未接通"
	case 906:
		content = fmt.Sprintf("通话时长 %02d:%02d", r.n("callTime")/60, r.n("callTime")%60)
	case 907:
		content = "对方忙线"
	case 908:
		content = "已在其他端处理"
	}
	if event == "acceptRtc" {
		content = "已接听"
	}
	if event == "iceCandidate" {
		content = "交换连接信息"
	}
	if a.hub.online(to) == 0 && event == "calling" {
		ext["code"] = 907
		ext["event"] = "busy"
		data := M{"id": r.s("id"), "msg_id": 0, "type": "webrtc", "content": "对方不在线", "toContactId": to, "fromUser": a.safeUser(r.user), "extends": ext, "sendTime": time.Now().UnixMilli(), "is_group": 0, "is_read": 0, "at": []any{}, "status": "succeed"}
		a.hub.send([]int64{r.uid()}, "webrtc", data)
		return data, nil
	}
	var data M
	if event == "calling" && a.hub.online(to) > 0 {
		p := M{"id": r.s("id"), "toContactId": to, "type": "webrtc", "content": content, "extends": ext}
		v, e := a.sendMessageWithPublish(r, p, false)
		if e != nil {
			return nil, e
		}
		data = v.(M)
	} else {
		m, e := a.accessibleMessage(r)
		if e != nil {
			return nil, e
		}
		if str(m["type"]) != "webrtc" || str(m["chat_identify"]) != chatKey(r.uid(), to, 0) {
			return nil, deny()
		}
		if event == "hangup" {
			enc, err := encryptContent(a.cfg.ChatKey, content)
			if err != nil {
				return nil, err
			}
			if e = update(r.ctx(), a.db, a.t("message"), M{"extends": js(ext), "content": enc}, "msg_id=?", m["msg_id"]); e != nil {
				return nil, e
			}
		}
		data = M{"id": m["id"], "msg_id": m["msg_id"], "type": "webrtc", "sendTime": time.Now().UnixMilli(), "fromUser": a.safeUser(r.user), "is_group": 0, "status": "succeed"}
	}
	data["extends"] = ext
	data["content"] = content
	data["toUser"] = to
	data["is_read"] = 0
	data["at"] = []any{}
	data["toContactId"] = r.uid()
	a.hub.send([]int64{to}, "webrtc", data)
	data["toContactId"] = to
	if event == "calling" || event == "hangup" || event == "acceptRtc" {
		own := M{}
		for k, v := range data {
			own[k] = v
		}
		ownExt := M{}
		for k, v := range ext {
			ownExt[k] = v
		}
		if event != "calling" {
			ownExt["event"] = "otherOpt"
		}
		own["extends"] = ownExt
		own["contactInfo"], _ = a.contact(r, fmt.Sprint(to))
		a.hub.send([]int64{r.uid()}, "webrtc", own)
	}
	return data, nil
}

var _ = fmt.Sprint

func nameIndex(s string) string {
	s = strings.ToUpper(s)
	if len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z' {
		return s[:1]
	}
	return "#"
}
