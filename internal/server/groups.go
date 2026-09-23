package server

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func groupID(r *request) int64 {
	s := r.s("group_id")
	if s == "" {
		s = r.s("id")
	}
	return number(strings.TrimPrefix(s, "group-"))
}
func (a *App) group(r *request) (any, error) {
	act := action(r)
	gid := groupID(r)
	if act == "add" {
		return a.createGroup(r)
	}
	if act == "getalluser" {
		scopeWhere, args := "", []any{r.uid(), gid}
		if number(r.user["admin_role_id"]) > 0 {
			scope, err := a.adminScope(r.ctx(), r.user)
			if err != nil {
				return nil, err
			}
			if !scope.Global {
				if err := a.requireScopedGroup(r.ctx(), a.db, scope, gid); err != nil {
					return nil, err
				}
				predicate, params := scope.userPredicate("u")
				scopeWhere = " AND " + predicate
				args = append(args, params...)
			}
		}
		list, e := r.list("SELECT user_id,realname,avatar,name_py FROM "+a.t("user")+" u WHERE status=1 AND delete_time=0 AND user_id<>? AND user_id NOT IN (SELECT user_id FROM "+a.t("group_user")+" WHERE group_id=? AND status=1)"+scopeWhere+" ORDER BY user_id LIMIT 2000", args...)
		if e != nil {
			return nil, e
		}
		exclude := map[int64]bool{}
		for _, n := range ids(r.p["user_ids"]) {
			exclude[n] = true
		}
		out := []M{}
		for _, u := range list {
			if !exclude[number(u["user_id"])] {
				out = append(out, a.safeUser(u))
			}
		}
		return out, nil
	}
	g, e := r.one("SELECT * FROM "+a.t("group")+" WHERE group_id=? AND status=1 AND COALESCE(delete_time,0)=0", gid)
	if e != nil {
		if e == sql.ErrNoRows && number(r.user["admin_role_id"]) > 0 && (act == "groupinfo" || act == "groupuserlist" || act == "editgroupavatar") {
			scope, err := a.adminScope(r.ctx(), r.user)
			if err != nil {
				return nil, err
			}
			if !scope.Global {
				return nil, deny()
			}
		}
		return nil, e
	}
	if act == "joingroup" {
		if number(g["is_public"]) != 1 {
			token := r.s("token")
			parts := strings.Split(token, ".")
			if len(parts) != 3 || number(parts[0]) != gid || number(parts[1]) < time.Now().Unix() || parts[2] != a.signLink(parts[0]+"."+parts[1]) {
				return nil, r.fail("私有群需要有效邀请")
			}
		}
		return nil, a.addMembers(r, g, []int64{r.uid()})
	}
	member, e := a.member(r.ctx(), a.db, gid, r.uid())
	role := number(member["role"])
	resourceScope := adminScope{Global: true}
	// Membership is sufficient for normal chat access. Resolve administrative
	// permissions only when this action needs to bypass the member's authority.
	needsAdmin := (e != nil && (act == "groupinfo" || act == "groupuserlist")) || (act == "editgroupavatar" && role != 1 && role != 2)
	systemAdmin := r.uid() == 1 || number(r.user["role"]) > 0
	if needsAdmin && number(r.user["admin_role_id"]) > 0 {
		scope, err := a.adminScope(r.ctx(), r.user)
		if err != nil {
			return nil, err
		}
		if !scope.Global {
			if err := a.requireScopedUser(r.ctx(), a.db, scope, number(g["owner_id"])); err != nil {
				return nil, err
			}
		}
		if err := a.authorizeManage(r.ctx(), r.user, "manage.groups"); err != nil {
			return nil, err
		}
		resourceScope = scope
		systemAdmin = true
	}
	adminAvatar := systemAdmin && act == "editgroupavatar"
	adminRead := systemAdmin && (act == "groupinfo" || act == "groupuserlist")
	inviteRead := act == "groupinfo" && a.validGroupToken(r.s("token")) && number(strings.Split(r.s("token"), ".")[0]) == gid
	if e != nil && !adminRead && !inviteRead && !adminAvatar {
		return nil, e
	}
	switch act {
	case "groupinfo":
		v := M{}
		for k, val := range g {
			v[k] = val
		}
		v["id"] = "group-" + fmt.Sprint(gid)
		v["displayName"] = g["name"]
		v["avatar"] = a.groupAvatarURL(gid)
		v["setting"] = obj(g["setting"])
		v["isJoin"] = role
		v["canEditAvatar"] = systemAdmin || role == 1 || role == 2
		owner, e := r.one("SELECT user_id,realname,avatar FROM "+a.t("user")+" WHERE user_id=?", g["owner_id"])
		if e != nil {
			return nil, e
		}
		v["userInfo"] = a.safeUser(owner)
		v["ownerName"] = owner["realname"]
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("group_user")+" WHERE group_id=? AND status=1", gid)
		if e != nil {
			return nil, e
		}
		v["groupUserCount"] = n["n"]
		exp := time.Now().Add(7 * 24 * time.Hour)
		value := fmt.Sprintf("%d.%d", gid, exp.Unix())
		v["qrUrl"] = a.scanURL("g", value+"."+a.signLink(value))
		v["qrExpire"] = exp.Format("01月02日")
		if role == 0 && !adminRead {
			// An invitation preview must not mint a fresh invitation or expose settings.
			v = M{"id": v["id"], "group_id": gid, "name": g["name"], "displayName": g["name"], "isJoin": 0, "groupUserCount": n["n"], "setting": M{"invite": obj(g["setting"])["invite"]}, "avatar": a.mediaPath("/avatar/群/120/" + fmt.Sprint(gid))}
		}
		return v, nil
	case "groupuserlist":
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("group_user")+" WHERE group_id=? AND status=1", gid)
		if e != nil {
			return nil, e
		}
		r.count = number(n["n"])
		limit, offset := r.pagination()
		if r.n("limit") == 0 {
			limit = 2000
		}
		list, e := r.list("SELECT * FROM "+a.t("group_user")+" WHERE group_id=? AND status=1 ORDER BY role,user_id LIMIT ? OFFSET ?", gid, limit, offset)
		if e != nil {
			return nil, e
		}
		for _, m := range list {
			u, e := r.one("SELECT user_id,realname,avatar,name_py FROM "+a.t("user")+" WHERE user_id=?", m["user_id"])
			if e == nil {
				m["userInfo"] = a.safeUser(u)
			}
		}
		return list, nil
	case "addgroupuser":
		if role > 2 && number(obj(g["setting"])["invite"]) == 0 {
			return nil, deny()
		}
		return nil, a.addMembers(r, g, ids(r.p["user_ids"]))
	case "editgroupname":
		if role > 2 {
			return nil, deny()
		}
		name := r.s("displayName")
		if name == "" || len([]rune(name)) > 64 {
			return nil, r.fail("群名称长度无效")
		}
		e = update(r.ctx(), a.db, a.t("group"), M{"name": name, "name_py": namePinyin(name)}, "group_id=?", gid)
	case "editgroupavatar":
		if !systemAdmin && role != 1 && role != 2 {
			return nil, deny()
		}
		fid := r.n("file_id")
		if fid <= 0 {
			return nil, r.fail("请选择群头像")
		}
		f, err := r.one("SELECT * FROM "+a.t("file")+" WHERE file_id=? AND user_id=? AND status=1 AND COALESCE(delete_time,0)=0", fid, r.uid())
		if err != nil || number(f["cate"]) != 2 || number(f["size"]) > 5<<20 {
			return nil, r.fail("群头像必须是本人上传的图片，且不超过 5 MB")
		}
		src := str(f["src"])
		if !strings.HasPrefix(src, "/storage/image/") || !safeAsset(strings.TrimPrefix(src, "/")) {
			return nil, r.fail("群头像文件无效")
		}
		where, args := "group_id=?", []any{gid}
		if !resourceScope.Global {
			predicate, params := a.groupScopePredicate(resourceScope, a.t("group"))
			where += " AND " + predicate
			args = append(args, params...)
		}
		changed, e := a.scopedResourceWrite(r, resourceScope, func(db DB) (sql.Result, error) {
			return updateResult(r.ctx(), db, a.t("group"), M{"avatar": src}, where, args...)
		}, func(db DB) error { return a.requireScopedGroup(r.ctx(), db, resourceScope, gid) })
		if e != nil {
			return nil, e
		}
		avatar := a.groupAvatarURL(gid) + "?v=" + fmt.Sprint(time.Now().UnixNano())
		if changed {
			a.groupEvent(r.ctx(), gid, "editGroupAvatar", M{"group_id": "group-" + fmt.Sprint(gid), "avatar": avatar})
		}
		return M{"avatar": avatar}, nil
	case "setmanager":
		if role != 1 {
			return nil, deny()
		}
		target := r.n("user_id")
		newRole := r.n("role")
		if target == number(g["owner_id"]) || (newRole != 2 && newRole != 3) {
			return nil, deny()
		}
		e = update(r.ctx(), a.db, a.t("group_user"), M{"role": newRole}, "group_id=? AND user_id=? AND status=1", gid, target)
	case "removeuser":
		uid := r.n("user_id")
		target, e2 := a.member(r.ctx(), a.db, gid, uid)
		if e2 != nil {
			return nil, e2
		}
		if uid == number(g["owner_id"]) {
			return nil, r.fail("群主请先转让或解散群聊")
		}
		if uid != r.uid() && (role > 2 || role >= number(target["role"])) {
			return nil, deny()
		}
		e = r.exec("DELETE FROM "+a.t("group_user")+" WHERE group_id=? AND user_id=?", gid, uid)
		if e == nil {
			a.hub.send([]int64{uid}, "removeUser", M{"group_id": "group-" + fmt.Sprint(gid), "user_id": uid})
		}
	case "setnospeak":
		uid := r.n("user_id")
		target, err := a.member(r.ctx(), a.db, gid, uid)
		if err != nil {
			return nil, err
		}
		if role > 2 || role >= number(target["role"]) {
			return nil, deny()
		}
		duration := r.n("noSpeakDay") * 86400
		if duration < 0 || duration > 31536000 {
			return nil, r.fail("禁言时长无效")
		}
		timer := r.n("noSpeakTimer")
		if timer > 0 && timer <= 4 {
			duration = []int64{600, 3600, 10800, 86400}[timer-1]
		}
		until := int64(0)
		if duration > 0 {
			until = time.Now().Unix() + duration
		}
		e = update(r.ctx(), a.db, a.t("group_user"), M{"no_speak_time": until}, "group_id=? AND user_id=?", gid, uid)
	case "removegroup":
		if role != 1 {
			return nil, deny()
		}
		return nil, a.deleteGroup(r, gid)
	case "setnotice":
		if e := a.moderate(r.ctx(), r.s("notice"), "comment_detection"); e != nil {
			return nil, e
		}
		if role > 2 {
			return nil, deny()
		}
		if len([]rune(r.s("notice"))) > 4000 {
			return nil, r.fail("公告过长")
		}
		e = update(r.ctx(), a.db, a.t("group"), M{"notice": sanitizeText(r.s("notice"))}, "group_id=?", gid)
	case "groupsetting":
		if role != 1 {
			return nil, deny()
		}
		settings := pick(obj(r.p["setting"]), "manage", "invite", "nospeak", "history")
		e = update(r.ctx(), a.db, a.t("group"), M{"setting": js(settings)}, "group_id=?", gid)
	case "changeowner":
		if role != 1 {
			return nil, deny()
		}
		return nil, a.changeOwner(r, gid, r.n("user_id"))
	case "clearmessage":
		if role != 1 && r.uid() != 1 {
			return nil, deny()
		}
		return nil, a.clearGroupMessages(r, gid)
	default:
		return nil, r.fail("未知操作")
	}
	if e != nil {
		return nil, e
	}
	data := pick(r.p, "id", "user_id", "displayName", "role", "setting", "notice", "noSpeakTimer", "noSpeakDay")
	data["group_id"] = "group-" + fmt.Sprint(gid)
	a.groupEvent(r.ctx(), gid, groupEventName(act), data)
	return nil, nil
}
func groupEventName(act string) string {
	for _, s := range []string{"editGroupName", "editGroupAvatar", "setManager", "removeUser", "setNoSpeak", "setNotice", "groupSetting", "changeOwner"} {
		if strings.ToLower(s) == act {
			return s
		}
	}
	return act
}
func (a *App) createGroup(r *request) (any, error) {
	if number(a.config(r.ctx(), "chatInfo")["groupChat"]) == 0 {
		return nil, deny()
	}
	users := ids(r.p["user_ids"])
	if len(users) > 100 {
		return nil, r.fail("单次创建最多邀请 100 人")
	}
	users = append(users, r.uid())
	unique := map[int64]bool{}
	for _, u := range users {
		unique[u] = true
	}
	users = users[:0]
	for u := range unique {
		users = append(users, u)
	}
	max := number(a.config(r.ctx(), "chatInfo")["groupUserMax"])
	if max > 0 && int64(len(users)) > max {
		return nil, r.fail("超过群人数上限")
	}
	limit := number(r.user["group_limit"])
	if limit != 0 && number(r.user["role"]) == 0 {
		n, e := r.one("SELECT COUNT(*) n FROM "+a.t("group")+" WHERE owner_id=? AND status=1", r.uid())
		if e != nil {
			return nil, e
		}
		if limit < 0 || number(n["n"]) >= limit {
			return nil, r.fail("超过建群上限")
		}
	}
	name := r.s("name")
	if name == "" {
		name = "群聊"
	}
	if len([]rune(name)) > 64 {
		return nil, r.fail("群名过长")
	}
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	valid, e := rows(r.ctx(), tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id IN ("+marks(len(users))+") AND status=1 AND delete_time=0", values(users)...)
	if e != nil {
		return nil, e
	}
	if len(valid) != len(users) {
		return nil, r.fail("存在无效成员")
	}
	gid, e := insert(r.ctx(), tx, a.t("group"), M{"name": name, "name_py": namePinyin(name), "owner_id": r.uid(), "create_user": r.uid(), "create_time": time.Now().Unix(), "setting": `{"manage":0,"invite":1,"nospeak":0,"history":1}`, "status": 1})
	if e != nil {
		return nil, e
	}
	for _, u := range users {
		role := 3
		if u == r.uid() {
			role = 1
		}
		if _, e = insert(r.ctx(), tx, a.t("group_user"), M{"group_id": gid, "user_id": u, "role": role, "invite_id": r.uid(), "create_time": time.Now().Unix(), "status": 1}); e != nil {
			return nil, e
		}
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	contact, e := a.contact(r, "group-"+fmt.Sprint(gid))
	if e != nil {
		return nil, e
	}
	event := M{}
	for k, v := range contact {
		event[k] = v
	}
	event["role"] = 3
	a.hub.send(users, "addGroup", event)
	return contact, nil
}
func (a *App) addMembers(r *request, g M, users []int64, scopes ...adminScope) error {
	if len(users) < 1 || len(users) > 100 {
		return r.fail("邀请人数须为 1–100")
	}
	gid := number(g["group_id"])
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	locked, e := one(r.ctx(), tx, "SELECT group_id,owner_id FROM "+a.t("group")+" WHERE group_id=? AND status=1 FOR UPDATE", gid)
	if e != nil {
		return e
	}
	if len(scopes) > 0 && !scopes[0].Global {
		if err := a.requireScopedUser(r.ctx(), tx, scopes[0], number(locked["owner_id"])); err != nil {
			return err
		}
		for _, uid := range users {
			if err := a.requireScopedUser(r.ctx(), tx, scopes[0], uid); err != nil {
				return err
			}
		}
	}
	n, e := one(r.ctx(), tx, "SELECT COUNT(*) n FROM "+a.t("group_user")+" WHERE group_id=? AND status=1", gid)
	if e != nil {
		return e
	}
	max := number(a.config(r.ctx(), "chatInfo")["groupUserMax"])
	total := number(n["n"])
	added := []int64{}
	for _, uid := range users {
		if _, e = one(r.ctx(), tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", uid); e != nil {
			return e
		}
		if _, err := one(r.ctx(), tx, "SELECT id FROM "+a.t("group_user")+" WHERE group_id=? AND user_id=?", gid, uid); err == nil {
			continue
		} else if err != sql.ErrNoRows {
			return err
		}
		total++
		if max > 0 && total > max {
			return r.fail("超过群人数上限")
		}
		if _, e = insert(r.ctx(), tx, a.t("group_user"), M{"group_id": gid, "user_id": uid, "role": 3, "invite_id": r.uid(), "create_time": time.Now().Unix(), "status": 1}); e != nil {
			return e
		}
		added = append(added, uid)
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	data := M{"group_id": "group-" + fmt.Sprint(gid), "avatar": a.groupAvatarURL(gid)}
	a.groupEvent(r.ctx(), gid, "addGroupUser", data)
	a.hub.send(added, "addGroup", M{"id": data["group_id"], "displayName": g["name"], "is_group": 1, "role": 3, "avatar": data["avatar"], "setting": obj(g["setting"])})
	return nil
}
func (a *App) changeOwner(r *request, gid, uid int64, scopes ...adminScope) error {
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	g, e := one(r.ctx(), tx, "SELECT * FROM "+a.t("group")+" WHERE group_id=? AND status=1 FOR UPDATE", gid)
	if e != nil {
		return e
	}
	if len(scopes) > 0 && !scopes[0].Global {
		if err := a.requireScopedUser(r.ctx(), tx, scopes[0], number(g["owner_id"])); err != nil {
			return err
		}
		if err := a.requireScopedUser(r.ctx(), tx, scopes[0], uid); err != nil {
			return err
		}
	}
	if _, e = a.member(r.ctx(), tx, gid, uid); e != nil {
		return e
	}
	if uid == number(g["owner_id"]) {
		return nil
	}
	if e = update(r.ctx(), tx, a.t("group_user"), M{"role": 3}, "group_id=? AND user_id=?", gid, g["owner_id"]); e != nil {
		return e
	}
	if e = update(r.ctx(), tx, a.t("group_user"), M{"role": 1}, "group_id=? AND user_id=?", gid, uid); e != nil {
		return e
	}
	if e = update(r.ctx(), tx, a.t("group"), M{"owner_id": uid}, "group_id=?", gid); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	a.groupEvent(r.ctx(), gid, "changeOwner", M{"group_id": "group-" + fmt.Sprint(gid), "user_id": uid})
	return nil
}
func (a *App) deleteGroup(r *request, gid int64, scopes ...adminScope) error {
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if len(scopes) > 0 && !scopes[0].Global {
		locked, err := one(r.ctx(), tx, "SELECT owner_id FROM "+a.t("group")+" WHERE group_id=? AND status=1 FOR UPDATE", gid)
		if err != nil {
			return err
		}
		if err := a.requireScopedUser(r.ctx(), tx, scopes[0], number(locked["owner_id"])); err != nil {
			return err
		}
	}
	members, e := rows(r.ctx(), tx, "SELECT user_id FROM "+a.t("group_user")+" WHERE group_id=?", gid)
	if e != nil {
		return e
	}
	if e = update(r.ctx(), tx, a.t("group"), M{"status": 0, "delete_time": time.Now().Unix()}, "group_id=?", gid); e != nil {
		return e
	}
	if _, e = tx.ExecContext(r.ctx(), "DELETE FROM "+a.t("group_user")+" WHERE group_id=?", gid); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	uids := []int64{}
	for _, m := range members {
		uids = append(uids, number(m["user_id"]))
	}
	a.hub.send(uids, "removeGroup", M{"group_id": "group-" + fmt.Sprint(gid)})
	return nil
}
func (a *App) clearGroupMessages(r *request, gid int64) error {
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.ctx(), "UPDATE "+a.t("message")+" SET status=0 WHERE chat_identify=?", "group-"+fmt.Sprint(gid)); e != nil {
		return e
	}
	if _, e = tx.ExecContext(r.ctx(), "UPDATE "+a.t("group_user")+" SET unread=0 WHERE group_id=?", gid); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	a.groupEvent(r.ctx(), gid, "clearMessage", M{"group_id": "group-" + fmt.Sprint(gid)})
	return nil
}
