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

func (a *App) loadGlobalAutoTask(ctx context.Context, tx *sql.Tx) (M, M, error) {
	state, err := one(ctx, tx, "SELECT id,value FROM "+a.t("config")+" WHERE name='autoTask' LIMIT 1 FOR UPDATE")
	if err == sql.ErrNoRows {
		id, e := insert(ctx, tx, a.t("config"), M{"name": "autoTask", "value": `{"user_id":0,"group_id":0,"group_num":1}`, "status": 1})
		if e != nil {
			return nil, nil, e
		}
		state = M{"id": id, "value": M{"group_num": 1}}
	} else if err != nil {
		return nil, nil, err
	}
	return state, obj(state["value"]), nil
}

func (a *App) loadAgentAutoState(ctx context.Context, tx *sql.Tx, agentID int64) (M, error) {
	if _, err := tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_agent_auto_state")+" (agent_user_id,updated_at) VALUES (?,?) ON DUPLICATE KEY UPDATE agent_user_id=VALUES(agent_user_id)", agentID, time.Now().Unix()); err != nil {
		return nil, err
	}
	row, err := one(ctx, tx, "SELECT last_customer_user_id,group_id,group_num FROM "+a.t("imgo_agent_auto_state")+" WHERE agent_user_id=? FOR UPDATE", agentID)
	if err != nil {
		return nil, err
	}
	return M{"user_id": row["last_customer_user_id"], "group_id": row["group_id"], "group_num": row["group_num"]}, nil
}

func (a *App) saveAgentAutoState(ctx context.Context, tx *sql.Tx, agentID int64, state M) error {
	_, err := tx.ExecContext(ctx, "UPDATE "+a.t("imgo_agent_auto_state")+" SET last_customer_user_id=?,group_id=?,group_num=?,updated_at=? WHERE agent_user_id=?", number(state["user_id"]), number(state["group_id"]), number(state["group_num"]), time.Now().Unix(), agentID)
	return err
}

func (a *App) applyRegistrationAutomation(ctx context.Context, tx *sql.Tx, uid int64, autoUser, autoGroup, userTask, groupTask M) (int64, error) {
	var e error
	if number(autoUser["status"]) == 1 {
		valid := []int64{}
		for _, id := range ids(autoUser["user_ids"]) {
			if _, err := one(ctx, tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", id); err == nil {
				valid = append(valid, id)
			} else if err != sql.ErrNoRows {
				return 0, err
			}
		}
		cs := nextCustomerService(valid, number(userTask["user_id"]))
		if cs > 0 {
			userTask["user_id"] = cs
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
		gid = number(groupTask["group_id"])
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
			seq := number(groupTask["group_num"])
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
			groupTask["group_num"] = seq
		}
		if _, e = insert(ctx, tx, a.t("group_user"), M{"group_id": gid, "user_id": uid, "role": 3, "invite_id": owner, "status": 1, "create_time": time.Now().Unix()}); e != nil {
			return 0, e
		}
		groupTask["group_id"] = gid
	}
	return gid, nil
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
	// Lock order: registration row, chatInfo row, mentor setting row,
	// global autoTask row, then mentor state row.
	automation, e := a.registrationAutomation(ctx, tx, inviterID)
	if e != nil {
		return 0, e
	}
	var globalRow, globalTask, agentTask M
	if automation.UserInherited || automation.GroupInherited {
		globalRow, globalTask, e = a.loadGlobalAutoTask(ctx, tx)
		if e != nil {
			return 0, e
		}
	}
	if automation.AgentUserID > 0 && (!automation.UserInherited || !automation.GroupInherited) {
		agentTask, e = a.loadAgentAutoState(ctx, tx, automation.AgentUserID)
		if e != nil {
			return 0, e
		}
	}
	userTask, groupTask := globalTask, globalTask
	if !automation.UserInherited {
		userTask = agentTask
	}
	if !automation.GroupInherited {
		groupTask = agentTask
	}
	if userTask == nil || groupTask == nil {
		return 0, fmt.Errorf("registration automation state is missing")
	}
	gid, e := a.applyRegistrationAutomation(ctx, tx, uid, automation.AutoUser, automation.AutoGroup, userTask, groupTask)
	if e != nil {
		return 0, e
	}
	if globalTask != nil {
		if e = update(ctx, tx, a.t("config"), M{"value": js(globalTask)}, "id=?", globalRow["id"]); e != nil {
			return 0, e
		}
	}
	if agentTask != nil {
		if e = a.saveAgentAutoState(ctx, tx, automation.AgentUserID, agentTask); e != nil {
			return 0, e
		}
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	if gid > 0 {
		a.groupEvent(ctx, gid, "addGroupUser", M{"group_id": fmt.Sprintf("group-%d", gid), "user_id": uid})
	}
	return uid, nil
}
