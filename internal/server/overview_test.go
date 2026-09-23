package server

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestOverviewCalendarBoundaries(t *testing.T) {
	now := time.Date(2026, 1, 31, 16, 30, 0, 0, time.UTC) // February in Beijing
	days := periodStarts(now, false, 30)
	if len(days) != 31 || days[29].Format("2006-01-02") != "2026-02-01" || days[30].Format("2006-01-02") != "2026-02-02" {
		t.Fatal(days)
	}
	months := periodStarts(now, true, 12)
	if months[0].Format("2006-01-02") != "2025-03-01" || months[11].Format("2006-01-02") != "2026-02-01" {
		t.Fatal(months)
	}
}
func TestOverviewOnlineCounts(t *testing.T) {
	now := time.Now()
	h := newHub(nil)
	for id, uid := range map[string]int64{"a": 1, "b": 1, "c": 2, "anonymous": 0} {
		h.peers[id] = &peer{uid: uid, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	}
	h.peers["expired"] = &peer{uid: 3, claims: claims{Exp: now.Add(-time.Second).Unix()}, done: make(chan struct{})}
	h.peers["closed"] = &peer{uid: 4, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	close(h.peers["closed"].done)
	u, d := h.onlineCounts(now)
	if u != 2 || d != 3 {
		t.Fatalf("got %d users %d devices", u, d)
	}
}

func TestOnlineCountsForUsersExcludesOutsidersAndInactiveConnections(t *testing.T) {
	now := time.Now()
	h := newHub(nil)
	for id, uid := range map[string]int64{"one": 12, "two": 12, "nested": 13, "outsider": 99, "agent": 7, "anonymous": 0} {
		h.peers[id] = &peer{uid: uid, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	}
	h.peers["expired"] = &peer{uid: 14, claims: claims{Exp: now.Add(-time.Second).Unix()}, done: make(chan struct{})}
	h.peers["closed"] = &peer{uid: 15, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	close(h.peers["closed"].done)
	users, devices := h.onlineCountsForUsers(map[int64]bool{12: true, 13: true, 14: true, 15: true}, now)
	if users != 2 || devices != 3 {
		t.Fatalf("scoped online = %d users, %d devices", users, devices)
	}
	users, devices = h.onlineCountsForUsers(nil, now)
	if users != 0 || devices != 0 {
		t.Fatalf("nil scope exposed %d users, %d devices", users, devices)
	}
}

func TestAgentOverviewScopesCardsTrendsAndOnlineSamples(t *testing.T) {
	a, mock := testApp(t)
	t.Cleanup(func() { a.hub.peers = map[string]*peer{} })
	expectAgentMemberScope(mock)
	for _, tc := range []struct {
		table, scope string
		args         []driver.Value
	}{
		{"user", "scope_path.descendant_user_id=u.user_id", []driver.Value{int64(7), int64(7)}},
		{"group", "scope_path.descendant_user_id=scope_owner.user_id", []driver.Value{int64(7), int64(7)}},
		{"message", "scope_path.descendant_user_id=scope_from.user_id", []driver.Value{int64(7), int64(7), int64(7), int64(7), int64(7), int64(7)}},
		{"file", "scope_path.descendant_user_id=f.user_id", []driver.Value{int64(7), int64(7)}},
	} {
		args := append([]driver.Value{sqlmock.AnyArg()}, tc.args...)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM `yu_" + tc.table + "` .*" + regexp.QuoteMeta(tc.scope)).WithArgs(args...).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)", "COALESCE(SUM(create_time>=?),0)"}).AddRow(2, 1))
	}
	mock.ExpectQuery("SELECT .*FROM `yu_imgo_referral_path`.*path.descendant_user_id<>\\?").WithArgs(int64(7), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7).AddRow(12).AddRow(13))
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(users\\),0\\).*FROM `yu_imgo_agent_online_sample`.*agent_user_id=\\?").
		WithArgs(int64(7), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"peak_users", "peak_devices"}).AddRow(1, 2))
	for _, tc := range []struct {
		table, scope string
		args         []driver.Value
	}{
		{"user", "scope_path.descendant_user_id=u.user_id", []driver.Value{int64(7), int64(7)}},
		{"user", "scope_path.descendant_user_id=u.user_id", []driver.Value{int64(7), int64(7)}},
		{"message", "scope_path.descendant_user_id=scope_from.user_id", []driver.Value{int64(7), int64(7), int64(7), int64(7), int64(7), int64(7)}},
		{"group", "scope_path.descendant_user_id=scope_owner.user_id", []driver.Value{int64(7), int64(7)}},
		{"file", "scope_path.descendant_user_id=f.user_id", []driver.Value{int64(7), int64(7)}},
	} {
		args := append([]driver.Value{sqlmock.AnyArg()}, tc.args...)
		args = append(args, sqlmock.AnyArg(), sqlmock.AnyArg())
		mock.ExpectQuery("SELECT DATE_FORMAT.*FROM `yu_" + tc.table + "` .*" + regexp.QuoteMeta(tc.scope)).WithArgs(args...).
			WillReturnRows(sqlmock.NewRows([]string{"bucket", "n"}))
	}
	mock.ExpectQuery("SELECT FLOOR.*FROM `yu_imgo_agent_online_sample`.*agent_user_id=\\?").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(7), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"time", "users", "devices"}))
	now := time.Now()
	a.hub.peers["child"] = &peer{uid: 12, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	a.hub.peers["outsider"] = &peer{uid: 99, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	a.hub.peers["agent"] = &peer{uid: 7, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	data, err := a.overview(agentMemberRequest(a, "/manage/index/overview", M{"online_days": 7}))
	if err != nil {
		t.Fatal(err)
	}
	got := obj(data)
	if number(obj(got["totals"])["online_users"]) != 1 || number(obj(got["today"])["peak_users"]) != 1 {
		t.Fatalf("agent online leaked other connections: %#v", got)
	}
	if samples, ok := got["online"].([]M); !ok || len(samples) != 0 {
		t.Fatalf("pre-upgrade online history should be empty: %#v", got["online"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentOverviewRecordsEachEnabledAgentIncludingEmptyTeams(t *testing.T) {
	a, mock := testApp(t)
	t.Cleanup(func() { a.hub.peers = map[string]*peer{} })
	now := time.Now()
	a.hub.peers["child-a"] = &peer{uid: 12, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	a.hub.peers["child-a-device"] = &peer{uid: 12, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	a.hub.peers["child-b"] = &peer{uid: 13, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	a.hub.peers["outsider"] = &peer{uid: 99, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
	mock.ExpectExec("INSERT INTO `yu_imgo_online_sample`").WithArgs(now.Unix()/60*60, 3, 4).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT .*agent_user_id.*descendant_user_id.*FROM `yu_user`.*JOIN `yu_imgo_admin_role`.*LEFT JOIN `yu_imgo_referral_path`").
		WillReturnRows(sqlmock.NewRows([]string{"agent_user_id", "descendant_user_id"}).AddRow(7, 12).AddRow(7, 12).AddRow(7, 13).AddRow(8, nil))
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_online_sample`").WithArgs(int64(7), now.Unix()/60*60, 2, 3).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_online_sample`").WithArgs(int64(8), now.Unix()/60*60, 0, 0).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := a.recordOnline(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentOverviewGlobalSampleSurvivesTeamLookupFailure(t *testing.T) {
	a, mock := testApp(t)
	now := time.Now()
	lookupError := errors.New("team lookup unavailable")
	mock.ExpectExec("INSERT INTO `yu_imgo_online_sample`").WithArgs(now.Unix()/60*60, 0, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT .*agent_user_id.*descendant_user_id.*FROM `yu_user`").
		WillReturnError(lookupError)
	if err := a.recordOnline(context.Background(), now); !errors.Is(err, lookupError) {
		t.Fatalf("lookup error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentOverviewContinuesAfterAgentSampleFailure(t *testing.T) {
	a, mock := testApp(t)
	now := time.Now()
	writeError := errors.New("one agent sample failed")
	mock.ExpectExec("INSERT INTO `yu_imgo_online_sample`").WithArgs(now.Unix()/60*60, 0, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT .*agent_user_id.*descendant_user_id.*FROM `yu_user`").
		WillReturnRows(sqlmock.NewRows([]string{"agent_user_id", "descendant_user_id"}).AddRow(7, nil).AddRow(8, nil).AddRow(9, nil))
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_online_sample`").WithArgs(int64(7), now.Unix()/60*60, 0, 0).
		WillReturnError(writeError)
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_online_sample`").WithArgs(int64(8), now.Unix()/60*60, 0, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_agent_online_sample`").WithArgs(int64(9), now.Unix()/60*60, 0, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := a.recordOnline(context.Background(), now); !errors.Is(err, writeError) {
		t.Fatalf("sample error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOverviewGlobalRolesKeepGlobalMetrics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		roleLookup bool
	}{
		{"super administrator", false},
		{"non agent administrator", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			t.Cleanup(func() { a.hub.peers = map[string]*peer{} })
			if tc.roleLookup {
				mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
					WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(0))
			}
			for _, table := range []string{"user", "group", "message", "file"} {
				mock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM `yu_" + table + "` .*WHERE .* AND 1=1$").
					WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"total", "added"}).AddRow(5, 2))
			}
			mock.ExpectQuery("SELECT COALESCE\\(MAX\\(users\\),0\\).*FROM `yu_imgo_online_sample` WHERE sample_at>=\\?").
				WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"peak_users", "peak_devices"}).AddRow(4, 4))
			for _, table := range []string{"user", "user", "message", "group", "file"} {
				mock.ExpectQuery("SELECT DATE_FORMAT.*FROM `yu_"+table+"` .*WHERE .* AND 1=1 AND .*create_time>=\\?").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"bucket", "n"}))
			}
			mock.ExpectQuery("SELECT FLOOR.*FROM `yu_imgo_online_sample` WHERE sample_at>=\\?").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"time", "users", "devices"}))
			now := time.Now()
			a.hub.peers["global"] = &peer{uid: 99, claims: claims{Exp: now.Add(time.Hour).Unix()}, done: make(chan struct{})}
			uid := int64(1)
			if tc.roleLookup {
				uid = 7
			}
			req := bankRequest(a, "/manage/index/overview", uid, M{})
			if tc.roleLookup {
				req.user["admin_role_id"] = int64(3)
			}
			data, err := a.overview(req)
			if err != nil {
				t.Fatal(err)
			}
			got := obj(data)
			if number(obj(got["totals"])["users"]) != 5 || number(obj(got["totals"])["online_users"]) != 1 || number(obj(got["today"])["peak_users"]) != 4 {
				t.Fatalf("global overview = %#v", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
