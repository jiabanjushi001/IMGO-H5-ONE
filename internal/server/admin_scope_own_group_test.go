package server

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAgentOwnGroupScope(t *testing.T) {
	a, mock := testApp(t)
	scope := adminScope{AgentUserID: 7, referralTable: a.t("imgo_referral_path")}
	predicate, args := a.groupScopePredicate(scope, "g")
	if !strings.Contains(predicate, "scope_owner.user_id=? OR EXISTS") || !reflect.DeepEqual(args, []any{int64(7), int64(7)}) {
		t.Fatalf("group scope must include owner and descendants: %s %v", predicate, args)
	}
	for _, transactional := range []bool{false, true} {
		var db DB = a.db
		if transactional {
			mock.ExpectBegin()
			tx, err := a.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			db = tx
			mock.ExpectQuery("SELECT owner_id FROM `yu_group`.*FOR UPDATE").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(7))
			mock.ExpectRollback()
			defer tx.Rollback()
		} else {
			mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(7))
		}
		if err := a.requireScopedGroup(context.Background(), db, scope, 9); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOwnGroupDoesNotAllowSelfUserManagement(t *testing.T) {
	a, mock := testApp(t)
	scope := adminScope{AgentUserID: 7, referralTable: a.t("imgo_referral_path")}
	mock.ExpectQuery("SELECT 1 FROM `yu_user`.*u.user_id<>\\?").WithArgs(int64(7), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}))
	assertDenied(t, a.requireScopedUser(context.Background(), a.db, scope, 7))
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
