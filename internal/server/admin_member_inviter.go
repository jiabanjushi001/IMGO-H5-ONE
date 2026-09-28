package server

import "strings"

// The mentor's parent is derived from authenticated scope, never from the form.
func (a *App) memberAddInviter(r *request, db DB, scope adminScope) (int64, error) {
	code := strings.TrimSpace(r.s("parent_invite_code"))
	if code == "" {
		if !scope.Global {
			return scope.AgentUserID, nil
		}
		return 0, nil
	}
	if !validInviteCode(code) {
		return 0, r.fail("上级邀请码必须是6位数字")
	}
	uid, err := a.resolveInviter(r.ctx(), db, code)
	if err != nil {
		return 0, err
	}
	if !scope.Global && uid != scope.AgentUserID {
		return 0, r.fail("导师只能使用本人的邀请码添加成员")
	}
	return uid, nil
}
