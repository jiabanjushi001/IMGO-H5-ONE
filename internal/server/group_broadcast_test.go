package server

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGroupBroadcastRouteUsesGroupPermission(t *testing.T) {
	a, _ := testApp(t)
	route, ok := a.routes["/manage/group/broadcast"]
	if !ok || route.super || route.permission != "manage.groups" {
		t.Fatalf("broadcast route = %#v", route)
	}
}

func TestChooseBroadcastCustomerKeepsAssignmentOrBalances(t *testing.T) {
	configured := []int64{11, 12, 13}
	if got := chooseBroadcastCustomer(configured, 12, 0); got != 12 {
		t.Fatalf("existing assignment changed to %d", got)
	}
	for offset, want := range []int64{11, 12, 13, 11} {
		if got := chooseBroadcastCustomer(configured, 99, offset); got != want {
			t.Fatalf("offset %d: got %d want %d", offset, got, want)
		}
	}
	if got := chooseBroadcastCustomer(nil, 12, 0); got != 0 {
		t.Fatalf("empty configuration returned %d", got)
	}
}

func TestBroadcastAgentMapUsesEachMembersNearestMentor(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT p.descendant_user_id,p.ancestor_user_id,p.depth FROM `yu_imgo_referral_path`").
		WithArgs(int64(20), int64(30)).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "ancestor_user_id", "depth"}).
			AddRow(20, 7, 1).
			AddRow(20, 6, 2))

	got, err := a.broadcastAgentMap(context.Background(), []M{
		{"user_id": int64(10), "agent_mode": int64(1), "role_status": int64(1)},
		{"user_id": int64(20), "agent_mode": int64(0), "role_status": int64(0)},
		{"user_id": int64(30), "agent_mode": int64(0), "role_status": int64(0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[10] != 10 || got[20] != 7 || got[30] != 0 {
		t.Fatalf("agent map = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
