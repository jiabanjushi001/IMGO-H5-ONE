package server

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
)

// The ticker only checks whether a persisted job is due. Stopping this job never
// stops HTTP or WebSocket, and settings take priority over the legacy env flag.
func (a *App) startMaintenance() {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				job, stop := context.WithTimeout(ctx, 30*time.Second)
				if e := a.maintenanceTick(job, time.Now()); e != nil {
					a.log.Error("maintenance", "error", e)
				}
				stop()
			}
		}
	}()
}
func (a *App) loadMaintenance(ctx context.Context) (M, error) {
	c := M{"enabled": os.Getenv("MAINTENANCE_ENABLED") == "true", "interval_minutes": 60, "next_run_at": 0, "last_run_at": 0, "last_error": "", "logs": []any{}}
	row, e := one(ctx, a.db, "SELECT value FROM "+a.t("config")+" WHERE name='maintenance' AND status=1 LIMIT 1")
	if e == nil {
		for k, v := range obj(row["value"]) {
			c[k] = v
		}
	} else if e != sql.ErrNoRows {
		return nil, e
	}
	chat := defaultConfig("chatInfo")
	row, e = one(ctx, a.db, "SELECT value FROM "+a.t("config")+" WHERE name='chatInfo' AND status=1 LIMIT 1")
	if e == nil {
		chat = obj(row["value"])
	} else if e != sql.ErrNoRows {
		return nil, e
	}
	c["clear_messages"] = number(chat["msgClear"]) == 1
	days := number(chat["msgClearDay"])
	if days < 1 || days > 3650 {
		days = 30
	}
	c["retention_days"] = days
	if interval := number(c["interval_minutes"]); interval < 1 || interval > 43200 {
		c["interval_minutes"] = 60
	}
	return c, nil
}
func (a *App) saveMaintenance(ctx context.Context, db DB, c M) error {
	row, e := one(ctx, db, "SELECT id FROM "+a.t("config")+" WHERE name='maintenance' LIMIT 1")
	if e == sql.ErrNoRows {
		_, e = insert(ctx, db, a.t("config"), M{"name": "maintenance", "value": js(c), "status": 1, "create_time": time.Now().Unix()})
		return e
	}
	if e != nil {
		return e
	}
	return update(ctx, db, a.t("config"), M{"value": js(c), "status": 1, "update_time": time.Now().Unix()}, "id=?", row["id"])
}
func (a *App) configureMaintenance(r *request, enabled *bool) (any, error) {
	a.maintenanceMu.Lock()
	defer a.maintenanceMu.Unlock()
	c, e := a.loadMaintenance(r.ctx())
	if e != nil {
		return nil, e
	}
	for key, maximum := range map[string]int64{"interval_minutes": 43200, "retention_days": 3650} {
		if value, ok := r.p[key]; ok {
			n := number(value)
			if n < 1 || n > maximum || str(value) != fmt.Sprint(n) {
				return nil, r.fail(fmt.Sprintf("%s 必须为 1 至 %d 的整数", key, maximum))
			}
			c[key] = n
		}
	}
	if v, ok := r.p["clear_messages"]; ok {
		if str(v) != "true" && str(v) != "false" && str(v) != "0" && str(v) != "1" {
			return nil, r.fail("消息清理开关无效")
		}
		c["clear_messages"] = number(v) == 1
	}
	if enabled != nil {
		c["enabled"] = *enabled
	}
	c["next_run_at"] = 0
	if number(c["enabled"]) == 1 {
		c["next_run_at"] = time.Now().Add(time.Duration(number(c["interval_minutes"])) * time.Minute).Unix()
	}
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	row, e := one(r.ctx(), tx, "SELECT id,value FROM "+a.t("config")+" WHERE name='chatInfo' LIMIT 1 FOR UPDATE")
	if e != nil {
		return nil, e
	}
	chat := obj(row["value"])
	chat["msgClear"] = number(c["clear_messages"])
	chat["msgClearDay"] = number(c["retention_days"])
	if e = update(r.ctx(), tx, a.t("config"), M{"value": js(chat)}, "id=?", row["id"]); e != nil {
		return nil, e
	}
	if e = a.saveMaintenance(r.ctx(), tx, c); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return maintenanceView(c), nil
}
func maintenanceView(c M) M {
	v := M{}
	for k, x := range c {
		if k != "logs" {
			v[k] = x
		}
	}
	return v
}
func (a *App) maintenanceTick(ctx context.Context, now time.Time) error {
	a.maintenanceMu.Lock()
	defer a.maintenanceMu.Unlock()
	c, e := a.loadMaintenance(ctx)
	if e != nil {
		return e
	}
	if number(c["enabled"]) != 1 {
		return nil
	}
	if number(c["next_run_at"]) == 0 {
		c["next_run_at"] = now.Add(time.Duration(number(c["interval_minutes"])) * time.Minute).Unix()
		return a.saveMaintenance(ctx, a.db, c)
	}
	if now.Unix() < number(c["next_run_at"]) {
		return nil
	}
	c["next_run_at"] = now.Add(time.Duration(number(c["interval_minutes"])) * time.Minute).Unix()
	e = a.runMaintenance(ctx, c, now)
	if e != nil {
		c["last_error"] = "清理失败，请检查服务日志"
		c["last_run_at"] = now.Unix()
		appendMaintenanceLog(c, now.Format(time.RFC3339)+" 清理失败")
		// Keep retries at the configured interval even when the job transaction fails.
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if saveErr := a.saveMaintenance(saveCtx, a.db, c); saveErr != nil {
			return fmt.Errorf("maintenance: %v; save failure state: %w", e, saveErr)
		}
	}
	return e
}
func appendMaintenanceLog(c M, entry string) {
	logs, _ := c["logs"].([]any)
	logs = append(logs, entry)
	if len(logs) > 20 {
		logs = logs[len(logs)-20:]
	}
	c["logs"] = logs
}
func (a *App) runMaintenance(ctx context.Context, c M, now time.Time) error {
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, "DELETE FROM "+a.t("imgo_session")+" WHERE expires_at<?", now.Unix())
	if e != nil {
		return e
	}
	sessions, _ := res.RowsAffected()
	messages := int64(0)
	if number(c["clear_messages"]) == 1 {
		cutoff := now.Add(-time.Duration(number(c["retention_days"])) * 24 * time.Hour).Unix()
		res, e = tx.ExecContext(ctx, "UPDATE "+a.t("message")+" SET status=0 WHERE status=1 AND chat_identify<>'admin_notice' AND create_time<?", cutoff)
		if e != nil {
			return e
		}
		messages, _ = res.RowsAffected()
	}
	c["last_run_at"] = now.Unix()
	c["last_error"] = ""
	c["last_sessions"] = sessions
	c["last_messages"] = messages
	appendMaintenanceLog(c, fmt.Sprintf("%s 清理 %d 条过期会话，隐藏 %d 条过期消息", now.Format(time.RFC3339), sessions, messages))
	if e = a.saveMaintenance(ctx, tx, c); e != nil {
		return e
	}
	return tx.Commit()
}
