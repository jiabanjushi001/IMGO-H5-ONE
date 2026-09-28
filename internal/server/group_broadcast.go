package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
)

type groupBroadcastDelivery struct {
	From M
	To   M
}

func containsID(list []int64, id int64) bool {
	for _, value := range list {
		if value == id {
			return true
		}
	}
	return false
}

func chooseBroadcastCustomer(configured []int64, assigned int64, offset int) int64 {
	if containsID(configured, assigned) {
		return assigned
	}
	if len(configured) == 0 {
		return 0
	}
	return configured[offset%len(configured)]
}

func (a *App) activeBroadcastCustomers(ctx context.Context, configured M) ([]M, error) {
	if number(configured["status"]) != 1 {
		return nil, nil
	}
	userIDs := ids(configured["user_ids"])
	if len(userIDs) == 0 {
		return nil, nil
	}
	args := values(userIDs)
	return rows(ctx, a.db,
		"SELECT user_id,realname,avatar,sex,motto,name_py,role FROM "+a.t("user")+
			" WHERE user_id IN ("+marks(len(args))+") AND status=1 AND delete_time=0 ORDER BY user_id", args...)
}

func (a *App) broadcastAgentMap(ctx context.Context, members []M) (map[int64]int64, error) {
	result := map[int64]int64{}
	lookup := []int64{}
	for _, member := range members {
		uid := number(member["user_id"])
		if number(member["agent_mode"]) == 1 && number(member["role_status"]) == 1 {
			result[uid] = uid
		} else {
			lookup = append(lookup, uid)
		}
	}
	if len(lookup) == 0 {
		return result, nil
	}
	args := values(lookup)
	paths, err := rows(ctx, a.db,
		"SELECT p.descendant_user_id,p.ancestor_user_id,p.depth FROM "+a.t("imgo_referral_path")+" p "+
			"JOIN "+a.t("user")+" agent ON agent.user_id=p.ancestor_user_id "+
			"JOIN "+a.t("imgo_admin_role")+" role ON role.role_id=agent.admin_role_id "+
			"WHERE p.descendant_user_id IN ("+marks(len(args))+") AND p.ancestor_user_id<>p.descendant_user_id "+
			"AND agent.status=1 AND agent.delete_time=0 AND role.status=1 AND role.agent_mode=1 "+
			"ORDER BY p.descendant_user_id,p.depth,p.ancestor_user_id", args...)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		uid := number(path["descendant_user_id"])
		if _, exists := result[uid]; !exists {
			result[uid] = number(path["ancestor_user_id"])
		}
	}
	return result, nil
}

func (a *App) effectiveBroadcastCustomers(ctx context.Context, agentID int64) ([]M, error) {
	configured := M(nil)
	if agentID > 0 {
		user, _, inherited, _, err := a.agentSetting(ctx, a.db, agentID)
		if err != nil {
			return nil, err
		}
		if !inherited {
			configured = user
		}
	}
	if configured == nil {
		configured = obj(a.config(ctx, "chatInfo")["autoAddUser"])
	}
	return a.activeBroadcastCustomers(ctx, configured)
}

func (a *App) ensureBroadcastFriend(ctx context.Context, tx *sql.Tx, from, to, now int64) error {
	for _, pair := range [][2]int64{{from, to}, {to, from}} {
		friend, err := one(ctx, tx,
			"SELECT friend_id FROM "+a.t("friend")+" WHERE create_user=? AND friend_user_id=? ORDER BY friend_id LIMIT 1 FOR UPDATE",
			pair[0], pair[1])
		if errors.Is(err, sql.ErrNoRows) {
			if _, err = insert(ctx, tx, a.t("friend"), M{"create_user": pair[0], "friend_user_id": pair[1], "status": 1, "create_time": now}); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err = update(ctx, tx, a.t("friend"), M{"status": 1, "delete_time": 0}, "friend_id=?", friend["friend_id"]); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) persistBroadcast(ctx context.Context, deliveries []groupBroadcastDelivery, content string) ([]M, error) {
	if len(deliveries) == 0 {
		return nil, nil
	}
	sort.Slice(deliveries, func(i, j int) bool {
		return chatKey(number(deliveries[i].From["user_id"]), number(deliveries[i].To["user_id"]), 0) <
			chatKey(number(deliveries[j].From["user_id"]), number(deliveries[j].To["user_id"]), 0)
	})
	enc, err := encryptContent(a.cfg.ChatKey, content)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	messages := make([]M, 0, len(deliveries))
	for _, delivery := range deliveries {
		from, to := number(delivery.From["user_id"]), number(delivery.To["user_id"])
		key := chatKey(from, to, 0)
		if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_chat_lock")+" (chat_identify) VALUES (?) ON DUPLICATE KEY UPDATE chat_identify=VALUES(chat_identify)", key); err != nil {
			return nil, err
		}
		if _, err = one(ctx, tx, "SELECT chat_identify FROM "+a.t("imgo_chat_lock")+" WHERE chat_identify=? FOR UPDATE", key); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE "+a.t("message")+" SET is_last=0 WHERE chat_identify=? AND is_last=1", key); err != nil {
			return nil, err
		}
		if number(delivery.To["cs_uid"]) != from {
			if err = update(ctx, tx, a.t("user"), M{"cs_uid": from}, "user_id=?", to); err != nil {
				return nil, err
			}
		}
		if err = a.ensureBroadcastFriend(ctx, tx, from, to, now); err != nil {
			return nil, err
		}
		message := M{"id": randomID()[:32], "from_user": from, "to_user": to, "content": enc, "chat_identify": key,
			"type": "text", "is_group": 0, "is_read": 0, "is_last": 1, "create_time": now, "status": 1,
			"at": "", "pid": 0, "file_id": 0, "file_cate": 0, "file_size": 0, "file_name": "", "extends": "{}"}
		mid, insertErr := insert(ctx, tx, a.t("message"), message)
		if insertErr != nil {
			return nil, insertErr
		}
		message["msg_id"] = mid
		messages = append(messages, message)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (a *App) publishBroadcast(ctx context.Context, deliveries []groupBroadcastDelivery, messages []M) {
	for index, message := range messages {
		data, err := a.serialize(ctx, message)
		if err != nil {
			a.log.Warn("serialize group broadcast", "error", err)
			continue
		}
		delivery := deliveries[index]
		from, to := number(delivery.From["user_id"]), number(delivery.To["user_id"])
		data["toUser"] = to
		data["contactInfo"] = a.safeUser(delivery.To)
		peer := M{}
		for key, value := range data {
			peer[key] = value
		}
		peer["toContactId"] = from
		peer["contactInfo"] = a.safeUser(delivery.From)
		obj(peer["contactInfo"])["lastContent"] = data["content"]
		obj(peer["contactInfo"])["lastSendTime"] = data["sendTime"]
		a.hub.send([]int64{to}, "simple", peer)
		a.hub.send([]int64{from}, "simple", data)
	}
}

func (a *App) broadcastGroupMembers(r *request, scope adminScope, gid int64) (any, error) {
	members, err := r.list(
		"SELECT u.user_id,u.realname,u.avatar,u.sex,u.motto,u.name_py,u.role,u.cs_uid,"+
			"COALESCE(ar.agent_mode,0) agent_mode,COALESCE(ar.status,0) role_status "+
			"FROM "+a.t("group_user")+" gu JOIN "+a.t("user")+" u ON u.user_id=gu.user_id "+
			"LEFT JOIN "+a.t("imgo_admin_role")+" ar ON ar.role_id=u.admin_role_id "+
			"WHERE gu.group_id=? AND gu.status=1 AND u.status=1 AND u.delete_time=0 ORDER BY u.user_id", gid)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, clientError{"群内没有可发送的成员", 400}
	}
	return a.broadcastRecipients(r, scope, members)
}

func (a *App) broadcastManagedUserOptions(r *request, scope adminScope) (any, error) {
	base := "SELECT u.user_id,u.realname,u.account,u.avatar FROM " + a.t("user") + " u "
	if scope.Global {
		return r.list(base+"WHERE u.status=1 AND u.delete_time=0 AND u.user_id<>? ORDER BY u.user_id", r.uid())
	}
	return r.list(base+"JOIN "+a.t("imgo_referral_path")+" rp ON rp.descendant_user_id=u.user_id "+
		"WHERE rp.ancestor_user_id=? AND rp.depth>0 AND u.status=1 AND u.delete_time=0 ORDER BY u.user_id", scope.AgentUserID)
}

func (a *App) broadcastManagedUsers(r *request, scope adminScope) (any, error) {
	selected := ids(r.p["user_ids"])
	if len(selected) == 0 {
		return nil, clientError{"请至少选择一个群发用户", 400}
	}
	if len(selected) > 5000 {
		return nil, clientError{"单次群发最多选择 5000 个用户", 400}
	}
	args := values(selected)
	base := "SELECT u.user_id,u.realname,u.avatar,u.sex,u.motto,u.name_py,u.role,u.cs_uid," +
		"COALESCE(ar.agent_mode,0) agent_mode,COALESCE(ar.status,0) role_status " +
		"FROM " + a.t("user") + " u LEFT JOIN " + a.t("imgo_admin_role") + " ar ON ar.role_id=u.admin_role_id "
	var (
		members []M
		err     error
	)
	if scope.Global {
		queryArgs := append(args, r.uid())
		members, err = r.list(base+"WHERE u.user_id IN ("+marks(len(args))+") AND u.status=1 AND u.delete_time=0 AND u.user_id<>? ORDER BY u.user_id", queryArgs...)
	} else {
		queryArgs := append(args, scope.AgentUserID)
		members, err = r.list(base+"JOIN "+a.t("imgo_referral_path")+" rp ON rp.descendant_user_id=u.user_id "+
			"WHERE u.user_id IN ("+marks(len(args))+") AND rp.ancestor_user_id=? AND rp.depth>0 "+
			"AND u.status=1 AND u.delete_time=0 ORDER BY u.user_id", queryArgs...)
	}
	if err != nil {
		return nil, err
	}
	if len(members) != len(selected) {
		return nil, clientError{"选择的用户不存在或不在你的管理范围内", 403}
	}
	return a.broadcastRecipients(r, scope, members)
}

func (a *App) broadcastRecipients(r *request, scope adminScope, members []M) (any, error) {
	content := sanitizeText(r.s("content"))
	if content == "" || len([]rune(content)) > 2048 {
		return nil, clientError{"群发内容为空或超过 2048 个字符", 400}
	}
	if err := a.moderate(r.ctx(), content, "chat_detection"); err != nil {
		return nil, err
	}

	agentByUser := map[int64]int64{}
	var err error
	if scope.Global {
		agentByUser, err = a.broadcastAgentMap(r.ctx(), members)
		if err != nil {
			return nil, err
		}
	} else {
		for _, member := range members {
			uid := number(member["user_id"])
			if err = a.requireScopedUser(r.ctx(), a.db, scope, uid); err != nil {
				return nil, err
			}
			agentByUser[uid] = scope.AgentUserID
		}
	}

	customersByAgent := map[int64][]M{}
	customerIDsByAgent := map[int64][]int64{}
	customerCount := map[int64]bool{}
	agentCount := map[int64]bool{}
	deliveries := []groupBroadcastDelivery{}
	skippedNoCustomer, skippedCustomer := 0, 0
	offsetByAgent := map[int64]int{}
	for _, member := range members {
		uid := number(member["user_id"])
		agentID := agentByUser[uid]
		agentCount[agentID] = true
		customers, loaded := customersByAgent[agentID]
		if !loaded {
			customers, err = a.effectiveBroadcastCustomers(r.ctx(), agentID)
			if err != nil {
				return nil, err
			}
			customersByAgent[agentID] = customers
			customerIDsByAgent[agentID] = make([]int64, 0, len(customers))
			for _, customer := range customers {
				id := number(customer["user_id"])
				customerIDsByAgent[agentID] = append(customerIDsByAgent[agentID], id)
				customerCount[id] = true
			}
		}
		customerIDs := customerIDsByAgent[agentID]
		if len(customerIDs) == 0 {
			skippedNoCustomer++
			continue
		}
		if containsID(customerIDs, uid) {
			skippedCustomer++
			continue
		}
		fromID := chooseBroadcastCustomer(customerIDs, number(member["cs_uid"]), offsetByAgent[agentID])
		offsetByAgent[agentID]++
		var sender M
		for _, customer := range customers {
			if number(customer["user_id"]) == fromID {
				sender = customer
				break
			}
		}
		if sender == nil {
			skippedNoCustomer++
			continue
		}
		deliveries = append(deliveries, groupBroadcastDelivery{From: sender, To: member})
	}
	if len(deliveries) == 0 {
		return nil, clientError{"没有可用的客服账号，请先在导师设置或聊天设置中开启自动客服", 400}
	}
	messages, err := a.persistBroadcast(r.ctx(), deliveries, content)
	if err != nil {
		return nil, err
	}
	a.publishBroadcast(r.ctx(), deliveries, messages)
	return M{"sent": len(messages), "skipped": skippedNoCustomer + skippedCustomer,
		"skipped_no_customer": skippedNoCustomer, "skipped_customer": skippedCustomer,
		"customer_count": len(customerCount), "agent_count": len(agentCount),
		"message": fmt.Sprintf("群发完成：成功 %d 人，跳过 %d 人", len(messages), skippedNoCustomer+skippedCustomer)}, nil
}
