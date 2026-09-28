package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"sort"
	"strings"
	"time"
)

func splitLoginIPRules(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t', ' ':
			return true
		}
		return false
	})
}

func normalizeLoginIPRules(raw string) ([]string, error) {
	seen := map[string]bool{}
	rules := make([]string, 0)
	for _, item := range splitLoginIPRules(raw) {
		var canonical string
		if ip := net.ParseIP(item); ip != nil {
			canonical = ip.String()
		} else if _, network, err := net.ParseCIDR(item); err == nil {
			canonical = network.String()
		} else {
			return nil, clientError{"IP 白名单格式错误：" + item, 400}
		}
		if !seen[canonical] {
			seen[canonical] = true
			rules = append(rules, canonical)
		}
		if len(rules) > 200 {
			return nil, clientError{"IP 白名单最多允许 200 条", 400}
		}
	}
	sort.Strings(rules)
	return rules, nil
}

func loginIPMatches(ipText string, rules []string) bool {
	ip := net.ParseIP(strings.Trim(strings.TrimSpace(ipText), "[]"))
	if ip == nil {
		return false
	}
	for _, rule := range rules {
		if allowed := net.ParseIP(rule); allowed != nil {
			if allowed.Equal(ip) {
				return true
			}
			continue
		}
		if _, network, err := net.ParseCIDR(rule); err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func (a *App) loginIPWhitelistEnabled() bool { return a.loginIPOn.Load() }

func (a *App) loginIPWhitelistRules() []string {
	value := a.loginIPRules.Load()
	if value == nil {
		return []string{}
	}
	rules, _ := value.([]string)
	return append([]string(nil), rules...)
}

func (a *App) storeLoginIPWhitelist(enabled bool, rules []string) {
	a.loginIPRules.Store(append([]string(nil), rules...))
	a.loginIPOn.Store(enabled)
}

func (a *App) loginIPAllowed(ip string) bool {
	return loginIPMatches(ip, a.loginIPWhitelistRules())
}

func (a *App) refreshLoginIPWhitelist(ctx context.Context, db DB) error {
	row, err := one(ctx, db, "SELECT ip_whitelist_enabled,ip_whitelist FROM "+a.t("imgo_security_setting")+" WHERE id=1")
	if errors.Is(err, sql.ErrNoRows) {
		a.storeLoginIPWhitelist(false, nil)
		return nil
	}
	if err != nil {
		return err
	}
	var stored []string
	if raw := strings.TrimSpace(str(row["ip_whitelist"])); raw != "" {
		_ = json.Unmarshal([]byte(raw), &stored)
	}
	rules, err := normalizeLoginIPRules(strings.Join(stored, "\n"))
	if err != nil {
		return err
	}
	a.storeLoginIPWhitelist(number(row["ip_whitelist_enabled"]) == 1, rules)
	return nil
}

func (a *App) setLoginIPWhitelist(ctx context.Context, actorID int64, currentSID, currentIP string, enabled bool, rules []string) ([]string, error) {
	if enabled {
		if len(rules) == 0 {
			return nil, clientError{"请至少填写一个后台登录白名单 IP", 400}
		}
		if !loginIPMatches(currentIP, rules) {
			return nil, clientError{"当前 IP（" + currentIP + "）不在白名单中，为避免锁定后台，无法开启", 400}
		}
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if currentSID != "" {
		if _, err = tx.ExecContext(ctx, "UPDATE "+a.t("imgo_session")+" SET login_ip=?,admin_login=1 WHERE sid=? AND user_id=?", currentIP, currentSID, actorID); err != nil {
			return nil, err
		}
	}
	revoked := []string{}
	if enabled {
		sessions, queryErr := rows(ctx, tx, "SELECT sid,login_ip FROM "+a.t("imgo_session")+" WHERE admin_login=1")
		if queryErr != nil {
			return nil, queryErr
		}
		for _, session := range sessions {
			sid := str(session["sid"])
			if sid == "" || loginIPMatches(str(session["login_ip"]), rules) {
				continue
			}
			if _, err = tx.ExecContext(ctx, "DELETE FROM "+a.t("imgo_session")+" WHERE sid=?", sid); err != nil {
				return nil, err
			}
			revoked = append(revoked, sid)
		}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_security_setting")+" (id,ip_whitelist_enabled,ip_whitelist,updated_by,updated_at) VALUES (1,?,?,?,?) ON DUPLICATE KEY UPDATE ip_whitelist_enabled=VALUES(ip_whitelist_enabled),ip_whitelist=VALUES(ip_whitelist),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)", enabled, string(encoded), actorID, time.Now().Unix()); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return revoked, nil
}
