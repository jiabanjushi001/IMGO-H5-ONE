package server

import (
	"context"
	"database/sql"
	"errors"
)

type adminScope struct {
	Global        bool
	AgentUserID   int64
	referralTable string
}

func (a *App) adminScope(ctx context.Context, user M) (adminScope, error) {
	userID := number(user["user_id"])
	if userID == 1 {
		return adminScope{Global: true}, nil
	}
	roleID := number(user["admin_role_id"])
	if userID < 1 || roleID < 1 {
		return adminScope{}, deny()
	}
	role, err := one(ctx, a.db, "SELECT agent_mode FROM "+a.t("imgo_admin_role")+" WHERE role_id=? AND status=1", roleID)
	if errors.Is(err, sql.ErrNoRows) {
		return adminScope{}, deny()
	}
	if err != nil {
		return adminScope{}, err
	}
	if number(role["agent_mode"]) == 0 {
		return adminScope{Global: true}, nil
	}
	return adminScope{AgentUserID: userID, referralTable: a.t("imgo_referral_path")}, nil
}

func (s adminScope) userPredicate(alias string) (string, []any) {
	if s.Global {
		return "1=1", nil
	}
	if !safeSQLAlias(alias) {
		panic("invalid SQL alias for admin scope")
	}
	if s.referralTable == "" {
		panic("missing referral table for admin scope")
	}
	return "EXISTS (SELECT 1 FROM " + s.referralTable + " scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=" + alias + ".user_id)", []any{s.AgentUserID}
}

func safeSQLAlias(alias string) bool {
	if alias == "" {
		return false
	}
	for i, ch := range alias {
		if ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || i > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func (a *App) requireScopedUser(ctx context.Context, db DB, scope adminScope, userID int64) error {
	if userID < 1 || !scope.Global && scope.AgentUserID < 1 {
		return deny()
	}
	if scope.referralTable == "" {
		scope.referralTable = a.t("imgo_referral_path")
	}
	predicate, args := scope.userPredicate("u")
	params := append([]any{userID}, args...)
	_, err := one(ctx, db, "SELECT 1 FROM "+a.t("user")+" u WHERE u.user_id=? AND u.delete_time=0 AND "+predicate, params...)
	if errors.Is(err, sql.ErrNoRows) {
		return deny()
	}
	return err
}

func (a *App) nearestAgent(ctx context.Context, db DB, userID int64) (int64, error) {
	if userID < 1 {
		return 0, nil
	}
	row, err := one(ctx, db,
		"SELECT p.ancestor_user_id FROM "+a.t("imgo_referral_path")+" p "+
			"JOIN "+a.t("user")+" u ON u.user_id=p.ancestor_user_id "+
			"JOIN "+a.t("imgo_admin_role")+" r ON r.role_id=u.admin_role_id "+
			"WHERE p.descendant_user_id=? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 "+
			"ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return number(row["ancestor_user_id"]), nil
}
