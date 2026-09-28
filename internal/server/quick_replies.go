package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	maxQuickReplies     = 50
	maxQuickReplyLength = 500
	maxQuickRepliesSize = 10000
)

func quickReplyItems(value any) ([]string, error) {
	var raw []any
	switch value := value.(type) {
	case []any:
		raw = value
	case []string:
		raw = make([]any, len(value))
		for index := range value {
			raw[index] = value[index]
		}
	case string:
		if strings.TrimSpace(value) == "" {
			return []string{}, nil
		}
		if err := json.Unmarshal([]byte(value), &raw); err != nil {
			return nil, clientError{"快捷用语格式无效", 400}
		}
	default:
		return nil, clientError{"快捷用语格式无效", 400}
	}
	if len(raw) > maxQuickReplies {
		return nil, clientError{"快捷用语最多保存 50 条", 400}
	}
	items := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	total := 0
	for _, value := range raw {
		item := strings.TrimSpace(str(value))
		if item == "" || seen[item] {
			continue
		}
		if len([]rune(item)) > maxQuickReplyLength {
			return nil, clientError{"单条快捷用语不能超过 500 个字符", 400}
		}
		total += len([]byte(item))
		if total > maxQuickRepliesSize {
			return nil, clientError{"快捷用语总内容过长", 400}
		}
		seen[item] = true
		items = append(items, item)
	}
	return items, nil
}

func (a *App) quickReplyOwner(r *request) (int64, bool, error) {
	userID := r.uid()
	if userID == 1 {
		return userID, false, nil
	}
	isMentor, err := a.mentorEnabled(r.ctx(), a.db, userID)
	if err != nil {
		return 0, false, err
	}
	if isMentor {
		return userID, true, nil
	}
	mentorID, err := a.nearestAgent(r.ctx(), a.db, userID)
	if err != nil {
		return 0, false, err
	}
	if mentorID > 0 {
		return mentorID, false, nil
	}
	return userID, false, nil
}

func (a *App) manageQuickReplies(r *request) (any, error) {
	ownerID, canManage, err := a.quickReplyOwner(r)
	if err != nil {
		return nil, err
	}
	row, err := r.one("SELECT quick_replies FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", ownerID)
	if errors.Is(err, sql.ErrNoRows) || row["quick_replies"] == nil {
		return M{"items": []string{}, "owner_user_id": ownerID, "can_manage": canManage}, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := quickReplyItems(row["quick_replies"])
	if err != nil {
		return nil, err
	}
	return M{"items": items, "owner_user_id": ownerID, "can_manage": canManage}, nil
}

func (a *App) saveQuickReplies(r *request) (any, error) {
	ownerID, canManage, err := a.quickReplyOwner(r)
	if err != nil {
		return nil, err
	}
	if !canManage || ownerID != r.uid() {
		return nil, deny()
	}
	items, err := quickReplyItems(r.p["items"])
	if err != nil {
		return nil, err
	}
	_, err = a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_agent_setting")+" (agent_user_id,quick_replies,updated_by,updated_at) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE quick_replies=VALUES(quick_replies),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)", ownerID, js(items), r.uid(), time.Now().Unix())
	if err != nil {
		return nil, err
	}
	return M{"items": items}, nil
}
