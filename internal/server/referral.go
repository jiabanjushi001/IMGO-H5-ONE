package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const inviteCodeCount = 1_000_000

func validInviteCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func validLegacyInviteCode(code string) bool {
	if len(code) != 5 {
		return false
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func (a *App) referralURL(code string) string {
	base := strings.TrimSpace(a.cfg.H5URL)
	if base == "" {
		base = strings.SplitN(strings.TrimSpace(a.cfg.InviteURL), "#", 2)[0]
	}
	if base == "" {
		base = a.cfg.BaseURL
	}
	return strings.TrimRight(base, "/") + "/#/pages/login/register?inviteCode=" + code
}

func newInviteCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(inviteCodeCount))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// The database's unique index owns collision resolution, including concurrent registrations.
func (a *App) ensureInviteCode(ctx context.Context, db DB, userID int64) (string, error) {
	row, err := one(ctx, db, "SELECT invite_code FROM "+a.t("imgo_referral")+" WHERE user_id=?", userID)
	if err == nil {
		return str(row["invite_code"]), nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	for attempt := 0; attempt < 32; attempt++ {
		code, err := newInviteCode()
		if err != nil {
			return "", err
		}
		_, err = db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_referral")+" (user_id,invite_code,parent_user_id,created_at) VALUES (?,?,0,?)", userID, code, time.Now().Unix())
		if err == nil {
			return code, nil
		}
		var duplicate *mysql.MySQLError
		if !errors.As(err, &duplicate) || duplicate.Number != 1062 {
			return "", err
		}
		// A duplicate user ID means another request already created this code.
		if row, lookupErr := one(ctx, db, "SELECT invite_code FROM "+a.t("imgo_referral")+" WHERE user_id=?", userID); lookupErr == nil {
			return str(row["invite_code"]), nil
		} else if lookupErr != sql.ErrNoRows {
			return "", lookupErr
		}
	}
	return "", errors.New("无法生成唯一邀请码")
}

// Check first for a useful error message; the unique index also closes races.
func (a *App) setMemberInviteCode(r *request, userID int64, whereUser string) (any, error) {
	code := strings.TrimSpace(r.s("invite_code"))
	if !validInviteCode(code) {
		return nil, clientError{"邀请码必须是 6 位纯数字", 400}
	}
	if userID < 1 {
		return nil, clientError{"成员ID无效", 400}
	}
	if _, err := r.one("SELECT user_id FROM "+a.t("user")+" WHERE "+whereUser, userID); err != nil {
		return nil, err
	}
	current, err := r.one("SELECT invite_code FROM "+a.t("imgo_referral")+" WHERE user_id=?", userID)
	if err == sql.ErrNoRows {
		return nil, clientError{"该成员没有邀请码记录", 400}
	}
	if err != nil {
		return nil, err
	}
	if str(current["invite_code"]) == code {
		return M{"invite_code": code}, nil
	}
	owner, err := r.one("SELECT user_id FROM "+a.t("imgo_referral")+" WHERE invite_code=?", code)
	if err == nil && number(owner["user_id"]) != userID {
		return nil, clientError{"邀请码已被其他成员使用", 400}
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	_, err = a.db.ExecContext(r.ctx(), "UPDATE "+a.t("imgo_referral")+" SET invite_code=? WHERE user_id=?", code, userID)
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return nil, clientError{"邀请码已被其他成员使用", 400}
		}
		return nil, err
	}
	return M{"invite_code": code}, nil
}

func (a *App) resolveInviter(ctx context.Context, db DB, raw string) (int64, error) {
	code := strings.TrimSpace(raw)
	if code == "" {
		return 0, nil
	}
	var inviter int64
	if validInviteCode(code) {
		row, err := one(ctx, db, "SELECT r.user_id FROM "+a.t("imgo_referral")+" r JOIN "+a.t("user")+" u ON u.user_id=r.user_id WHERE r.invite_code=? AND u.status=1 AND u.delete_time=0", code)
		if err == nil {
			inviter = number(row["user_id"])
		} else if err != sql.ErrNoRows {
			return 0, err
		}
	} else if validLegacyInviteCode(code) {
		row, err := one(ctx, db, "SELECT l.user_id FROM "+a.t("imgo_referral_legacy_code")+" l JOIN "+a.t("user")+" u ON u.user_id=l.user_id WHERE l.invite_code=? AND u.status=1 AND u.delete_time=0", code)
		if err == nil {
			inviter = number(row["user_id"])
		} else if err != sql.ErrNoRows {
			return 0, err
		}
	} else if cached := a.get("invite:"+code, false); cached != nil {
		inviter = number(cached)
	}
	if inviter < 1 {
		return 0, clientError{"邀请码无效", 400}
	}
	if _, err := one(ctx, db, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", inviter); err != nil {
		if err == sql.ErrNoRows {
			return 0, clientError{"邀请人账号不可用", 400}
		}
		return 0, err
	}
	return inviter, nil
}

func (a *App) bindInviter(ctx context.Context, tx *sql.Tx, userID, inviterID int64) error {
	if inviterID == 0 {
		return nil
	}
	if userID == inviterID {
		return clientError{"不能邀请自己", 400}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE "+a.t("imgo_referral")+" SET parent_user_id=? WHERE user_id=? AND parent_user_id=0", inviterID, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_referral_path")+" (ancestor_user_id,descendant_user_id,depth) VALUES (?,?,1)", inviterID, userID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_referral_path")+" (ancestor_user_id,descendant_user_id,depth) SELECT ancestor_user_id,?,depth+1 FROM "+a.t("imgo_referral_path")+" WHERE descendant_user_id=?", userID, inviterID)
	return err
}

// A run remains active before today's sign-in if yesterday was signed in.
func inviteStreak(dates []string, now time.Time) int64 {
	if len(dates) == 0 {
		return 0
	}
	expected, _ := time.ParseInLocation("2006-01-02", checkInDate(now), checkInZone)
	if dates[0] != expected.Format("2006-01-02") {
		expected = expected.AddDate(0, 0, -1)
		if dates[0] != expected.Format("2006-01-02") {
			return 0
		}
	}
	var streak int64
	for _, day := range dates {
		if day != expected.Format("2006-01-02") {
			break
		}
		streak++
		expected = expected.AddDate(0, 0, -1)
	}
	return streak
}

// The H5 sees only direct invites and check-in streak; the full referral tree
// remains admin-only.
func (a *App) inviteStatus(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	row, err := r.one("SELECT invite_code FROM "+a.t("imgo_referral")+" WHERE user_id=?", r.uid())
	if err != nil {
		return nil, err
	}
	count, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_referral_path")+" p JOIN "+a.t("user")+" u ON u.user_id=p.descendant_user_id WHERE p.ancestor_user_id=? AND p.depth=1 AND u.delete_time=0", r.uid())
	if err != nil {
		return nil, err
	}
	days, err := r.list("SELECT DATE_FORMAT(sign_date,'%Y-%m-%d') sign_date FROM "+a.t("imgo_check_in")+" WHERE user_id=? ORDER BY sign_date DESC", r.uid())
	if err != nil {
		return nil, err
	}
	signDates := make([]string, 0, len(days))
	for _, day := range days {
		signDates = append(signDates, str(day["sign_date"]))
	}
	code := str(row["invite_code"])
	return M{
		"invite_code":  code,
		"invite_url":   a.referralURL(code),
		"direct_count": number(count["n"]),
		"streak_days":  inviteStreak(signDates, time.Now()),
	}, nil
}

// One code query and one aggregate query for a paginated member list.
func (a *App) memberReferralSummaries(r *request, users []M) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[int64]M, len(users))
	userIDs := make([]int64, 0, len(users))
	for _, user := range users {
		id := number(user["user_id"])
		byID[id] = user
		userIDs = append(userIDs, id)
		user["invite_code"] = ""
		user["direct_invite_count"] = int64(0)
		user["team_count"] = int64(0)
	}
	placeholders := marks(len(userIDs))
	codes, err := r.list("SELECT user_id,invite_code FROM "+a.t("imgo_referral")+" WHERE user_id IN ("+placeholders+")", values(userIDs)...)
	if err != nil {
		return err
	}
	for _, code := range codes {
		if user := byID[number(code["user_id"])]; user != nil {
			user["invite_code"] = str(code["invite_code"])
		}
	}
	counts, err := r.list("SELECT p.ancestor_user_id,COUNT(*) team_count,COALESCE(SUM(p.depth=1),0) direct_count FROM "+a.t("imgo_referral_path")+" p JOIN "+a.t("user")+" u ON u.user_id=p.descendant_user_id WHERE p.ancestor_user_id IN ("+placeholders+") AND u.delete_time=0 GROUP BY p.ancestor_user_id", values(userIDs)...)
	if err != nil {
		return err
	}
	for _, count := range counts {
		if user := byID[number(count["ancestor_user_id"])]; user != nil {
			user["direct_invite_count"] = number(count["direct_count"])
			user["team_count"] = number(count["team_count"])
		}
	}
	return nil
}
