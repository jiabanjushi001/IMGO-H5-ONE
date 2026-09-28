package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSystemAlertMessageContainsAuditIdentityAndAmount(t *testing.T) {
	message := formatSystemAlertMessage(systemAlertEvent{
		Type:        "recharge",
		OccurredAt:  time.Date(2026, 9, 25, 14, 30, 0, 0, time.FixedZone("CST", 8*60*60)),
		IP:          "203.0.113.18",
		Actor:       "administrator (#1)",
		Target:      "member007 (#7)",
		AmountCents: 12345,
		Detail:      "订单 #22",
	})
	for _, required := range []string{
		"操作类型：后台：充值入账",
		"IP：203.0.113.18",
		"操作人：administrator (#1)",
		"操作对象：member007 (#7)",
		"金额：¥123.45",
		"订单 #22",
	} {
		if !strings.Contains(message, required) {
			t.Fatalf("message missing %q:\n%s", required, message)
		}
	}
}

func TestTelegramGroupIDCommand(t *testing.T) {
	for _, command := range []string{
		"@my_alert_bot id",
		"  @MyAlertBot   ID  ",
		"/id",
		"/id@my_alert_bot",
	} {
		if !isTelegramGroupIDCommand(command) {
			t.Fatalf("expected %q to be recognized", command)
		}
	}
	for _, message := range []string{"id", "@my_alert_bot hello", "/ids", "hello id"} {
		if isTelegramGroupIDCommand(message) {
			t.Fatalf("expected %q to be ignored", message)
		}
	}
}

func TestTelegramGroupIDReplyOnlyInGroups(t *testing.T) {
	update := telegramUpdate{UpdateID: 42, Message: &telegramMessage{
		Text: "@my_alert_bot id",
		Chat: telegramChat{ID: -1009988, Type: "supergroup", Title: "运营报警群"},
		From: telegramUser{IsBot: false},
	}}
	chatID, message, ok := telegramGroupIDReply(update)
	if !ok || chatID != "-1009988" || !strings.Contains(message, "群 Chat ID：-1009988") {
		t.Fatalf("unexpected group ID reply: ok=%v chat=%q message=%q", ok, chatID, message)
	}

	update.Message.Chat.Type = "private"
	if _, _, ok := telegramGroupIDReply(update); ok {
		t.Fatal("private messages must not receive a group ID reply")
	}
	update.Message.Chat.Type = "group"
	update.Message.From.IsBot = true
	if _, _, ok := telegramGroupIDReply(update); ok {
		t.Fatal("bot-authored messages must be ignored")
	}
}

func TestFetchTelegramUpdatesUsesLongPollingOffset(t *testing.T) {
	var gotPath string
	var gotOffset int64
	var gotTimeout int
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		var payload struct {
			Offset  int64 `json:"offset"`
			Timeout int   `json:"timeout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		gotOffset, gotTimeout = payload.Offset, payload.Timeout
		body := `{"ok":true,"result":[{"update_id":77,"message":{"message_id":3,"text":"@my_alert_bot id","chat":{"id":-1009988,"type":"supergroup","title":"ops"},"from":{"is_bot":false}}}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	updates, err := fetchTelegramUpdates(context.Background(), client, "https://api.telegram.test", "123456:secret", 75, 15)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/bot123456:secret/getUpdates" || gotOffset != 75 || gotTimeout != 15 {
		t.Fatalf("unexpected getUpdates request: path=%s offset=%d timeout=%d", gotPath, gotOffset, gotTimeout)
	}
	if len(updates) != 1 || updates[0].UpdateID != 77 || updates[0].Message == nil {
		t.Fatalf("unexpected updates: %#v", updates)
	}
}

func TestSystemAlertMessageUsesUnknownFallbacks(t *testing.T) {
	message := formatSystemAlertMessage(systemAlertEvent{Type: "admin_login", OccurredAt: time.Unix(0, 0)})
	for _, required := range []string{"IP：未知", "操作人：未知", "操作对象：未知", "操作类型：后台：登录"} {
		if !strings.Contains(message, required) {
			t.Fatalf("message missing %q: %s", required, message)
		}
	}
	if strings.Contains(message, "金额：") {
		t.Fatal("non-financial event must not invent an amount")
	}
}

func TestSendTelegramMessageUsesBotAPIJSON(t *testing.T) {
	var gotPath, gotChatID, gotText string
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		gotChatID, gotText = payload["chat_id"], payload["text"]
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`))}, nil
	})}

	err := sendTelegramMessage(context.Background(), client, "https://api.telegram.test", "123456:secret", "-1009988", "test alert")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/bot123456:secret/sendMessage" || gotChatID != "-1009988" || gotText != "test alert" {
		t.Fatalf("unexpected Telegram request: path=%s chat=%s text=%s", gotPath, gotChatID, gotText)
	}
}

func TestSystemAlertConfigValidationAndVisibility(t *testing.T) {
	cfg, err := normalizeSystemAlertConfig(M{
		"enabled":   1,
		"bot_token": "123456:secret",
		"chat_id":   "-1009988",
		"events":    []any{"admin_login", "user_login", "recharge", "admin_login", "unknown"},
	}, systemAlertConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Events) != 3 || cfg.Events[0] != "admin_login" || cfg.Events[1] != "user_login" || cfg.Events[2] != "recharge" {
		t.Fatalf("unexpected normalized events: %#v", cfg.Events)
	}
	public := publicSystemAlertConfig(cfg)
	if public["bot_token"] != "123456:secret" || public["bot_token_set"] != true {
		t.Fatalf("super-admin config must display the saved bot token: %#v", public)
	}
	options := map[string]string{}
	for _, option := range public["options"].([]M) {
		options[str(option["value"])] = str(option["group"])
	}
	for event, group := range map[string]string{
		"admin_login": "后台操作", "admin_withdraw": "后台操作",
		"user_login": "用户端操作", "user_register": "用户端操作", "user_withdraw": "用户端操作",
	} {
		if options[event] != group {
			t.Fatalf("event %s group = %q, want %q", event, options[event], group)
		}
	}
	if events := targetValues(defaultConfig("systemAlert")["events"]); len(events) != 0 {
		t.Fatalf("all alert choices must start unchecked: %#v", events)
	}
	if legacy := decodeSystemAlertConfig(M{"enabled": 0, "events": []any{"login", "register", "withdraw", "recharge"}}); len(legacy.Events) != 0 {
		t.Fatalf("legacy broad choices must not auto-select granular alerts: %#v", legacy.Events)
	}
	if _, err := normalizeSystemAlertConfig(M{"enabled": 1, "chat_id": "-100", "events": []any{"login"}}, systemAlertConfig{}); err == nil {
		t.Fatal("enabled configuration without bot token must fail")
	}
}

func TestSystemAlertRoutesAreSuperAdminOnly(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{
		"/manage/config/getsystemalert",
		"/manage/config/setsystemalert",
		"/manage/config/testsystemalert",
	} {
		route, ok := a.routes[path]
		if !ok || !route.super {
			t.Fatalf("route %s must be registered as super-admin only: %#v", path, route)
		}
	}
}
