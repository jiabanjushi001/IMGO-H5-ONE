package server

import (
	"testing"
	"time"
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
