package server

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func nextCustomerService(ids []int64, last int64) int64 {
	if len(ids) == 0 {
		return 0
	}
	for i, id := range ids {
		if id == last && i+1 < len(ids) {
			return ids[i+1]
		}
	}
	return ids[0]
}

func (a *App) addMutualFriendship(ctx context.Context, db DB, customerID, userID, now int64) error {
	for _, pair := range [][2]int64{{customerID, userID}, {userID, customerID}} {
		if _, e := insert(ctx, db, a.t("friend"), M{
			"create_user":    pair[0],
			"friend_user_id": pair[1],
			"status":         1,
			"create_time":    now,
		}); e != nil {
			return e
		}
	}
	return nil
}

// Register plus assignment is one transaction; a failed rule cannot leave a half-created account.
func (a *App) createRegisteredUser(ctx context.Context, p M, ip string) (int64, error) {
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	inviterID, e := a.resolveInviter(ctx, tx, str(p["inviteCode"]))
	if e != nil {
		return 0, e
	}
	uid, e := a.createUser(ctx, tx, p, ip)
	if e != nil {
		return 0, e
	}
	if e = a.bindInviter(ctx, tx, uid, inviterID); e != nil {
		return 0, e
	}
	if number(a.config(ctx, "sysInfo")["runMode"]) != 2 {
		if e = tx.Commit(); e != nil {
			return 0, e
		}
		return uid, nil
	}
	// Serialize round-robin state even if autoTask did not previously exist.
	if _, e = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_chat_lock")+" (chat_identify) VALUES ('__registration__') ON DUPLICATE KEY UPDATE chat_identify=VALUES(chat_identify)"); e != nil {
		return 0, e
	}
	state, e := one(ctx, tx, "SELECT id,value FROM "+a.t("config")+" WHERE name='autoTask' LIMIT 1 FOR UPDATE")
	if e == sql.ErrNoRows {
		id, err := insert(ctx, tx, a.t("config"), M{"name": "autoTask", "value": `{"user_id":0,"group_id":0,"group_num":1}`, "status": 1})
		if err != nil {
			return 0, err
		}
		state = M{"id": id, "value": M{"group_num": 1}}
	} else if e != nil {
		return 0, e
	}
	task := obj(state["value"])
	chat := a.config(ctx, "chatInfo")
	autoUser := obj(chat["autoAddUser"])
	autoGroup := obj(chat["autoAddGroup"])
	if number(autoUser["status"]) == 1 {
		valid := []int64{}
		for _, id := range ids(autoUser["user_ids"]) {
			if _, err := one(ctx, tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", id); err == nil {
				valid = append(valid, id)
			} else if err != sql.ErrNoRows {
				return 0, err
			}
		}
		cs := nextCustomerService(valid, number(task["user_id"]))
		if cs > 0 {
			task["user_id"] = cs
			if e = update(ctx, tx, a.t("user"), M{"cs_uid": cs}, "user_id=?", uid); e != nil {
				return 0, e
			}
			now := time.Now().Unix()
			if e = a.addMutualFriendship(ctx, tx, cs, uid, now); e != nil {
				return 0, e
			}
			if welcome := str(autoUser["welcome"]); welcome != "" {
				content, _ := encryptContent(a.cfg.ChatKey, sanitizeText(welcome))
				if _, e = insert(ctx, tx, a.t("message"), M{"id": randomID()[:32], "from_user": cs, "to_user": uid, "chat_identify": chatKey(cs, uid, 0), "content": content, "type": "text", "create_time": now, "is_group": 0, "is_read": 0, "is_last": 1, "status": 1}); e != nil {
					return 0, e
				}
			}
		}
	}
	gid := int64(0)
	if number(autoGroup["status"]) == 1 && number(autoGroup["owner_uid"]) > 0 {
		owner := number(autoGroup["owner_uid"])
		if _, e = one(ctx, tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", owner); e != nil {
			return 0, clientError{"自动建群的群主不可用", 400}
		}
		gid = number(task["group_id"])
		max := number(autoGroup["userMax"])
		if max < 2 {
			max = 100
		}
		g, err := one(ctx, tx, "SELECT group_id,owner_id FROM "+a.t("group")+" WHERE group_id=? AND status=1 FOR UPDATE", gid)
		if err != nil && err != sql.ErrNoRows {
			return 0, err
		}
		full := false
		if err == nil {
			n, err := one(ctx, tx, "SELECT COUNT(*) n FROM "+a.t("group_user")+" WHERE group_id=? AND status=1", gid)
			if err != nil {
				return 0, err
			}
			full = number(n["n"]) >= max || number(g["owner_id"]) != owner
		}
		if err == sql.ErrNoRows || full {
			seq := number(task["group_num"])
			if seq < 1 {
				seq = 1
			}
			if gid > 0 {
				seq++
			}
			name := str(autoGroup["name"])
			if name == "" {
				name = "群聊"
			}
			name += fmt.Sprint(seq)
			gid, e = insert(ctx, tx, a.t("group"), M{"name": name, "name_py": namePinyin(name), "create_user": owner, "owner_id": owner, "create_time": time.Now().Unix(), "status": 1, "setting": `{"manage":0,"invite":1,"nospeak":0,"history":1}`})
			if e != nil {
				return 0, e
			}
			if _, e = insert(ctx, tx, a.t("group_user"), M{"group_id": gid, "user_id": owner, "role": 1, "invite_id": owner, "status": 1, "create_time": time.Now().Unix()}); e != nil {
				return 0, e
			}
			task["group_num"] = seq
		}
		if _, e = insert(ctx, tx, a.t("group_user"), M{"group_id": gid, "user_id": uid, "role": 3, "invite_id": owner, "status": 1, "create_time": time.Now().Unix()}); e != nil {
			return 0, e
		}
		task["group_id"] = gid
	}
	if e = update(ctx, tx, a.t("config"), M{"value": js(task)}, "id=?", state["id"]); e != nil {
		return 0, e
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	if gid > 0 {
		a.groupEvent(ctx, gid, "addGroupUser", M{"group_id": fmt.Sprintf("group-%d", gid), "user_id": uid})
	}
	return uid, nil
}
