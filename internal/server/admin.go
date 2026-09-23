package server

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (a *App) manageUser(r *request) (any, error) {
	uid := r.n("user_id")
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	if !scope.Global {
		if action(r) == "setrole" {
			return nil, deny()
		}
		if action(r) != "index" && action(r) != "add" {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
		}
	}
	whereUser := "user_id=? AND delete_time=0"
	if r.uid() != 1 && action(r) != "index" && action(r) != "detail" && action(r) != "checkinhistory" && action(r) != "add" {
		if action(r) == "setrole" {
			return nil, deny()
		}
		if _, e := r.one("SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND role=0 AND user_id<>1 AND delete_time=0", uid); e != nil {
			return nil, deny()
		}
		whereUser += " AND role=0 AND user_id<>1"
	}
	switch action(r) {
	case "index":
		where := "delete_time=0"
		args := []any{}
		from := a.t("user")
		column := ""
		referralScope := strings.TrimSpace(r.s("referral_scope"))
		if referralScope != "" && referralScope != "direct" && referralScope != "all" {
			return nil, r.fail("下级范围无效")
		}
		if !scope.Global {
			from += " u"
			column = "u."
			predicate, scopeArgs := scope.userPredicate("u")
			where = "u.delete_time=0 AND " + predicate
			args = append(args, scopeArgs...)
			if referralScope == "direct" {
				where += " AND EXISTS (SELECT 1 FROM " + a.t("imgo_referral_path") + " scope_depth WHERE scope_depth.ancestor_user_id=? AND scope_depth.descendant_user_id=u.user_id AND scope_depth.depth=1)"
				args = append(args, scope.AgentUserID)
			}
		}
		if username := strings.TrimSpace(r.s("keywords")); username != "" {
			match := column + "account=?"
			exactWhere := "delete_time=0 AND account=?"
			exactArgs := []any{username}
			if !scope.Global {
				exactWhere = where + " AND u.account=?"
				exactArgs = append(append([]any{}, args...), username)
			}
			exact, err := r.one("SELECT COUNT(*) n FROM "+from+" WHERE "+exactWhere, exactArgs...)
			if err != nil {
				return nil, err
			}
			value := username
			if number(exact["n"]) == 0 {
				match = column + "account LIKE ? ESCAPE '!'"
				value = "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(username) + "%"
			}
			if !scope.Global || referralScope == "" {
				where += " AND " + match
			} else {
				where += " AND user_id IN (SELECT p.descendant_user_id FROM " + a.t("imgo_referral_path") + " p JOIN " + a.t("user") + " inviter ON inviter.user_id=p.ancestor_user_id WHERE inviter." + match + " AND inviter.delete_time=0"
				if referralScope == "direct" {
					where += " AND p.depth=1"
				}
				where += ")"
			}
			args = append(args, value)
		} else if scope.Global && referralScope != "" {
			return nil, r.fail("请输入用户名后查询下级")
		}
		n, e := r.one("SELECT COUNT(*) n FROM "+from+" WHERE "+where, args...)
		if e != nil {
			return nil, e
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		order := "user_id"
		if candidate := r.s("order_field"); candidate == "create_time" || candidate == "last_login_time" || candidate == "login_count" {
			order = candidate
		}
		direction := "DESC"
		if r.n("order_type") == 1 {
			direction = "ASC"
		}
		selectColumns := "*"
		if !scope.Global {
			selectColumns = "u.*"
		}
		list, e := r.list("SELECT "+selectColumns+" FROM "+from+" WHERE "+where+" ORDER BY "+column+order+" "+direction+" LIMIT ? OFFSET ?", append(args, limit, offset)...)
		if e != nil {
			return nil, e
		}
		if e = a.memberCheckInSummaries(r, list); e != nil {
			return nil, e
		}
		if e = a.memberReferralSummaries(r, list); e != nil {
			return nil, e
		}
		if e = a.attachAdminRoleNames(r, list); e != nil {
			return nil, e
		}
		for _, u := range list {
			delete(u, "password")
			delete(u, "salt")
			u["avatar"] = a.userAvatar(u)
			u["location"] = a.location(u["last_login_ip"])
			u["reg_location"] = a.location(u["register_ip"])
		}
		return list, nil
	case "detail":
		u, e := r.one("SELECT * FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid)
		if e != nil {
			return nil, e
		}
		if e = a.memberCheckInSummaries(r, []M{u}); e != nil {
			return nil, e
		}
		if e = a.memberReferralSummaries(r, []M{u}); e != nil {
			return nil, e
		}
		if e = a.attachAdminRoleNames(r, []M{u}); e != nil {
			return nil, e
		}
		delete(u, "salt")
		u["password"] = ""
		u["avatar"] = a.userAvatar(u)
		u["location"] = a.location(u["last_login_ip"])
		u["reg_location"] = a.location(u["register_ip"])
		return u, nil
	case "checkinhistory":
		if uid < 1 {
			return nil, r.fail("用户ID无效")
		}
		if _, e := r.one("SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid); e != nil {
			return nil, e
		}
		r.c.Header("Cache-Control", "no-store")
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_check_in")+" WHERE user_id=?", uid)
		if e != nil {
			return nil, e
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		return r.list("SELECT DATE_FORMAT(sign_date,'%Y-%m-%d') sign_date,created_at signed_at FROM "+a.t("imgo_check_in")+" WHERE user_id=? ORDER BY sign_date DESC LIMIT ? OFFSET ?", uid, limit, offset)
	case "add":
		limits, e := userLimits(r.p)
		if e != nil {
			return nil, e
		}
		tx, e := a.db.BeginTx(r.ctx(), nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		id, e := a.createUser(r.ctx(), tx, r.p, a.clientIP(r.c))
		if e != nil {
			return nil, e
		}
		if !scope.Global {
			if e = a.bindInviter(r.ctx(), tx, id, scope.AgentUserID); e != nil {
				return nil, e
			}
		}
		if len(limits) > 0 {
			if e = update(r.ctx(), tx, a.t("user"), limits, "user_id=?", id); e != nil {
				return nil, e
			}
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
		return M{"user_id": id}, nil
	case "setremark":
		remark := r.s("remark")
		if len([]rune(remark)) > 191 {
			return nil, r.fail("备注最多191个字")
		}
		if _, err := r.one("SELECT user_id FROM "+a.t("user")+" WHERE "+whereUser, uid); err != nil {
			return nil, err
		}
		return nil, update(r.ctx(), a.db, a.t("user"), M{"remark": remark}, whereUser, uid)
	case "setinvitecode":
		return a.setMemberInviteCode(r, uid, whereUser, scope)
	case "edit":
		if uid == 1 && r.uid() != 1 {
			return nil, deny()
		}
		limits, err := userLimits(r.p)
		if err != nil {
			return nil, err
		}
		p := pick(r.p, "account", "realname", "email", "remark", "sex", "friend_limit", "group_limit", "cs_uid", "status")
		for k, v := range limits {
			p[k] = v
		}
		if name, ok := p["realname"]; ok {
			p["name_py"] = namePinyin(str(name))
		}
		if uid == 1 {
			p["role"] = 1
			p["status"] = 1
		}
		if s, ok := p["account"]; ok && (len(str(s)) < 3 || len(str(s)) > 32) {
			return nil, r.fail("账号长度无效")
		}
		if v, ok := p["cs_uid"]; ok && number(v) == uid {
			return nil, r.fail("不能将自己设为专属客服")
		}
		e := update(r.ctx(), a.db, a.t("user"), p, whereUser, uid)
		if e == nil {
			e = a.revoke(r.ctx(), uid)
		}
		return nil, e
	case "setrole", "setstatus":
		if uid == 1 {
			return nil, r.fail("不能停用或修改初始管理员角色")
		}
		if action(r) == "setrole" {
			roleID := r.n("admin_role_id")
			if roleID < 0 {
				return nil, r.fail("角色无效")
			}
			legacyRole := int64(0)
			if roleID > 0 {
				if _, e := r.one("SELECT role_id FROM "+a.t("imgo_admin_role")+" WHERE role_id=? AND status=1", roleID); e != nil {
					return nil, r.fail("角色不存在或已禁用")
				}
				legacyRole = 2
			}
			e := update(r.ctx(), a.db, a.t("user"), M{"admin_role_id": roleID, "role": legacyRole}, whereUser, uid)
			if e == nil {
				e = a.revoke(r.ctx(), uid)
			}
			return nil, e
		}
		value := r.n("status")
		if value < 0 || value > 1 {
			return nil, r.fail("参数无效")
		}
		e := update(r.ctx(), a.db, a.t("user"), M{"status": value}, whereUser, uid)
		if e == nil {
			e = a.revoke(r.ctx(), uid)
		}
		return nil, e
	case "editpassword":
		hash, e := hashPassword(r.s("password"))
		if e != nil {
			return nil, r.fail(e.Error())
		}
		if e = update(r.ctx(), a.db, a.t("user"), M{"password": hash, "salt": ""}, whereUser, uid); e != nil {
			return nil, e
		}
		return nil, a.revoke(r.ctx(), uid)
	case "del":
		if uid <= 1 {
			return nil, deny()
		}
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("group")+" WHERE owner_id=? AND status=1", uid)
		if e != nil {
			return nil, e
		}
		if number(n["n"]) > 0 {
			return nil, r.fail("请先转让或解散该用户的群聊")
		}
		tx, e := a.db.BeginTx(r.ctx(), nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		if !scope.Global {
			if e = a.requireScopedUser(r.ctx(), tx, scope, uid); e != nil {
				return nil, e
			}
		}
		if _, e = one(r.ctx(), tx, "SELECT user_id FROM "+a.t("user")+" WHERE "+whereUser+" FOR UPDATE", uid); e != nil {
			return nil, deny()
		}
		if e = update(r.ctx(), tx, a.t("user"), M{"status": 0, "delete_time": time.Now().Unix()}, "user_id=?", uid); e != nil {
			return nil, e
		}
		if _, e = tx.ExecContext(r.ctx(), "DELETE FROM "+a.t("friend")+" WHERE create_user=? OR friend_user_id=?", uid, uid); e != nil {
			return nil, e
		}
		if _, e = tx.ExecContext(r.ctx(), "DELETE FROM "+a.t("group_user")+" WHERE user_id=?", uid); e != nil {
			return nil, e
		}
		if e = tx.Commit(); e != nil {
			return nil, e
		}
		return nil, a.revoke(r.ctx(), uid)
	}
	return nil, r.fail("未知操作")
}
func (a *App) manageGroup(r *request) (any, error) {
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	gid := groupID(r)
	if gid == 0 {
		gid = r.n("group_id")
	}
	if !scope.Global && action(r) != "index" {
		if err := a.requireScopedGroup(r.ctx(), a.db, scope, gid); err != nil {
			return nil, err
		}
		targets := []int64{}
		switch action(r) {
		case "changeowner", "delgroupuser", "setmanager":
			targets = []int64{r.n("user_id")}
		case "addgroupuser":
			targets = ids(r.p["user_ids"])
		}
		for _, uid := range targets {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
		}
	}
	switch action(r) {
	case "index":
		where := "g.status=1 AND COALESCE(g.delete_time,0)=0"
		args := []any{}
		if !scope.Global {
			predicate, params := a.groupScopePredicate(scope, "g")
			where += " AND " + predicate
			args = append(args, params...)
		}
		if r.s("keywords") != "" {
			where += " AND g.name LIKE ?"
			args = append(args, "%"+r.s("keywords")+"%")
		}
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("group")+" g WHERE "+where, args...)
		if e != nil {
			return nil, e
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		list, e := r.list("SELECT g.*,(SELECT COUNT(*) FROM "+a.t("group_user")+" gu WHERE gu.group_id=g.group_id AND gu.status=1) AS user_count FROM "+a.t("group")+" g WHERE "+where+" ORDER BY g.group_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
		if e != nil {
			return nil, e
		}
		for _, g := range list {
			g["id"] = "group-" + str(g["group_id"])
			g["avatar"] = a.groupAvatarURL(number(g["group_id"]))
			u, e := r.one("SELECT user_id,realname,avatar FROM "+a.t("user")+" WHERE user_id=?", g["owner_id"])
			if e == nil {
				g["owner_id_info"] = a.safeUser(u)
			}
		}
		return list, nil
	case "changeowner":
		return nil, a.changeOwner(r, gid, r.n("user_id"), scope)
	case "del":
		return nil, a.deleteGroup(r, gid, scope)
	case "addgroupuser":
		g, e := r.one("SELECT * FROM "+a.t("group")+" WHERE group_id=? AND status=1", gid)
		if e != nil {
			return nil, e
		}
		return nil, a.addMembers(r, g, ids(r.p["user_ids"]), scope)
	case "delgroupuser":
		g, e := r.one("SELECT owner_id FROM "+a.t("group")+" WHERE group_id=?", gid)
		if e != nil {
			return nil, e
		}
		uid := r.n("user_id")
		if uid == number(g["owner_id"]) {
			return nil, r.fail("请先转让群主")
		}
		where, args := a.groupMemberScopeWhere(scope, gid, uid)
		e = r.exec("DELETE FROM "+a.t("group_user")+" WHERE "+where, args...)
		if e == nil {
			a.hub.send([]int64{uid}, "removeUser", M{"group_id": "group-" + fmt.Sprint(gid), "user_id": uid})
		}
		return nil, e
	case "setmanager":
		role := r.n("role")
		if role != 2 && role != 3 {
			return nil, r.fail("角色无效")
		}
		where, args := a.groupMemberScopeWhere(scope, gid, r.n("user_id"))
		return nil, update(r.ctx(), a.db, a.t("group_user"), M{"role": role}, where+" AND role<>1", args...)
	}
	return nil, r.fail("未知操作")
}
func (a *App) manageConfig(r *request) (any, error) {
	name := r.s("name")
	switch action(r) {
	case "getinfo", "getconfig":
		return a.config(r.ctx(), name), nil
	case "getallconfig":
		return r.list("SELECT * FROM " + a.t("config") + " WHERE status=1 ORDER BY id")
	case "setconfig":
		valid := map[string]bool{"sysInfo": true, "chatInfo": true, "fileUpload": true, "compass": true, "email": true, "smtp": true, "appVersion": true, "sms": true}
		if !valid[name] {
			return nil, r.fail("配置名称无效")
		}
		v := obj(r.p["value"])
		if len(v) == 0 {
			return nil, r.fail("配置必须是 JSON 对象")
		}
		if name == "chatInfo" {
			if limit, ok := obj(v["autoAddGroup"])["userMax"]; ok {
				if err := validateAutoGroupUserMax(limit); err != nil {
					return nil, err
				}
			}
		}
		if name == "fileUpload" && str(v["disk"]) != "local" {
			if _, err := newObjectStore(str(v["disk"]), obj(v[str(v["disk"])])); err != nil {
				return nil, err
			}
		}
		existing, e := r.one("SELECT id FROM "+a.t("config")+" WHERE name=?", name)
		if e == sql.ErrNoRows {
			_, e = insert(r.ctx(), a.db, a.t("config"), M{"name": name, "value": js(v), "create_user": r.uid(), "create_time": time.Now().Unix(), "status": 1})
		} else if e == nil {
			e = update(r.ctx(), a.db, a.t("config"), M{"value": js(v), "update_time": time.Now().Unix()}, "id=?", existing["id"])
		}
		if e == nil && (name == "sysInfo" || name == "chatInfo") {
			safe := M{}
			for k, val := range v {
				safe[k] = val
			}
			delete(safe, "stunPass")
			a.hub.broadcast("updateConfig", M{"name": name, "value": safe})
		}
		return nil, e
	case "getinvitelink":
		token := randomID()
		a.put("invite:"+token, r.uid(), 48*time.Hour)
		return a.inviteURL(token), nil
	case "sendtestemail":
		return nil, a.sendMail(r.s("email"), "Imgo 邮件测试", "邮件服务连接成功。")
	}
	return nil, r.fail("未知操作")
}

func validateAutoGroupUserMax(value any) error {
	limit, err := strconv.ParseInt(str(value), 10, 64)
	if err != nil || limit < 5 || limit > 10000 {
		return clientError{"自动加入群聊的成员上限必须是 5 到 10000 的整数", 400}
	}
	return nil
}
func (a *App) manageIndex(r *request) (any, error) {
	if action(r) != "noticelist" {
		if err := a.requireGlobalAdminScope(r); err != nil {
			return nil, err
		}
	}
	switch action(r) {
	case "noticelist":
		n, e := r.one("SELECT COUNT(*) n FROM " + a.t("message") + " WHERE chat_identify='admin_notice' AND status=1")
		if e != nil {
			return nil, e
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		list, e := r.list("SELECT * FROM "+a.t("message")+" WHERE chat_identify='admin_notice' AND status=1 ORDER BY msg_id DESC LIMIT ? OFFSET ?", limit, offset)
		if e != nil {
			return nil, e
		}
		for _, m := range list {
			m["title"] = obj(m["extends"])["title"]
			m["content"], e = decryptContent(a.cfg.ChatKey, str(m["content"]))
			if e != nil {
				return nil, e
			}
		}
		return list, nil
	case "delnotice":
		return nil, r.exec("UPDATE "+a.t("message")+" SET status=0 WHERE msg_id=? AND chat_identify='admin_notice'", r.n("id"))
	case "publishnotice":
		if r.s("title") == "" || len([]rune(r.s("title"))) > 200 || len(r.s("content")) > 100000 {
			return nil, r.fail("标题或内容长度无效")
		}
		content, e := encryptContent(a.cfg.ChatKey, sanitizeText(r.s("content")))
		if e != nil {
			return nil, e
		}
		p := M{"content": content, "extends": js(M{"title": r.s("title"), "notice": sanitizeText(r.s("content"))})}
		id := r.n("msgId")
		if id > 0 {
			e = update(r.ctx(), a.db, a.t("message"), p, "msg_id=? AND chat_identify='admin_notice'", id)
		} else {
			p["id"] = randomID()[:32]
			p["from_user"] = r.uid()
			p["to_user"] = 0
			p["chat_identify"] = "admin_notice"
			p["type"] = "text"
			p["is_group"] = 2
			p["is_read"] = 1
			p["is_last"] = 1
			p["status"] = 1
			p["create_time"] = time.Now().Unix()
			id, e = insert(r.ctx(), a.db, a.t("message"), p)
		}
		if e == nil {
			a.hub.broadcast("simple", M{"toContactId": "admin_notice", "content": r.s("title"), "msg_id": id, "type": "text", "is_group": 2, "sendTime": time.Now().UnixMilli(), "fromUser": a.safeUser(r.user), "extends": obj(p["extends"])})
		}
		return M{"msg_id": id}, e
	case "clearmessage":
		return nil, r.exec("UPDATE " + a.t("message") + " SET status=0 WHERE chat_identify<>'admin_notice'")
	}
	return nil, r.fail("未知操作")
}
func (a *App) manageMessage(r *request) (any, error) {
	if action(r) == "index" {
		return a.messageList(r, true)
	}
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	switch action(r) {
	case "getcontacts":
		uid := r.n("user_id")
		if !scope.Global {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
		}
		u, e := r.one("SELECT * FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", uid)
		if e != nil {
			return nil, e
		}
		copy := *r
		copy.user = u
		out, e := a.contacts(&copy, uid, scope)
		r.count = copy.count
		return out, e
	case "dealmsg":
		where, args := "id=?", []any{r.s("id")}
		if !scope.Global {
			predicate, params := a.messageScopePredicate(scope, a.t("message"))
			where += " AND " + predicate
			args = append(args, params...)
		}
		m, e := r.one("SELECT * FROM "+a.t("message")+" WHERE "+where, args...)
		if e == sql.ErrNoRows && !scope.Global {
			return nil, deny()
		}
		if e != nil {
			return nil, e
		}
		event := "updateMessage"
		content := "此消息已被管理员屏蔽"
		if r.n("dealType") == 1 {
			event = "delMessage"
			e = update(r.ctx(), a.db, a.t("message"), M{"status": 0}, where, args...)
		} else {
			encrypted, _ := encryptContent(a.cfg.ChatKey, content)
			e = update(r.ctx(), a.db, a.t("message"), M{"content": encrypted, "type": "text"}, where, args...)
		}
		if e == nil {
			a.messageEvent(r.ctx(), m, event, M{"id": m["id"], "content": content})
		}
		return nil, e
	}
	return nil, r.fail("未知操作")
}
func (a *App) task(r *request) (any, error) {
	switch action(r) {
	case "settaskconfig", "starttask", "stoptask", "cleartasklog":
		if err := a.requireGlobalAdminScope(r); err != nil {
			return nil, err
		}
	}
	switch action(r) {
	case "gettasklist":
		c, e := a.loadMaintenance(r.ctx())
		if e != nil {
			return nil, e
		}
		status := "stop"
		if number(c["enabled"]) == 1 {
			status = "active"
		}
		return []M{{"name": "schedule", "remark": "Go 定时清理任务", "status": status, "pid": 0, "settings": maintenanceView(c)}, {"name": "worker", "remark": "Go WebSocket 服务（随主服务运行）", "status": "active", "pid": 0}}, nil
	case "settaskconfig":
		return a.configureMaintenance(r, nil)
	case "starttask":
		enabled := true
		return a.configureMaintenance(r, &enabled)
	case "stoptask":
		enabled := false
		return a.configureMaintenance(r, &enabled)
	case "gettasklog", "cleartasklog":
		a.maintenanceMu.Lock()
		defer a.maintenanceMu.Unlock()
		c, e := a.loadMaintenance(r.ctx())
		if e != nil {
			return nil, e
		}
		if action(r) == "cleartasklog" {
			c["logs"] = []any{}
			return nil, a.saveMaintenance(r.ctx(), a.db, c)
		}
		lines := []string{}
		if logs, ok := c["logs"].([]any); ok {
			for _, line := range logs {
				lines = append(lines, str(line))
			}
		}
		return strings.Join(lines, "\n"), nil
	}
	return nil, r.fail("未知操作")
}

var _ = strings.TrimSpace

// Keep administrative quotas out of the public registration allowlist.
func userLimits(p M) (M, error) {
	out := M{}
	for _, key := range []string{"friend_limit", "group_limit"} {
		if value, ok := p[key]; ok {
			n := number(value)
			if n < -1 || n > 1000 || str(value) != fmt.Sprint(n) {
				return nil, clientError{"好友和群聊上限必须是 -1 到 1000 的整数", 400}
			}
			out[key] = n
		}
	}
	return out, nil
}
