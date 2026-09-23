package server

import (
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var checkInZone = time.FixedZone("China Standard Time", 8*60*60)

func checkInDate(now time.Time) string {
	return now.In(checkInZone).Format("2006-01-02")
}

func (a *App) checkInStatus(r *request, day string) (M, error) {
	row, err := r.one("SELECT COUNT(*) total,COALESCE(SUM(sign_date=?),0) signed_today FROM "+a.t("imgo_check_in")+" WHERE user_id=?", day, r.uid())
	if err != nil {
		return nil, err
	}
	return M{"date": day, "signed_today": number(row["signed_today"]) > 0, "total_days": number(row["total"])}, nil
}

// Attach summaries for just the visible member page, never one query per member.
func (a *App) memberCheckInSummaries(r *request, users []M) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[int64]M, len(users))
	args := make([]any, 0, len(users)+1)
	args = append(args, checkInDate(time.Now()))
	for _, u := range users {
		u["checkin_days"] = int64(0)
		u["checkin_today"] = false
		u["checkin_last_date"] = ""
		id := number(u["user_id"])
		byID[id] = u
		args = append(args, id)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(users)), ",")
	query := "SELECT user_id,COUNT(*) total_days,DATE_FORMAT(MAX(sign_date),'%Y-%m-%d') last_date,COALESCE(SUM(sign_date=?),0) signed_today FROM " + a.t("imgo_check_in") + " WHERE user_id IN (" + placeholders + ") GROUP BY user_id"
	summaries, err := r.list(query, args...)
	if err != nil {
		return err
	}
	for _, summary := range summaries {
		if u := byID[number(summary["user_id"])]; u != nil {
			u["checkin_days"] = number(summary["total_days"])
			u["checkin_today"] = number(summary["signed_today"]) > 0
			u["checkin_last_date"] = str(summary["last_date"])
		}
	}
	return nil
}

func (a *App) checkIn(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	now := time.Now()
	day := checkInDate(now)
	switch action(r) {
	case "status":
		return a.checkInStatus(r, day)
	case "submit":
		_, err := a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_check_in")+" (user_id,sign_date,created_at) VALUES (?,?,?)", r.uid(), day, now.Unix())
		alreadySigned := false
		if err != nil {
			var duplicate *mysql.MySQLError
			if !errors.As(err, &duplicate) || duplicate.Number != 1062 {
				return nil, err
			}
			alreadySigned = true
		}
		result, err := a.checkInStatus(r, day)
		if err != nil {
			return nil, err
		}
		result["already_signed"] = alreadySigned
		return result, nil
	default:
		return nil, r.fail("未知操作")
	}
}
