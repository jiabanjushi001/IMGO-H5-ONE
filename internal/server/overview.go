package server

import (
	"context"
	"time"
)

var dashboardZone = time.FixedZone("Asia/Shanghai", 8*60*60)

func (a *App) recordOnline(ctx context.Context, now time.Time) error {
	// Resolve team membership before touching the Hub, then release its lock
	// before any sample write. A LEFT JOIN keeps agents with no descendants.
	teams, err := rows(ctx, a.db, "SELECT agent.user_id AS agent_user_id,child.user_id AS descendant_user_id FROM "+a.t("user")+" agent JOIN "+a.t("imgo_admin_role")+" ar ON ar.role_id=agent.admin_role_id LEFT JOIN "+a.t("imgo_referral_path")+" rp ON rp.ancestor_user_id=agent.user_id AND rp.depth>0 LEFT JOIN "+a.t("user")+" child ON child.user_id=rp.descendant_user_id AND child.delete_time=0 WHERE agent.status=1 AND agent.delete_time=0 AND ar.status=1 AND ar.agent_mode=1 ORDER BY agent.user_id,rp.descendant_user_id")
	if err != nil {
		return err
	}
	teamUsers := map[int64]map[int64]bool{}
	agentIDs := []int64{}
	for _, row := range teams {
		agentID := number(row["agent_user_id"])
		if agentID < 1 {
			continue
		}
		if _, ok := teamUsers[agentID]; !ok {
			teamUsers[agentID] = map[int64]bool{}
			agentIDs = append(agentIDs, agentID)
		}
		if childID := number(row["descendant_user_id"]); childID > 0 && childID != agentID {
			teamUsers[agentID][childID] = true
		}
	}
	users, devices := a.hub.onlineCounts(now)
	sampleAt := now.Unix() / 60 * 60
	if _, err := a.db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_online_sample")+" (sample_at,users,devices) VALUES (?,?,?) ON DUPLICATE KEY UPDATE users=GREATEST(users,VALUES(users)),devices=GREATEST(devices,VALUES(devices))", sampleAt, users, devices); err != nil {
		return err
	}
	for _, agentID := range agentIDs {
		users, devices := a.hub.onlineCountsForUsers(teamUsers[agentID], now)
		if _, err := a.db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_agent_online_sample")+" (agent_user_id,sample_at,users,devices) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE users=GREATEST(users,VALUES(users)),devices=GREATEST(devices,VALUES(devices))", agentID, sampleAt, users, devices); err != nil {
			return err
		}
	}
	return nil
}

// Called only by the HTTP server, after schema validation, not by migration tools.
func (a *App) StartOverviewMetrics() {
	ctx, cancel := context.WithCancel(context.Background())
	a.metricsCancel = cancel
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		for {
			job, stop := context.WithTimeout(ctx, 5*time.Second)
			if err := a.recordOnline(job, time.Now()); err != nil && ctx.Err() == nil {
				a.log.Error("online metrics", "error", err)
			}
			stop()
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

func periodStarts(now time.Time, months bool, count int) []time.Time {
	now = now.In(dashboardZone)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, dashboardZone)
	if months {
		day = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, dashboardZone)
	}
	out := make([]time.Time, count+1)
	for i := range out {
		if months {
			out[i] = day.AddDate(0, i-count+1, 0)
		} else {
			out[i] = day.AddDate(0, 0, i-count+1)
		}
	}
	return out
}

func (a *App) trend(ctx context.Context, table, alias, predicate string, predicateArgs []any, boundaries []time.Time, monthly bool) ([]M, error) {
	format := "2006-01-02"
	sqlFormat := "%Y-%m-%d"
	if monthly {
		format = "2006-01"
		sqlFormat = "%Y-%m"
	}
	// DATE_ADD from a literal epoch avoids dependency on the MySQL session timezone.
	q := "SELECT DATE_FORMAT(DATE_ADD('1970-01-01', INTERVAL (" + alias + ".create_time+28800) SECOND), ?) AS bucket,COUNT(*) AS n FROM " + a.t(table) + " " + alias + " WHERE " + predicate + " AND " + alias + ".create_time>=? AND " + alias + ".create_time<? GROUP BY bucket"
	args := append([]any{sqlFormat}, predicateArgs...)
	args = append(args, boundaries[0].Unix(), boundaries[len(boundaries)-1].Unix())
	data, err := rows(ctx, a.db, q, args...)
	if err != nil {
		return nil, err
	}
	values := map[string]int64{}
	for _, r := range data {
		values[str(r["bucket"])] = number(r["n"])
	}
	out := make([]M, 0, len(boundaries)-1)
	for _, t := range boundaries[:len(boundaries)-1] {
		label := t.Format(format)
		out = append(out, M{"label": label, "value": values[label]})
	}
	return out, nil
}

func (a *App) overview(r *request) (any, error) {
	now := time.Now()
	today := periodStarts(now, false, 1)[0].Unix()
	scope, err := a.adminScope(r.ctx(), r.user)
	if err != nil {
		return nil, err
	}
	result := M{"timezone": "Asia/Shanghai", "generated_at": now.Unix()}
	totals := M{}
	daily := M{}
	userScope, userArgs := scope.userPredicate("u")
	groupScope, groupArgs := a.groupScopePredicate(scope, "g")
	messageScope, messageArgs := a.messageScopePredicate(scope, "m")
	fileScope, fileArgs := scope.userPredicate("f")
	type overviewSpec struct {
		name, table, alias, where string
		args                      []any
	}
	specs := []overviewSpec{
		{"users", "user", "u", "u.delete_time=0 AND " + userScope, userArgs},
		{"groups", "group", "g", "g.status=1 AND g.delete_time=0 AND " + groupScope, groupArgs},
		{"messages", "message", "m", "m.status=1 AND m.chat_identify<>'admin_notice' AND " + messageScope, messageArgs},
		{"files", "file", "f", "f.status=1 AND f.delete_time=0 AND " + fileScope, fileArgs},
	}
	for _, spec := range specs {
		var total, added int64
		args := append([]any{today}, spec.args...)
		err := a.db.QueryRowContext(r.ctx(), "SELECT COUNT(*),COALESCE(SUM("+spec.alias+".create_time>=?),0) FROM "+a.t(spec.table)+" "+spec.alias+" WHERE "+spec.where, args...).Scan(&total, &added)
		if err != nil {
			return nil, err
		}
		totals[spec.name] = total
		daily[spec.name] = added
	}
	users, devices := 0, 0
	if scope.Global {
		users, devices = a.hub.onlineCounts(now)
	} else {
		allowed := map[int64]bool{}
		members, err := rows(r.ctx(), a.db, "SELECT path.descendant_user_id AS user_id FROM "+a.t("imgo_referral_path")+" path JOIN "+a.t("user")+" child ON child.user_id=path.descendant_user_id WHERE path.ancestor_user_id=? AND path.depth>0 AND child.delete_time=0", scope.AgentUserID)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			allowed[number(member["user_id"])] = true
		}
		users, devices = a.hub.onlineCountsForUsers(allowed, now)
	}
	totals["online_users"] = users
	totals["online_devices"] = devices
	var pu, pd int64
	peakTable := a.t("imgo_online_sample")
	peakWhere := "sample_at>=?"
	peakArgs := []any{today}
	if !scope.Global {
		peakTable = a.t("imgo_agent_online_sample")
		peakWhere = "agent_user_id=? AND sample_at>=?"
		peakArgs = []any{scope.AgentUserID, today}
	}
	if err := a.db.QueryRowContext(r.ctx(), "SELECT COALESCE(MAX(users),0),COALESCE(MAX(devices),0) FROM "+peakTable+" WHERE "+peakWhere, peakArgs...).Scan(&pu, &pd); err != nil {
		return nil, err
	}
	if int64(users) > pu {
		pu = int64(users)
	}
	if int64(devices) > pd {
		pd = int64(devices)
	}
	daily["peak_users"] = pu
	daily["peak_devices"] = pd
	result["totals"] = totals
	result["today"] = daily
	for _, spec := range []struct {
		key     string
		spec    overviewSpec
		monthly bool
		count   int
	}{{"registration_month", specs[0], true, 12}, {"registration_day", specs[0], false, 30}, {"messages", specs[2], false, 30}, {"groups", specs[1], true, 12}, {"files", specs[3], true, 12}} {
		data, err := a.trend(r.ctx(), spec.spec.table, spec.spec.alias, spec.spec.where, spec.spec.args, periodStarts(now, spec.monthly, spec.count), spec.monthly)
		if err != nil {
			return nil, err
		}
		result[spec.key] = data
	}
	days := r.n("online_days")
	if days == 0 {
		days = 1
	}
	if days != 1 && days != 7 && days != 30 {
		return nil, r.fail("在线趋势仅支持今日、近7天或近30天")
	}
	since := periodStarts(now, false, int(days))[0].Unix()
	bucket := int64(300)
	if days > 1 {
		bucket = 3600
	}
	sampleTable := a.t("imgo_online_sample")
	sampleWhere := "sample_at>=? AND sample_at<=?"
	sampleArgs := []any{bucket, bucket, since, now.Unix()}
	if !scope.Global {
		sampleTable = a.t("imgo_agent_online_sample")
		sampleWhere = "agent_user_id=? AND " + sampleWhere
		sampleArgs = []any{bucket, bucket, scope.AgentUserID, since, now.Unix()}
	}
	samples, err := rows(r.ctx(), a.db, "SELECT FLOOR(sample_at/?)*? AS time,MAX(users) AS users,MAX(devices) AS devices FROM "+sampleTable+" WHERE "+sampleWhere+" GROUP BY `time` ORDER BY `time`", sampleArgs...)
	if err != nil {
		return nil, err
	}
	result["online"] = samples
	result["online_bucket_seconds"] = bucket
	result["online_since"] = since
	return result, nil
}
