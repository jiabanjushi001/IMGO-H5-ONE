package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var checkInZone = time.FixedZone("China Standard Time", 8*60*60)

func checkInDate(now time.Time) string {
	return now.In(checkInZone).Format("2006-01-02")
}

type checkInRewardSettings struct {
	Enabled                      bool
	CycleDays, HighDays          int64
	HighMinYuan, HighMaxYuan     int64
	NormalMinYuan, NormalMaxYuan int64
}

func checkInSettingInt(value any, name string, min, max int64) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(str(value)), 10, 64)
	if err != nil || parsed < min || parsed > max {
		return 0, fmt.Errorf("%s必须是%d到%d的整数", name, min, max)
	}
	return parsed, nil
}

func decodeCheckInSettings(value M) (checkInRewardSettings, error) {
	enabledText := strings.ToLower(strings.TrimSpace(str(value["rewardEnabled"])))
	settings := checkInRewardSettings{}
	switch enabledText {
	case "1", "true":
		settings.Enabled = true
	case "0", "false":
		settings.Enabled = false
	default:
		return settings, errors.New("签到奖励开关无效")
	}
	var err error
	if settings.CycleDays, err = checkInSettingInt(value["cycleDays"], "奖励周期", 2, 31); err != nil {
		return settings, err
	}
	if settings.HighDays, err = checkInSettingInt(value["highDays"], "高奖励天数", 1, settings.CycleDays); err != nil {
		return settings, err
	}
	if settings.HighMinYuan, err = checkInSettingInt(value["highMinYuan"], "前段最低奖励", 1, 10000); err != nil {
		return settings, err
	}
	if settings.HighMaxYuan, err = checkInSettingInt(value["highMaxYuan"], "前段最高奖励", 1, 10000); err != nil {
		return settings, err
	}
	if settings.NormalMinYuan, err = checkInSettingInt(value["normalMinYuan"], "后段最低奖励", 1, 10000); err != nil {
		return settings, err
	}
	if settings.NormalMaxYuan, err = checkInSettingInt(value["normalMaxYuan"], "后段最高奖励", 1, 10000); err != nil {
		return settings, err
	}
	if settings.HighMinYuan > settings.HighMaxYuan || settings.NormalMinYuan > settings.NormalMaxYuan {
		return settings, errors.New("签到奖励最低金额不能大于最高金额")
	}
	return settings, nil
}

func checkInSettingsValue(settings checkInRewardSettings) M {
	enabled := 0
	if settings.Enabled {
		enabled = 1
	}
	return M{
		"rewardEnabled": enabled,
		"cycleDays":     settings.CycleDays,
		"highDays":      settings.HighDays,
		"highMinYuan":   settings.HighMinYuan,
		"highMaxYuan":   settings.HighMaxYuan,
		"normalMinYuan": settings.NormalMinYuan,
		"normalMaxYuan": settings.NormalMaxYuan,
	}
}

func normalizeCheckInConfig(value M) (M, error) {
	settings, err := decodeCheckInSettings(value)
	if err != nil {
		return nil, clientError{err.Error(), 400}
	}
	return checkInSettingsValue(settings), nil
}

func (a *App) checkInSettings(ctx context.Context) checkInRewardSettings {
	settings, err := decodeCheckInSettings(a.config(ctx, "checkIn"))
	if err == nil {
		return settings
	}
	settings, _ = decodeCheckInSettings(defaultConfig("checkIn"))
	return settings
}

// checkInSettingsForUser resolves the nearest active mentor first. A mentor's
// explicit rules apply only to that mentor and their closest descendant team;
// users without a mentor (or mentors that have not saved rules yet) use the
// global setting.
func (a *App) checkInSettingsForUser(ctx context.Context, db DB, userID int64) (checkInRewardSettings, error) {
	global := a.checkInSettings(ctx)
	if userID < 1 {
		return global, nil
	}
	agentID := int64(0)
	direct, err := a.mentorEnabled(ctx, db, userID)
	if err != nil {
		return global, err
	}
	if direct {
		agentID = userID
	} else if agentID, err = a.nearestAgent(ctx, db, userID); err != nil {
		return global, err
	}
	if agentID < 1 {
		return global, nil
	}
	row, err := one(ctx, db, "SELECT check_in_setting FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && row["check_in_setting"] == nil {
		return global, nil
	}
	if err != nil {
		return global, err
	}
	settings, err := decodeCheckInSettings(obj(row["check_in_setting"]))
	if err != nil {
		return global, nil
	}
	return settings, nil
}

func (a *App) manageCheckInSettings(r *request) (any, error) {
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	act := action(r)
	if act != "getcheckinsettings" && act != "setcheckinsettings" {
		return nil, r.fail("未知操作")
	}
	if scope.Global {
		if r.uid() != 1 {
			if err = a.authorizeManage(r.ctx(), r.user, "manage.settings"); err != nil {
				return nil, err
			}
		}
		value := a.config(r.ctx(), "checkIn")
		if act == "setcheckinsettings" {
			value, err = normalizeCheckInConfig(r.p)
			if err != nil {
				return nil, err
			}
			existing, findErr := r.one("SELECT id FROM "+a.t("config")+" WHERE name=?", "checkIn")
			if errors.Is(findErr, sql.ErrNoRows) {
				_, err = insert(r.ctx(), a.db, a.t("config"), M{"name": "checkIn", "value": js(value), "create_user": r.uid(), "create_time": time.Now().Unix(), "status": 1})
			} else if findErr == nil {
				err = update(r.ctx(), a.db, a.t("config"), M{"value": js(value), "update_time": time.Now().Unix()}, "id=?", existing["id"])
			} else {
				err = findErr
			}
			if err != nil {
				return nil, err
			}
		}
		result := M{}
		for key, item := range value {
			result[key] = item
		}
		result["setting_scope"] = "global"
		result["inherited"] = false
		return result, nil
	}

	agentID := scope.AgentUserID
	if agentID < 1 {
		return nil, deny()
	}
	if act == "setcheckinsettings" {
		value, normalizeErr := normalizeCheckInConfig(r.p)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		_, err = a.db.ExecContext(r.ctx(), "INSERT INTO "+a.t("imgo_agent_setting")+" (agent_user_id,check_in_setting,updated_by,updated_at) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE check_in_setting=VALUES(check_in_setting),updated_by=VALUES(updated_by),updated_at=VALUES(updated_at)", agentID, js(value), r.uid(), time.Now().Unix())
		if err != nil {
			return nil, err
		}
		value["setting_scope"] = "mentor"
		value["inherited"] = false
		return value, nil
	}

	row, findErr := r.one("SELECT check_in_setting FROM "+a.t("imgo_agent_setting")+" WHERE agent_user_id=?", agentID)
	inherited := errors.Is(findErr, sql.ErrNoRows) || findErr == nil && row["check_in_setting"] == nil
	if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
		return nil, findErr
	}
	value := a.config(r.ctx(), "checkIn")
	if !inherited {
		if normalized, normalizeErr := normalizeCheckInConfig(obj(row["check_in_setting"])); normalizeErr == nil {
			value = normalized
		} else {
			inherited = true
		}
	}
	result := M{}
	for key, item := range value {
		result[key] = item
	}
	result["setting_scope"] = "mentor"
	result["inherited"] = inherited
	return result, nil
}

func checkInRewardBounds(settings checkInRewardSettings, cycleDay int64) (int64, int64) {
	if cycleDay <= settings.HighDays {
		return settings.HighMinYuan * 100, settings.HighMaxYuan * 100
	}
	return settings.NormalMinYuan * 100, settings.NormalMaxYuan * 100
}

func randomCheckInReward(settings checkInRewardSettings, cycleDay int64) (int64, error) {
	if !settings.Enabled {
		return 0, nil
	}
	minimum, maximum := checkInRewardBounds(settings, cycleDay)
	span := maximum/100 - minimum/100 + 1
	random, err := rand.Int(rand.Reader, big.NewInt(span))
	if err != nil {
		return 0, err
	}
	return minimum + random.Int64()*100, nil
}

func checkInResult(settings checkInRewardSettings, day string, total int64, signed bool, todayReward, totalReward, cycleDay int64, alreadySigned bool) M {
	if cycleDay < 1 {
		if signed && total > 0 {
			cycleDay = (total-1)%settings.CycleDays + 1
		} else {
			cycleDay = total%settings.CycleDays + 1
		}
	}
	return M{
		"date": day, "signed_today": signed, "total_days": total,
		"already_signed": alreadySigned, "reward_enabled": settings.Enabled,
		"today_reward_cents": todayReward, "total_reward_cents": totalReward,
		"cycle_day": cycleDay, "cycle_days": settings.CycleDays,
	}
}

func (a *App) checkInStatus(r *request, day string, settings checkInRewardSettings) (M, error) {
	row, err := r.one("SELECT COUNT(*) total,COALESCE(MAX(CASE WHEN sign_date=? THEN 1 ELSE 0 END),0) signed_today,COALESCE(MAX(CASE WHEN sign_date=? THEN reward_cents ELSE 0 END),0) today_reward_cents,COALESCE(MAX(CASE WHEN sign_date=? THEN cycle_day ELSE 0 END),0) cycle_day,COALESCE(SUM(reward_cents),0) total_reward_cents FROM "+a.t("imgo_check_in")+" WHERE user_id=?", day, day, day, r.uid())
	if err != nil {
		return nil, err
	}
	return checkInResult(settings, day, number(row["total"]), number(row["signed_today"]) > 0, number(row["today_reward_cents"]), number(row["total_reward_cents"]), number(row["cycle_day"]), false), nil
}

// Attach summaries for just the visible member page, never one query per member.
func (a *App) memberCheckInSummaries(r *request, users []M) error {
	if len(users) == 0 {
		return nil
	}
	byID := make(map[int64]M, len(users))
	args := make([]any, 0, len(users)+2)
	today := checkInDate(time.Now())
	args = append(args, today, today)
	for _, u := range users {
		u["checkin_days"] = int64(0)
		u["checkin_today"] = false
		u["checkin_last_date"] = ""
		u["checkin_today_reward_cents"] = int64(0)
		u["checkin_total_reward_cents"] = int64(0)
		id := number(u["user_id"])
		byID[id] = u
		args = append(args, id)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(users)), ",")
	query := "SELECT user_id,COUNT(*) total_days,DATE_FORMAT(MAX(sign_date),'%Y-%m-%d') last_date,COALESCE(MAX(CASE WHEN sign_date=? THEN 1 ELSE 0 END),0) signed_today,COALESCE(MAX(CASE WHEN sign_date=? THEN reward_cents ELSE 0 END),0) today_reward_cents,COALESCE(SUM(reward_cents),0) total_reward_cents FROM " + a.t("imgo_check_in") + " WHERE user_id IN (" + placeholders + ") GROUP BY user_id"
	summaries, err := r.list(query, args...)
	if err != nil {
		return err
	}
	for _, summary := range summaries {
		if u := byID[number(summary["user_id"])]; u != nil {
			u["checkin_days"] = number(summary["total_days"])
			u["checkin_today"] = number(summary["signed_today"]) > 0
			u["checkin_last_date"] = str(summary["last_date"])
			u["checkin_today_reward_cents"] = number(summary["today_reward_cents"])
			u["checkin_total_reward_cents"] = number(summary["total_reward_cents"])
		}
	}
	return nil
}

func (a *App) checkIn(r *request) (any, error) {
	r.c.Header("Cache-Control", "no-store")
	now := time.Now()
	day := checkInDate(now)
	settings, err := a.checkInSettingsForUser(r.ctx(), a.db, r.uid())
	if err != nil {
		return nil, err
	}
	switch action(r) {
	case "status":
		return a.checkInStatus(r, day, settings)
	case "submit":
		ctx := r.ctx()
		tx, err := a.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO "+a.t("imgo_wallet")+" (user_id,available_cents,pending_cents,updated_at) VALUES (?,0,0,?)", r.uid(), now.Unix()); err != nil {
			return nil, err
		}
		if _, err = one(ctx, tx, "SELECT available_cents FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", r.uid()); err != nil {
			return nil, err
		}
		existing, err := one(ctx, tx, "SELECT reward_cents,cycle_day FROM "+a.t("imgo_check_in")+" WHERE user_id=? AND sign_date=? FOR UPDATE", r.uid(), day)
		if err == nil {
			count, countErr := one(ctx, tx, "SELECT COUNT(*) total,COALESCE(SUM(reward_cents),0) total_reward_cents FROM "+a.t("imgo_check_in")+" WHERE user_id=?", r.uid())
			if countErr != nil {
				return nil, countErr
			}
			if err = tx.Commit(); err != nil {
				return nil, err
			}
			return checkInResult(settings, day, number(count["total"]), true, number(existing["reward_cents"]), number(count["total_reward_cents"]), number(existing["cycle_day"]), true), nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		count, err := one(ctx, tx, "SELECT COUNT(*) total,COALESCE(SUM(reward_cents),0) total_reward_cents FROM "+a.t("imgo_check_in")+" WHERE user_id=?", r.uid())
		if err != nil {
			return nil, err
		}
		total := number(count["total"])
		cycleDay := total%settings.CycleDays + 1
		reward, err := randomCheckInReward(settings, cycleDay)
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_check_in")+" (user_id,sign_date,reward_cents,cycle_day,created_at) VALUES (?,?,?,?,?)", r.uid(), day, reward, cycleDay, now.Unix()); err != nil {
			return nil, err
		}
		if reward > 0 {
			if _, err = tx.ExecContext(ctx, "UPDATE "+a.t("imgo_wallet")+" SET available_cents=available_cents+?,updated_at=? WHERE user_id=?", reward, now.Unix(), r.uid()); err != nil {
				return nil, err
			}
			requestID := "checkin-" + strings.ReplaceAll(day, "-", "")
			note := fmt.Sprintf("第%d天签到奖励（%d天周期）", cycleDay, settings.CycleDays)
			if _, err = tx.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet_entry")+" (user_id,event,request_id,available_delta,pending_delta,reference_id,actor_id,note,created_at) VALUES (?,'checkin',?,?,0,0,?,?,?)", r.uid(), requestID, reward, r.uid(), note, now.Unix()); err != nil {
				return nil, err
			}
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return checkInResult(settings, day, total+1, true, reward, number(count["total_reward_cents"])+reward, cycleDay, false), nil
	default:
		return nil, r.fail("未知操作")
	}
}
