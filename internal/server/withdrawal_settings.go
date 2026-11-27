package server

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
)

type withdrawalSettings struct {
	Enabled          bool
	MinYuan          int64
	PasswordRequired bool
}

var withdrawalMethodTypes = []string{paymentTypeBank, paymentTypeAlipay, paymentTypeWechat}

func defaultWithdrawalMethods() M {
	return M{paymentTypeBank: true, paymentTypeAlipay: true, paymentTypeWechat: true}
}

func normalizeWithdrawalMethods(value M) (M, error) {
	methods := M{}
	for _, typeName := range withdrawalMethodTypes {
		enabled, valid := agentSettingBool(value[typeName])
		if !valid {
			return nil, clientError{"提现方式开关无效", 400}
		}
		methods[typeName] = enabled
	}
	return methods, nil
}

func storedWithdrawalMethods(value any) M {
	methods := defaultWithdrawalMethods()
	// JSON columns are decoded by rows() into map[string]any. Passing that
	// object through str() turns it into "map[...]", which is not JSON and
	// silently resets every switch to the default enabled state.
	stored := obj(value)
	for _, typeName := range withdrawalMethodTypes {
		if enabled, valid := agentSettingBool(stored[typeName]); valid {
			methods[typeName] = enabled
		}
	}
	return methods
}

func withdrawalMethodEnabled(methods M, typeName string) bool {
	enabled, valid := agentSettingBool(methods[typeName])
	return valid && enabled
}

func (a *App) agentWithdrawalMethods(ctx context.Context, db DB, agentID int64) (M, error) {
	row, err := one(ctx, db, "SELECT withdrawal_methods FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultWithdrawalMethods(), nil
	}
	if err != nil {
		return nil, err
	}
	return storedWithdrawalMethods(row["withdrawal_methods"]), nil
}

func (a *App) withdrawalMethodsForUser(ctx context.Context, db DB, userID int64) (M, error) {
	row, err := one(ctx, db,
		"SELECT s.withdrawal_methods FROM "+a.t("imgo_referral_path")+" p "+
			"JOIN "+a.t("user")+" u ON u.user_id=p.ancestor_user_id "+
			"JOIN "+a.t("imgo_admin_role")+" r ON r.role_id=u.admin_role_id "+
			"LEFT JOIN "+a.t("imgo_agent_setting")+" s ON s.agent_user_id=p.ancestor_user_id "+
			"WHERE p.descendant_user_id=? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 "+
			"ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultWithdrawalMethods(), nil
	}
	if err != nil {
		return nil, err
	}
	return storedWithdrawalMethods(row["withdrawal_methods"]), nil
}

func decodeWithdrawalSettings(value M) (withdrawalSettings, error) {
	settings := withdrawalSettings{}
	switch strings.ToLower(strings.TrimSpace(str(value["enabled"]))) {
	case "1", "true":
		settings.Enabled = true
	case "0", "false":
		settings.Enabled = false
	default:
		return settings, errors.New("提现开关无效")
	}
	minimum, err := strconv.ParseInt(strings.TrimSpace(str(value["minYuan"])), 10, 64)
	if err != nil || minimum < 1 || minimum > 100000000 {
		return settings, errors.New("最低提现金额必须是1到100000000元的整数")
	}
	settings.MinYuan = minimum
	if raw, exists := value["passwordRequired"]; exists {
		switch strings.ToLower(strings.TrimSpace(str(raw))) {
		case "1", "true":
			settings.PasswordRequired = true
		case "0", "false":
			settings.PasswordRequired = false
		default:
			return settings, errors.New("提现密码开关无效")
		}
	}
	return settings, nil
}

func withdrawalSettingsValue(settings withdrawalSettings) M {
	enabled := 0
	if settings.Enabled {
		enabled = 1
	}
	passwordRequired := 0
	if settings.PasswordRequired {
		passwordRequired = 1
	}
	return M{"enabled": enabled, "minYuan": settings.MinYuan, "passwordRequired": passwordRequired}
}

func normalizeWithdrawalConfig(value M) (M, error) {
	settings, err := decodeWithdrawalSettings(value)
	if err != nil {
		return nil, clientError{err.Error(), 400}
	}
	return withdrawalSettingsValue(settings), nil
}

func (a *App) withdrawalSettings(ctx context.Context) withdrawalSettings {
	settings, err := decodeWithdrawalSettings(a.config(ctx, "withdrawal"))
	if err == nil {
		return settings
	}
	settings, _ = decodeWithdrawalSettings(defaultConfig("withdrawal"))
	return settings
}

// withdrawalSettingsForUser keeps the global minimum amount. The enable switch
// resolves user -> nearest active mentor -> global; the password rule resolves
// nearest active mentor -> global because personal password overrides are not
// exposed.
func (a *App) withdrawalSettingsForUser(ctx context.Context, db DB, userID int64) (withdrawalSettings, error) {
	settings := a.withdrawalSettings(ctx)
	if userID < 1 {
		return settings, nil
	}
	user, err := one(ctx, db, "SELECT withdrawal_enabled FROM "+a.t("user")+" WHERE user_id=? AND delete_time=0", userID)
	if err != nil {
		return settings, err
	}
	personalOverride := user["withdrawal_enabled"] != nil
	if personalOverride {
		settings.Enabled = number(user["withdrawal_enabled"]) == 1
	}
	agentID := int64(0)
	direct, err := a.mentorEnabled(ctx, db, userID)
	if err != nil {
		return settings, err
	}
	if direct {
		agentID = userID
	} else if agentID, err = a.nearestAgent(ctx, db, userID); err != nil {
		return settings, err
	}
	if agentID < 1 {
		return settings, nil
	}
	row, err := one(ctx, db, "SELECT withdrawal_enabled,withdrawal_password_required FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	if errors.Is(err, sql.ErrNoRows) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if !personalOverride && row["withdrawal_enabled"] != nil {
		settings.Enabled = number(row["withdrawal_enabled"]) == 1
	}
	if row["withdrawal_password_required"] != nil {
		settings.PasswordRequired = number(row["withdrawal_password_required"]) == 1
	}
	return settings, nil
}

func (a *App) agentWithdrawalSetting(ctx context.Context, db DB, agentID int64) (bool, bool, error) {
	global := a.withdrawalSettings(ctx)
	row, err := one(ctx, db, "SELECT withdrawal_enabled FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && row["withdrawal_enabled"] == nil {
		return global.Enabled, true, nil
	}
	if err != nil {
		return false, false, err
	}
	return number(row["withdrawal_enabled"]) == 1, false, nil
}

func (a *App) agentWithdrawalPasswordSetting(ctx context.Context, db DB, agentID int64) (bool, bool, error) {
	row, err := one(ctx, db, "SELECT withdrawal_password_required FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	if err == nil && row["withdrawal_password_required"] != nil {
		return number(row["withdrawal_password_required"]) == 1, false, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, false, err
	}
	return a.withdrawalSettings(ctx).PasswordRequired, true, nil
}
