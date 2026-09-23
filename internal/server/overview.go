package server

import (
	"context"
	"time"
)

var dashboardZone = time.FixedZone("Asia/Shanghai", 8*60*60)

// Online means authenticated, live WebSocket connections; multiple connections
// from one user count as multiple devices, but only one online user.
func (h *Hub) onlineCounts(now time.Time) (int, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := map[int64]bool{}
	devices := 0
	for _, p := range h.peers {
		if p.uid == 0 || p.claims.Exp <= now.Unix() {
			continue
		}
		select {
		case <-p.done:
			continue
		default:
		}
		users[p.uid] = true
		devices++
	}
	return len(users), devices
}

func (a *App) recordOnline(ctx context.Context, now time.Time) error {
	users, devices := a.hub.onlineCounts(now)
	_, err := a.db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_online_sample")+" (sample_at,users,devices) VALUES (?,?,?) ON DUPLICATE KEY UPDATE users=GREATEST(users,VALUES(users)),devices=GREATEST(devices,VALUES(devices))", now.Unix()/60*60, users, devices)
	return err
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

func (a *App) trend(ctx context.Context, table, predicate string, boundaries []time.Time, monthly bool) ([]M, error) {
	format := "2006-01-02"
	sqlFormat := "%Y-%m-%d"
	if monthly {
		format = "2006-01"
		sqlFormat = "%Y-%m"
	}
	// DATE_ADD from a literal epoch avoids dependency on the MySQL session timezone.
	q := "SELECT DATE_FORMAT(DATE_ADD('1970-01-01', INTERVAL (create_time+28800) SECOND), ?) AS bucket,COUNT(*) AS n FROM " + a.t(table) + " WHERE " + predicate + " AND create_time>=? AND create_time<? GROUP BY bucket"
	data, err := rows(ctx, a.db, q, sqlFormat, boundaries[0].Unix(), boundaries[len(boundaries)-1].Unix())
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
	result := M{"timezone": "Asia/Shanghai", "generated_at": now.Unix()}
	totals := M{}
	daily := M{}
	for _, spec := range []struct{ name, table, where string }{{"users", "user", "delete_time=0"}, {"groups", "group", "status=1 AND delete_time=0"}, {"messages", "message", "status=1 AND chat_identify<>'admin_notice'"}, {"files", "file", "status=1 AND delete_time=0"}} {
		var total, added int64
		err := a.db.QueryRowContext(r.ctx(), "SELECT COUNT(*),COALESCE(SUM(create_time>=?),0) FROM "+a.t(spec.table)+" WHERE "+spec.where, today).Scan(&total, &added)
		if err != nil {
			return nil, err
		}
		totals[spec.name] = total
		daily[spec.name] = added
	}
	users, devices := a.hub.onlineCounts(now)
	totals["online_users"] = users
	totals["online_devices"] = devices
	var pu, pd int64
	if err := a.db.QueryRowContext(r.ctx(), "SELECT COALESCE(MAX(users),0),COALESCE(MAX(devices),0) FROM "+a.t("imgo_online_sample")+" WHERE sample_at>=?", today).Scan(&pu, &pd); err != nil {
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
		key, table, where string
		monthly           bool
		count             int
	}{{"registration_month", "user", "delete_time=0", true, 12}, {"registration_day", "user", "delete_time=0", false, 30}, {"messages", "message", "status=1 AND chat_identify<>'admin_notice'", false, 30}, {"groups", "group", "status=1 AND delete_time=0", true, 12}, {"files", "file", "status=1 AND delete_time=0", true, 12}} {
		data, err := a.trend(r.ctx(), spec.table, spec.where, periodStarts(now, spec.monthly, spec.count), spec.monthly)
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
	samples, err := rows(r.ctx(), a.db, "SELECT FLOOR(sample_at/?)*? AS time,MAX(users) AS users,MAX(devices) AS devices FROM "+a.t("imgo_online_sample")+" WHERE sample_at>=? AND sample_at<=? GROUP BY `time` ORDER BY `time`", bucket, bucket, since, now.Unix())
	if err != nil {
		return nil, err
	}
	result["online"] = samples
	result["online_bucket_seconds"] = bucket
	result["online_since"] = since
	return result, nil
}
