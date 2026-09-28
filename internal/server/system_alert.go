package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const telegramAPIBaseURL = "https://api.telegram.org"

var (
	systemAlertTokenPattern = regexp.MustCompile(`^[0-9]{3,20}:[A-Za-z0-9_-]{6,200}$`)
	systemAlertChatPattern  = regexp.MustCompile(`^(?:-?[0-9]{1,30}|@[A-Za-z][A-Za-z0-9_]{3,63})$`)
	telegramMentionID       = regexp.MustCompile(`(?i)^@[A-Za-z0-9_]{3,64}\s+id$`)
	telegramSlashID         = regexp.MustCompile(`(?i)^/id(?:@[A-Za-z0-9_]{3,64})?$`)
	systemAlertOptions      = []struct{ Value, Label, Group string }{
		{"admin_login", "后台：登录", "后台操作"},
		{"admin_withdraw", "后台：发起提现", "后台操作"},
		{"withdraw_review", "后台：提现审核", "后台操作"},
		{"recharge", "后台：充值入账", "后台操作"},
		{"balance_adjustment", "后台：余额调整", "后台操作"},
		{"member_password", "后台：修改成员密码", "后台操作"},
		{"user_login", "用户端：登录", "用户端操作"},
		{"user_register", "用户端：注册", "用户端操作"},
		{"user_withdraw", "用户端：提交提现", "用户端操作"},
	}
)

type systemAlertConfig struct {
	Enabled  bool     `json:"enabled"`
	BotToken string   `json:"bot_token"`
	ChatID   string   `json:"chat_id"`
	Events   []string `json:"events"`
}

type systemAlertEvent struct {
	Type        string
	OccurredAt  time.Time
	IP          string
	Actor       string
	ActorID     int64
	Target      string
	TargetID    int64
	AmountCents int64
	Detail      string
}

type telegramUser struct {
	IsBot bool `json:"is_bot"`
}

type telegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type telegramMessage struct {
	MessageID int64        `json:"message_id"`
	Text      string       `json:"text"`
	Chat      telegramChat `json:"chat"`
	From      telegramUser `json:"from"`
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

func systemAlertEventKeys() []string {
	keys := make([]string, 0, len(systemAlertOptions))
	for _, option := range systemAlertOptions {
		keys = append(keys, option.Value)
	}
	return keys
}

func decodeSystemAlertConfig(value M) systemAlertConfig {
	events := []string{}
	if number(value["version"]) >= 2 {
		events = normalizeSystemAlertEvents(targetValues(value["events"]))
	}
	return systemAlertConfig{
		Enabled:  number(value["enabled"]) == 1,
		BotToken: strings.TrimSpace(str(value["bot_token"])),
		ChatID:   strings.TrimSpace(str(value["chat_id"])),
		Events:   events,
	}
}

func normalizeSystemAlertEvents(values []string) []string {
	allowed := make(map[string]bool, len(systemAlertOptions))
	for _, option := range systemAlertOptions {
		allowed[option.Value] = true
	}
	seen := map[string]bool{}
	events := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if allowed[value] && !seen[value] {
			seen[value] = true
			events = append(events, value)
		}
	}
	return events
}

func normalizeSystemAlertConfig(value M, previous systemAlertConfig) (systemAlertConfig, error) {
	cfg := previous
	if raw, exists := value["enabled"]; exists {
		cfg.Enabled = number(raw) == 1
	}
	if raw, exists := value["chat_id"]; exists {
		cfg.ChatID = strings.TrimSpace(str(raw))
	}
	if number(value["clear_bot_token"]) == 1 {
		cfg.BotToken = ""
	} else if token := strings.TrimSpace(str(value["bot_token"])); token != "" {
		cfg.BotToken = token
	}
	if raw, exists := value["events"]; exists {
		cfg.Events = normalizeSystemAlertEvents(targetValues(raw))
	}
	if cfg.BotToken != "" && !systemAlertTokenPattern.MatchString(cfg.BotToken) {
		return cfg, clientError{"电报 Bot Token 格式无效", 400}
	}
	if cfg.ChatID != "" && !systemAlertChatPattern.MatchString(cfg.ChatID) {
		return cfg, clientError{"电报群 Chat ID 格式无效", 400}
	}
	if cfg.Enabled {
		if cfg.BotToken == "" || cfg.ChatID == "" {
			return cfg, clientError{"开启系统报警前请填写 Bot Token 和群 Chat ID", 400}
		}
		if len(cfg.Events) == 0 {
			return cfg, clientError{"至少选择一项报警内容", 400}
		}
	}
	return cfg, nil
}

func publicSystemAlertConfig(cfg systemAlertConfig) M {
	options := make([]M, 0, len(systemAlertOptions))
	for _, option := range systemAlertOptions {
		options = append(options, M{"value": option.Value, "label": option.Label, "group": option.Group})
	}
	return M{
		"enabled":       cfg.Enabled,
		"bot_token":     cfg.BotToken,
		"bot_token_set": cfg.BotToken != "",
		"chat_id":       cfg.ChatID,
		"events":        append([]string(nil), cfg.Events...),
		"options":       options,
	}
}

func (a *App) loadSystemAlertConfig(ctx context.Context) systemAlertConfig {
	return decodeSystemAlertConfig(a.config(ctx, "systemAlert"))
}

func (a *App) saveSystemAlertConfig(ctx context.Context, cfg systemAlertConfig, userID int64) error {
	value := M{"version": 2, "enabled": cfg.Enabled, "bot_token": cfg.BotToken, "chat_id": cfg.ChatID, "events": cfg.Events}
	existing, err := one(ctx, a.db, "SELECT id FROM "+a.t("config")+" WHERE name=?", "systemAlert")
	if errors.Is(err, sql.ErrNoRows) {
		_, err = insert(ctx, a.db, a.t("config"), M{"name": "systemAlert", "value": js(value), "create_user": userID, "create_time": time.Now().Unix(), "status": 1})
	} else if err == nil {
		err = update(ctx, a.db, a.t("config"), M{"value": js(value), "update_time": time.Now().Unix(), "status": 1}, "id=?", existing["id"])
	}
	if err == nil {
		a.systemAlertState.Store(cfg)
	}
	return err
}

func (a *App) manageSystemAlert(r *request) (any, error) {
	switch action(r) {
	case "getsystemalert":
		return publicSystemAlertConfig(a.loadSystemAlertConfig(r.ctx())), nil
	case "setsystemalert":
		current := a.loadSystemAlertConfig(r.ctx())
		cfg, err := normalizeSystemAlertConfig(r.p, current)
		if err != nil {
			return nil, err
		}
		if err = a.saveSystemAlertConfig(r.ctx(), cfg, r.uid()); err != nil {
			return nil, err
		}
		return publicSystemAlertConfig(cfg), nil
	case "testsystemalert":
		cfg := a.loadSystemAlertConfig(r.ctx())
		if cfg.BotToken == "" || cfg.ChatID == "" {
			return nil, r.fail("请先保存 Bot Token 和群 Chat ID")
		}
		test := systemAlertEvent{Type: "test", OccurredAt: time.Now(), IP: a.clientIP(r.c), Actor: systemAlertUserIdentity(r.user), Target: "Telegram 报警群 " + cfg.ChatID, Detail: "这是一条连接测试消息"}
		ctx, cancel := context.WithTimeout(r.ctx(), 8*time.Second)
		defer cancel()
		return nil, sendTelegramMessage(ctx, a.systemAlertHTTPClient(), telegramAPIBaseURL, cfg.BotToken, cfg.ChatID, formatSystemAlertMessage(test))
	}
	return nil, r.fail("未知操作")
}

func (a *App) startSystemAlerts() {
	ctx, cancel := context.WithCancel(context.Background())
	a.alertCancel = cancel
	a.alertQueue = make(chan systemAlertEvent, 256)
	a.systemAlertState.Store(decodeSystemAlertConfig(defaultConfig("systemAlert")))
	a.wg.Add(2)
	go func() {
		defer a.wg.Done()
		loadCtx, stop := context.WithTimeout(ctx, 3*time.Second)
		cfg := a.loadSystemAlertConfig(loadCtx)
		stop()
		a.systemAlertState.Store(cfg)
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-a.alertQueue:
				a.deliverSystemAlert(ctx, event)
			}
		}
	}()
	go func() {
		defer a.wg.Done()
		a.pollTelegramGroupID(ctx)
	}()
}

func (a *App) pollTelegramGroupID(ctx context.Context) {
	var activeToken, lastError string
	var offset int64
	for {
		cfg := a.currentSystemAlertConfig()
		token := strings.TrimSpace(cfg.BotToken)
		if token == "" {
			activeToken, lastError, offset = "", "", 0
			if !waitSystemAlertPoll(ctx, 2*time.Second) {
				return
			}
			continue
		}
		if token != activeToken {
			activeToken, lastError, offset = token, "", -1
		}

		timeoutSeconds := 15
		if offset < 0 {
			timeoutSeconds = 0
		}
		updates, err := fetchTelegramUpdates(ctx, a.systemAlertPollingHTTPClient(), telegramAPIBaseURL, token, offset, timeoutSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if err.Error() != lastError {
				a.log.Error("poll Telegram group ID commands failed", "error", err)
				lastError = err.Error()
			}
			if !waitSystemAlertPoll(ctx, 10*time.Second) {
				return
			}
			continue
		}
		lastError = ""
		discardPending := offset < 0
		if discardPending && len(updates) == 0 {
			offset = 0
		}
		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			if discardPending {
				continue
			}
			chatID, reply, ok := telegramGroupIDReply(update)
			if !ok {
				continue
			}
			replyCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			err = sendTelegramMessage(replyCtx, a.systemAlertHTTPClient(), telegramAPIBaseURL, token, chatID, reply)
			cancel()
			if err != nil && ctx.Err() == nil {
				a.log.Error("reply Telegram group ID command failed", "chat_id", chatID, "error", err)
			}
		}
	}
}

func waitSystemAlertPoll(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func isTelegramGroupIDCommand(text string) bool {
	text = strings.TrimSpace(text)
	return telegramMentionID.MatchString(text) || telegramSlashID.MatchString(text)
}

func telegramGroupIDReply(update telegramUpdate) (string, string, bool) {
	message := update.Message
	if message == nil || message.From.IsBot || !isTelegramGroupIDCommand(message.Text) {
		return "", "", false
	}
	chatType := strings.ToLower(strings.TrimSpace(message.Chat.Type))
	if chatType != "group" && chatType != "supergroup" {
		return "", "", false
	}
	chatID := fmt.Sprintf("%d", message.Chat.ID)
	reply := "群 Chat ID：" + chatID
	if title := strings.TrimSpace(message.Chat.Title); title != "" {
		reply += "\n群名称：" + title
	}
	return chatID, reply, true
}

func (a *App) currentSystemAlertConfig() systemAlertConfig {
	if stored := a.systemAlertState.Load(); stored != nil {
		return stored.(systemAlertConfig)
	}
	return decodeSystemAlertConfig(defaultConfig("systemAlert"))
}

func (a *App) queueSystemAlert(event systemAlertEvent) {
	if a.alertQueue == nil {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	select {
	case a.alertQueue <- event:
	default:
		a.log.Error("system alert queue full", "event", event.Type)
	}
}

func (a *App) deliverSystemAlert(parent context.Context, event systemAlertEvent) {
	cfg := a.currentSystemAlertConfig()
	if !cfg.Enabled || !containsSystemAlertEvent(cfg.Events, event.Type) {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	event = a.hydrateSystemAlertEvent(ctx, event)
	if err := sendTelegramMessage(ctx, a.systemAlertHTTPClient(), telegramAPIBaseURL, cfg.BotToken, cfg.ChatID, formatSystemAlertMessage(event)); err != nil && parent.Err() == nil {
		a.log.Error("send system alert failed", "event", event.Type, "error", err)
	}
}

func containsSystemAlertEvent(events []string, event string) bool {
	for _, candidate := range events {
		if candidate == event {
			return true
		}
	}
	return false
}

func (a *App) hydrateSystemAlertEvent(ctx context.Context, event systemAlertEvent) systemAlertEvent {
	load := func(id int64) string {
		if id < 1 {
			return ""
		}
		user, err := one(ctx, a.db, "SELECT user_id,account,realname FROM "+a.t("user")+" WHERE user_id=?", id)
		if err != nil {
			return ""
		}
		return systemAlertUserIdentity(user)
	}
	if strings.TrimSpace(event.Actor) == "" {
		event.Actor = load(event.ActorID)
	}
	if strings.TrimSpace(event.Target) == "" {
		if event.TargetID == event.ActorID && event.Actor != "" {
			event.Target = event.Actor
		} else {
			event.Target = load(event.TargetID)
		}
	}
	return event
}

func systemAlertUserIdentity(user M) string {
	id := number(user["user_id"])
	name := strings.TrimSpace(str(user["account"]))
	if name == "" {
		name = strings.TrimSpace(str(user["realname"]))
	}
	if name == "" && id > 0 {
		name = "用户"
	}
	if id > 0 {
		return fmt.Sprintf("%s (#%d)", name, id)
	}
	return name
}

func formatSystemAlertMessage(event systemAlertEvent) string {
	label := "系统操作"
	for _, option := range systemAlertOptions {
		if option.Value == event.Type {
			label = option.Label
			break
		}
	}
	if event.Type == "test" {
		label = "报警连接测试"
	}
	value := func(text string) string {
		if strings.TrimSpace(text) == "" {
			return "未知"
		}
		return strings.TrimSpace(text)
	}
	when := event.OccurredAt
	if when.IsZero() {
		when = time.Now()
	}
	lines := []string{
		"[Imgo 系统报警]",
		"时间：" + when.Format("2006-01-02 15:04:05 MST"),
		"操作类型：" + label,
		"IP：" + value(event.IP),
		"操作人：" + value(event.Actor),
		"操作对象：" + value(event.Target),
	}
	if event.AmountCents != 0 {
		amount := event.AmountCents
		sign := ""
		if amount < 0 {
			sign, amount = "-", -amount
		}
		lines = append(lines, fmt.Sprintf("金额：%s¥%d.%02d", sign, amount/100, amount%100))
	}
	if detail := strings.TrimSpace(event.Detail); detail != "" {
		lines = append(lines, "详情："+detail)
	}
	message := strings.Join(lines, "\n")
	if runes := []rune(message); len(runes) > 4000 {
		message = string(runes[:4000])
	}
	return message
}

func (a *App) systemAlertHTTPClient() *http.Client {
	if a.outbound != nil {
		return a.outbound
	}
	return &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func (a *App) systemAlertPollingHTTPClient() *http.Client {
	if a.outbound != nil {
		return a.outbound
	}
	return &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func fetchTelegramUpdates(ctx context.Context, client *http.Client, baseURL, token string, offset int64, timeoutSeconds int) ([]telegramUpdate, error) {
	payload, err := json.Marshal(M{
		"offset":          offset,
		"limit":           100,
		"timeout":         timeoutSeconds,
		"allowed_updates": []string{"message"},
	})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/bot" + token + "/getUpdates"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return nil, clientError{"电报服务连接失败", 503}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, clientError{fmt.Sprintf("电报服务返回 HTTP %d", response.StatusCode), 502}
	}
	var result struct {
		OK          bool             `json:"ok"`
		Description string           `json:"description"`
		Result      []telegramUpdate `json:"result"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return nil, clientError{"电报服务返回格式无效", 502}
	}
	if !result.OK {
		description := strings.TrimSpace(result.Description)
		if len([]rune(description)) > 160 {
			description = string([]rune(description)[:160])
		}
		if description == "" {
			description = "请检查 Bot Token，并确认未配置 webhook"
		}
		return nil, clientError{"电报轮询失败：" + description, 502}
	}
	return result.Result, nil
}

func sendTelegramMessage(ctx context.Context, client *http.Client, baseURL, token, chatID, text string) error {
	payload, err := json.Marshal(M{"chat_id": chatID, "text": text})
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/bot" + token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return clientError{"电报服务连接失败", 503}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return clientError{fmt.Sprintf("电报服务返回 HTTP %d", response.StatusCode), 502}
	}
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return clientError{"电报服务返回格式无效", 502}
	}
	if !result.OK {
		description := strings.TrimSpace(result.Description)
		if len([]rune(description)) > 160 {
			description = string([]rune(description)[:160])
		}
		if description == "" {
			description = "请检查 Bot Token 和群 Chat ID"
		}
		return clientError{"电报发送失败：" + description, 502}
	}
	return nil
}
