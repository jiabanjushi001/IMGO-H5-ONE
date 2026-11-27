package server

const (
	memberWithdrawalInherit  = "inherit"
	memberWithdrawalEnabled  = "enabled"
	memberWithdrawalDisabled = "disabled"
)

// setMemberWithdrawalPermission stores only the personal override. A NULL
// value deliberately restores inheritance from the mentor team/global switch.
func (a *App) setMemberWithdrawalPermission(r *request, userID int64, whereUser string) (any, error) {
	if userID <= 1 {
		return nil, r.fail("该账号不支持设置个人提现权限")
	}
	mode := r.s("mode")
	var value any
	switch mode {
	case memberWithdrawalInherit:
		value = nil
	case memberWithdrawalEnabled:
		value = 1
	case memberWithdrawalDisabled:
		value = 0
	default:
		return nil, r.fail("个人提现权限参数无效")
	}
	if _, err := r.one("SELECT user_id FROM "+a.t("user")+" WHERE "+whereUser, userID); err != nil {
		return nil, err
	}
	_, err := a.db.ExecContext(r.ctx(), "UPDATE "+a.t("user")+" SET withdrawal_enabled=? WHERE "+whereUser, value, userID)
	if err != nil {
		return nil, err
	}
	return M{"withdrawal_enabled": value, "mode": mode}, nil
}
