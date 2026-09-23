package server

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

type registrationAutomation struct {
	AgentUserID    int64
	AutoUser       M
	AutoGroup      M
	UserInherited  bool
	GroupInherited bool
}

func (a *App) mentorEnabled(ctx context.Context, db DB, userID int64) (bool, error) {
	_, err := one(ctx, db, "SELECT u.user_id FROM "+a.t("user")+" u JOIN "+a.t("imgo_admin_role")+" r ON r.role_id=u.admin_role_id WHERE u.user_id=? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (a *App) agentSetting(ctx context.Context, db DB, agentID int64) (M, M, bool, bool, error) {
	row, err := one(ctx, db, "SELECT auto_add_user,auto_add_group FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, true, true, nil
	}
	if err != nil {
		return nil, nil, false, false, err
	}
	return obj(row["auto_add_user"]), obj(row["auto_add_group"]), row["auto_add_user"] == nil, row["auto_add_group"] == nil, nil
}

func (a *App) registrationAutomation(ctx context.Context, db DB, inviterID int64) (registrationAutomation, error) {
	result := registrationAutomation{UserInherited: true, GroupInherited: true}
	if inviterID > 0 {
		direct, err := a.mentorEnabled(ctx, db, inviterID)
		if err != nil {
			return result, err
		}
		if direct {
			result.AgentUserID = inviterID
		} else {
			result.AgentUserID, err = a.nearestAgent(ctx, db, inviterID)
			if err != nil {
				return result, err
			}
		}
	}
	if result.AgentUserID > 0 {
		var err error
		result.AutoUser, result.AutoGroup, result.UserInherited, result.GroupInherited, err = a.agentSetting(ctx, db, result.AgentUserID)
		if err != nil {
			return result, err
		}
	}
	if result.UserInherited || result.GroupInherited {
		chat := a.config(ctx, "chatInfo")
		if result.UserInherited {
			result.AutoUser = obj(chat["autoAddUser"])
		}
		if result.GroupInherited {
			result.AutoGroup = obj(chat["autoAddGroup"])
		}
	}
	return result, nil
}

func (a *App) validateAgentSettingUser(ctx context.Context, agentID, userID int64) error {
	if userID < 1 {
		return clientError{"账号无效", 400}
	}
	if userID != agentID {
		if err := a.requireScopedUser(ctx, a.db, adminScope{AgentUserID: agentID}, userID); err != nil {
			return err
		}
	}
	_, err := one(ctx, a.db, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return clientError{"账号不可用", 400}
	}
	return err
}

func validAutoStatus(v M) error {
	if str(v["status"]) != "0" && str(v["status"]) != "1" {
		return clientError{"自动配置开关无效", 400}
	}
	return nil
}

func agentCustomerIDs(v M) ([]int64, error) {
	raw, present := v["user_ids"]
	if !present && number(v["status"]) == 0 {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, clientError{"自动客服账号列表无效", 400}
	}
	for _, value := range values {
		id, err := strconv.ParseInt(str(value), 10, 64)
		if err != nil || id < 1 {
			return nil, clientError{"自动客服账号无效", 400}
		}
	}
	return ids(values), nil
}

func (a *App) manageAgentSetting(r *request) (any, error) {
	agentID := r.n("agent_user_id")
	if agentID < 1 {
		return nil, deny()
	}
	enabled, err := a.mentorEnabled(r.ctx(), a.db, agentID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, deny()
	}
	switch action(r) {
	case "detail":
		user, group, userInherited, groupInherited, err := a.agentSetting(r.ctx(), a.db, agentID)
		if err != nil {
			return nil, err
		}
		chat := a.config(r.ctx(), "chatInfo")
		return M{"agent_user_id": agentID, "inherit_auto_user": userInherited, "inherit_auto_group": groupInherited,
			"auto_add_user": user, "auto_add_group": group,
			"global_auto_add_user": obj(chat["autoAddUser"]), "global_auto_add_group": obj(chat["autoAddGroup"])}, nil
	case "save":
		userInherited, ok := r.p["inherit_auto_user"].(bool)
		if !ok {
			return nil, clientError{"自动客服继承标志无效", 400}
		}
		groupInherited, ok := r.p["inherit_auto_group"].(bool)
		if !ok {
			return nil, clientError{"自动群聊继承标志无效", 400}
		}
		var userValue, groupValue any
		if !userInherited {
			user := obj(r.p["auto_add_user"])
			if len(user) == 0 {
				return nil, clientError{"自动客服配置无效", 400}
			}
			if err := validAutoStatus(user); err != nil {
				return nil, err
			}
			customerIDs, err := agentCustomerIDs(user)
			if err != nil {
				return nil, err
			}
			for _, uid := range customerIDs {
				if err := a.validateAgentSettingUser(r.ctx(), agentID, uid); err != nil {
					return nil, err
				}
			}
			userValue = js(user)
		}
		if !groupInherited {
			group := obj(r.p["auto_add_group"])
			if len(group) == 0 {
				return nil, clientError{"自动群聊配置无效", 400}
			}
			if err := validAutoStatus(group); err != nil {
				return nil, err
			}
			if limit, ok := group["userMax"]; ok {
				if err := validateAutoGroupUserMax(limit); err != nil {
					return nil, err
				}
			}
			if owner := number(group["owner_uid"]); owner > 0 {
				if err := a.validateAgentSettingUser(r.ctx(), agentID, owner); err != nil {
					return nil, err
				}
			} else if number(group["status"]) == 1 {
				return nil, clientError{"自动建群的群主无效", 400}
			}
			groupValue = js(group)
		}
		_, err = a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_agent_setting")+" (agent_user_id,auto_add_user,auto_add_group,updated_by,updated_at) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE auto_add_user=VALUES(auto_add_user),auto_add_group=VALUES(auto_add_group),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)", agentID, userValue, groupValue, r.uid(), time.Now().Unix())
		return nil, err
	}
	return nil, r.fail("未知操作")
}
