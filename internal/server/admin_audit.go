package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type auditDefinition struct {
	Category   string
	Action     string
	Risk       string
	TargetType string
}

var adminAuditDefinitions = map[string]auditDefinition{
	"/manage/user/add":              {"成员", "添加", "medium", "用户"},
	"/manage/user/batchadd":         {"成员", "批量创建", "high", "用户"},
	"/manage/user/edit":             {"成员", "编辑", "medium", "用户"},
	"/manage/user/setremark":        {"成员", "修改备注", "low", "用户"},
	"/manage/user/setinvitecode":    {"成员", "重置邀请码", "medium", "用户"},
	"/manage/user/del":              {"成员", "删除", "high", "用户"},
	"/manage/user/setstatus":        {"成员", "修改状态", "high", "用户"},
	"/manage/user/editpassword":     {"成员", "修改密码", "high", "用户"},
	"/manage/user/setrole":          {"成员", "设置角色", "high", "用户"},
	"/manage/user/googleauthbind":   {"成员", "设置谷歌", "high", "用户"},
	"/manage/user/googleauthunbind": {"成员", "解绑谷歌", "high", "用户"},
	"/manage/user/broadcast":        {"成员", "群发消息", "medium", "用户"},
	"/manage/agentsetting/save":     {"成员", "导师设置", "medium", "导师"},
	"/manage/group/changeowner":     {"群聊", "转让群主", "high", "群聊"},
	"/manage/group/del":             {"群聊", "删除群聊", "high", "群聊"},
	"/manage/group/addgroupuser":    {"群聊", "添加成员", "medium", "群聊"},
	"/manage/group/delgroupuser":    {"群聊", "移除成员", "medium", "群聊"},
	"/manage/group/setmanager":      {"群聊", "设置管理员", "medium", "群聊"},
	"/manage/group/broadcast":       {"群聊", "群发消息", "medium", "群聊"},
	"/manage/group/setnumberjoin":   {"群聊", "设置群号加入", "medium", "群聊"},
	"/manage/config/setconfig":      {"设置", "修改系统设置", "high", "系统"},
	"/manage/config/setsecurity":    {"设置", "修改安全策略", "critical", "系统"},
	"/manage/config/sendtestemail":  {"设置", "发送测试邮件", "low", "系统"},
	"/manage/index/clearmessage":    {"概况", "清理消息", "critical", "系统"},
	"/manage/index/delnotice":       {"概况", "删除公告", "medium", "公告"},
	"/manage/index/publishnotice":   {"概况", "发布公告", "medium", "公告"},
	"/manage/message/dealmsg":       {"消息", "处理消息", "medium", "消息"},
	"/manage/task/starttask":        {"任务", "启动任务", "medium", "任务"},
	"/manage/task/stoptask":         {"任务", "停止任务", "medium", "任务"},
	"/manage/task/settaskconfig":    {"任务", "修改任务设置", "high", "任务"},
	"/manage/task/cleartasklog":     {"任务", "清理任务日志", "high", "任务"},
	"/manage/role/save":             {"角色", "保存角色", "high", "角色"},
	"/manage/role/setstatus":        {"角色", "修改状态", "high", "角色"},
	"/manage/role/del":              {"角色", "删除角色", "critical", "角色"},
	"/manage/bank/edit":             {"绑卡", "审核绑卡", "high", "用户"},
	"/manage/wallet/credit":         {"财务", "调整余额", "critical", "用户"},
	"/manage/wallet/review":         {"财务", "审核提现", "critical", "提现单"},
	"/manage/wallet/freeze":         {"财务", "冻结提现", "critical", "提现单"},
	"/manage/wallet/recharge":       {"财务", "充值", "critical", "用户"},
	"/manage/wallet/withdraw":       {"财务", "提现", "critical", "用户"},
}

func auditTargetID(path string, r *request, data any) string {
	keys := []string{"user_id", "group_id", "role_id", "withdrawal_id", "order_id", "msg_id", "id"}
	if path == "/manage/user/add" {
		if result, ok := data.(M); ok && number(result["user_id"]) > 0 {
			return str(result["user_id"])
		}
	}
	for _, key := range keys {
		if value := strings.TrimSpace(r.s(key)); value != "" && value != "0" {
			return value
		}
	}
	return ""
}

func (a *App) auditDetail(def auditDefinition, r *request, data any) string {
	var detail string
	path := effectivePath(r.c)
	switch path {
	case "/manage/user/batchadd":
		detail = "批量创建 " + str(r.p["count"]) + " 个成员"
	case "/manage/config/setsecurity":
		if _, exists := r.p["google_auth_enabled"]; exists {
			if number(obj(data)["google_auth_enabled"]) == 1 {
				detail = "开启全局谷歌验证"
			} else {
				detail = "关闭全局谷歌验证"
			}
		} else if _, exists := r.p["ip_whitelist_enabled"]; exists {
			result := obj(data)
			rules := targetValues(result["ip_whitelist"])
			beforeEnabled := number(r.auditBefore["ip_whitelist_enabled"]) == 1
			afterEnabled := number(result["ip_whitelist_enabled"]) == 1
			if afterEnabled && !beforeEnabled {
				detail = "开启后台 IP 白名单"
			} else if !afterEnabled && beforeEnabled {
				detail = "关闭后台 IP 白名单"
			} else {
				detail = "更新后台 IP 白名单"
			}
			beforeRules := targetValues(r.auditBefore["ip_whitelist"])
			added, removed := auditRuleDifference(rules, beforeRules), auditRuleDifference(beforeRules, rules)
			if len(added) > 0 {
				detail += "；新增：" + strings.Join(added, "、")
			}
			if len(removed) > 0 {
				detail += "；删除：" + strings.Join(removed, "、")
			}
			if len(rules) > 0 {
				detail += "；当前允许：" + strings.Join(rules, "、")
			} else {
				detail += "；当前允许：空"
			}
		}
	case "/manage/user/googleauthbind", "/manage/user/googleauthunbind":
		userID := r.n("user_id")
		account := ""
		if userID > 0 {
			if user, err := one(r.ctx(), a.db, "SELECT account FROM "+a.t("user")+" WHERE user_id=?", userID); err == nil {
				account = str(user["account"])
			}
		}
		verb := "绑定"
		if path == "/manage/user/googleauthunbind" {
			verb = "解绑"
		}
		detail = verb + "用户谷歌验证"
		if account != "" {
			detail += "：" + account
		} else if userID > 0 {
			detail += "：用户 #" + str(userID)
		}
	}
	if detail != "" {
		if len([]rune(detail)) > 200 {
			detail = string([]rune(detail)[:200])
		}
		return detail
	}
	switch def.Category {
	case "成员":
		detail = strings.TrimSpace(r.s("account"))
	case "群聊":
		detail = strings.TrimSpace(r.s("name"))
	case "角色":
		detail = strings.TrimSpace(r.s("name"))
	case "财务":
		if amount := strings.TrimSpace(r.s("amount")); amount != "" {
			detail = "金额：" + amount
		}
	}
	if len([]rune(detail)) > 200 {
		detail = string([]rune(detail)[:200])
	}
	return detail
}

func auditRuleDifference(base, compare []string) []string {
	known := make(map[string]bool, len(compare))
	for _, item := range compare {
		known[item] = true
	}
	result := make([]string, 0)
	for _, item := range base {
		if !known[item] {
			result = append(result, item)
		}
	}
	return result
}

func (a *App) auditRoleName(ctx context.Context, user M) string {
	if number(user["user_id"]) == 1 {
		return "超级管理员"
	}
	roleID := number(user["admin_role_id"])
	if roleID < 1 {
		return "普通用户"
	}
	row, err := one(ctx, a.db, "SELECT name FROM "+a.t("imgo_admin_role")+" WHERE role_id=?", roleID)
	if err != nil {
		return "后台角色"
	}
	return str(row["name"])
}

func auditSensitiveKey(key string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	if strings.Contains(normalized, "password") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "token") || strings.Contains(normalized, "credential") {
		return true
	}
	switch normalized {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "session", "sessionid", "ximsign", "xapikey", "pass", "code", "googlecode", "captcha", "accountcipher", "receiptaccount", "bankaccount":
		return true
	}
	return false
}

func redactAuditValue(value any) any {
	switch typed := value.(type) {
	case M:
		result := M{}
		for key, item := range typed {
			if auditSensitiveKey(key) {
				result[key] = "[已隐藏]"
			} else {
				result[key] = redactAuditValue(item)
			}
		}
		return result
	case map[string]any:
		result := map[string]any{}
		for key, item := range typed {
			if auditSensitiveKey(key) {
				result[key] = "[已隐藏]"
			} else {
				result[key] = redactAuditValue(item)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = redactAuditValue(item)
		}
		return result
	default:
		return value
	}
}

func auditRequestSnapshot(r *request) (string, string, string, string) {
	method, path := "", ""
	headers := map[string]any{}
	if r != nil && r.c != nil && r.c.Request != nil {
		method = r.c.Request.Method
		path = effectivePath(r.c)
		for key, values := range r.c.Request.Header {
			if auditSensitiveKey(key) {
				headers[key] = "[已隐藏]"
			} else if len(values) == 1 {
				headers[key] = values[0]
			} else {
				headers[key] = values
			}
		}
	}
	headerJSON, _ := json.Marshal(headers)
	dataJSON, _ := json.Marshal(redactAuditValue(r.p))
	return method, path, string(headerJSON), string(dataJSON)
}

func (a *App) writeAdminAudit(ctx context.Context, r *request, user M, def auditDefinition, targetID, targetName, detail string, success bool, errorMessage string) {
	if number(user["user_id"]) < 1 {
		return
	}
	ip, userAgent := a.clientIP(r.c), r.c.Request.UserAgent()
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	if len([]rune(errorMessage)) > 200 {
		errorMessage = string([]rune(errorMessage)[:200])
	}
	status := 0
	if success {
		status = 1
	}
	method, path, headers, requestData := auditRequestSnapshot(r)
	_, err := a.db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_admin_audit_log")+" (actor_user_id,actor_account,actor_name,actor_role,category,action,risk_level,target_type,target_id,target_name,detail,ip,user_agent,request_method,request_path,request_headers,request_data,status,error_message,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		number(user["user_id"]), str(user["account"]), str(user["realname"]), a.auditRoleName(ctx, user), def.Category, def.Action, def.Risk, def.TargetType, targetID, targetName, detail, ip, userAgent, method, path, headers, requestData, status, errorMessage, time.Now().Unix())
	if err != nil {
		a.log.Error("write admin audit failed", "error", err, "category", def.Category, "action", def.Action)
	}
}

func (a *App) auditTargetName(ctx context.Context, def auditDefinition, targetID string) string {
	if number(targetID) < 1 || (def.TargetType != "用户" && def.TargetType != "导师" && def.TargetType != "账号") {
		return ""
	}
	user, err := one(ctx, a.db, "SELECT account FROM "+a.t("user")+" WHERE user_id=?", number(targetID))
	if err != nil {
		return ""
	}
	return str(user["account"])
}

func (a *App) recordAdminAudit(r *request, path string, data any, operationErr error) {
	def, ok := adminAuditDefinitions[path]
	if !ok || r.uid() < 1 {
		return
	}
	errorMessage := ""
	if operationErr != nil {
		errorMessage = operationErr.Error()
	}
	targetID := auditTargetID(path, r, data)
	a.writeAdminAudit(r.ctx(), r, r.user, def, targetID, a.auditTargetName(r.ctx(), def, targetID), a.auditDetail(def, r, data), operationErr == nil, errorMessage)
}

func (a *App) recordAdminLogin(r *request, user M) {
	a.writeAdminAudit(r.ctx(), r, user, auditDefinition{"登录", "后台登录", "medium", "账号"}, str(user["user_id"]), str(user["account"]), "", true, "")
}

func (a *App) manageAudit(r *request) (any, error) {
	if action(r) != "index" {
		return nil, r.fail("未知操作")
	}
	where := "1=1"
	args := []any{}
	if r.uid() != 1 {
		where += " AND actor_user_id=?"
		args = append(args, r.uid())
	}
	if category := strings.TrimSpace(r.s("category")); category != "" {
		where += " AND category=?"
		args = append(args, category)
	}
	if risk := strings.TrimSpace(r.s("risk_level")); risk != "" {
		switch risk {
		case "low", "medium", "high", "critical":
		default:
			return nil, r.fail("敏感等级无效")
		}
		where += " AND risk_level=?"
		args = append(args, risk)
	}
	if statusText := strings.TrimSpace(r.s("status")); statusText != "" {
		if statusText != "0" && statusText != "1" {
			return nil, r.fail("操作状态无效")
		}
		where += " AND status=?"
		args = append(args, number(statusText))
	}
	if keyword := strings.TrimSpace(r.s("keywords")); keyword != "" {
		like := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyword) + "%"
		where += " AND (actor_account LIKE ? ESCAPE '!' OR actor_name LIKE ? ESCAPE '!' OR action LIKE ? ESCAPE '!' OR target_id LIKE ? ESCAPE '!' OR target_name LIKE ? ESCAPE '!' OR ip LIKE ? ESCAPE '!')"
		args = append(args, like, like, like, like, like, like)
	}
	count, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_admin_audit_log")+" WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	r.count = number(count["n"])
	limit, offset := r.pagination()
	items, err := r.list("SELECT audit_id,actor_user_id,actor_account,actor_name,actor_role,category,action,risk_level,target_type,target_id,target_name,detail,ip,status,error_message,request_method,request_path,request_headers,request_data,created_at FROM "+a.t("imgo_admin_audit_log")+" WHERE "+where+" ORDER BY audit_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return items, nil
}
