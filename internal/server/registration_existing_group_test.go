package server

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRegistrationAutomationJoinsConfiguredExistingGroup(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT group_id,owner_id FROM `yu_group` WHERE group_id=\\? AND status=1 AND COALESCE\\(delete_time,0\\)=0 FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(77, 9))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_group_user` WHERE group_id=\\? AND status=1").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(3))
	mock.ExpectExec("INSERT INTO `yu_group_user`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	tx, err := a.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	state := M{"group_id": int64(12), "group_num": int64(4)}
	gid, err := a.applyRegistrationAutomation(context.Background(), tx, 88, M{"status": 0}, M{
		"status": 1, "mode": autoGroupModeExisting, "group_id": 77, "userMax": 20,
	}, M{}, state)
	if err != nil {
		t.Fatal(err)
	}
	if gid != 77 {
		t.Fatalf("gid = %d, want 77", gid)
	}
	if number(state["group_id"]) != 12 || number(state["group_num"]) != 4 {
		t.Fatalf("existing-group mode changed rotation state: %#v", state)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrationAutomationRejectsFullConfiguredExistingGroup(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT group_id,owner_id FROM `yu_group` WHERE group_id=\\? AND status=1 AND COALESCE\\(delete_time,0\\)=0 FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(77, 9))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_group_user` WHERE group_id=\\? AND status=1").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(20))
	mock.ExpectRollback()

	tx, err := a.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.applyRegistrationAutomation(context.Background(), tx, 88, M{"status": 0}, M{
		"status": 1, "mode": autoGroupModeExisting, "group_id": 77, "userMax": 20,
	}, M{}, M{})
	var client clientError
	if !errors.As(err, &client) || client.code != 400 || client.message != "指定群聊人数已达上限，请更换群聊或调整人数上限" {
		t.Fatalf("full existing group error = %#v", err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoGroupModeKeepsLegacyConfigurationCompatible(t *testing.T) {
	for _, tc := range []struct {
		value M
		want  string
		valid bool
	}{
		{M{}, autoGroupModeCreate, true},
		{M{"mode": ""}, autoGroupModeCreate, true},
		{M{"mode": autoGroupModeCreate}, autoGroupModeCreate, true},
		{M{"mode": autoGroupModeExisting}, autoGroupModeExisting, true},
		{M{"mode": "unknown"}, "", false},
	} {
		got, err := normalizedAutoGroupMode(tc.value)
		if tc.valid && (err != nil || got != tc.want) {
			t.Errorf("normalizedAutoGroupMode(%#v) = %q, %v", tc.value, got, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("normalizedAutoGroupMode(%#v) accepted invalid mode", tc.value)
		}
	}
}
