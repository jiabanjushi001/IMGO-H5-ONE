package server

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminScopeRoleMatrix(t *testing.T) {
	ctx := context.Background()
	t.Run("super administrator has global scope without role lookup", func(t *testing.T) {
		a, mock := testApp(t)
		scope, err := a.adminScope(ctx, M{"user_id": int64(1), "admin_role_id": int64(9)})
		if err != nil || !scope.Global || scope.AgentUserID != 0 {
			t.Fatalf("scope = %#v, error = %v", scope, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	for _, tc := range []struct {
		name     string
		roleMode int
		want     adminScope
	}{
		{"custom role has global scope", 0, adminScope{Global: true}},
		{"agent role uses current user as root", 1, adminScope{AgentUserID: 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT agent_mode FROM `yu_imgo_admin_role` WHERE role_id=? AND status=1")).
				WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(tc.roleMode))
			scope, err := a.adminScope(ctx, M{"user_id": int64(7), "admin_role_id": int64(3)})
			if err != nil || scope.Global != tc.want.Global || scope.AgentUserID != tc.want.AgentUserID {
				t.Fatalf("scope = %#v, error = %v", scope, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("ordinary user is denied", func(t *testing.T) {
		a, _ := testApp(t)
		_, err := a.adminScope(ctx, M{"user_id": int64(7), "admin_role_id": int64(0)})
		assertDenied(t, err)
	})
	t.Run("disabled role is denied", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}))
		_, err := a.adminScope(ctx, M{"user_id": int64(7), "admin_role_id": int64(3)})
		assertDenied(t, err)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAdminScopeUserPredicateUsesPrefixedPathAndSafeAlias(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(1))
	scope, err := a.adminScope(context.Background(), M{"user_id": int64(7), "admin_role_id": int64(3)})
	if err != nil {
		t.Fatal(err)
	}
	predicate, args := scope.userPredicate("u")
	want := "(u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id))"
	if predicate != want || !reflect.DeepEqual(args, []any{int64(7), int64(7)}) {
		t.Fatalf("predicate = %q, args = %#v", predicate, args)
	}
	globalPredicate, globalArgs := (adminScope{Global: true}).userPredicate("u")
	if globalPredicate != "1=1" || len(globalArgs) != 0 {
		t.Fatalf("global predicate = %q, args = %#v", globalPredicate, globalArgs)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("unsafe SQL alias was accepted")
		}
	}()
	scope.userPredicate("u; DROP TABLE user")
}

func TestRequireScopedUserChecksExistenceAndDescendant(t *testing.T) {
	ctx := context.Background()
	t.Run("agent descendant is allowed", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM `yu_user` u WHERE u.user_id=? AND u.delete_time=0 AND (u.user_id<>? AND EXISTS (SELECT 1 FROM `yu_imgo_referral_path` scope_path WHERE scope_path.ancestor_user_id=? AND scope_path.descendant_user_id=u.user_id))")).
			WithArgs(int64(12), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
		if err := a.requireScopedUser(ctx, a.db, adminScope{AgentUserID: 7}, 12); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("matching self path cannot authorize agent's own record", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=\\? AND u.delete_time=0 AND \\(u.user_id<>\\?").
			WithArgs(int64(7), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
		assertDenied(t, a.requireScopedUser(ctx, a.db, adminScope{AgentUserID: 7}, 7))
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("unrelated user is denied", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").
			WithArgs(int64(99), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}))
		assertDenied(t, a.requireScopedUser(ctx, a.db, adminScope{AgentUserID: 7}, 99))
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("global scope still checks user existence", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM `yu_user` u WHERE u.user_id=? AND u.delete_time=0 AND 1=1")).
			WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"1"}))
		assertDenied(t, a.requireScopedUser(ctx, a.db, adminScope{Global: true}, 12))
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("database failure is returned", func(t *testing.T) {
		a, mock := testApp(t)
		failure := errors.New("database unavailable")
		mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").
			WithArgs(int64(12), int64(7), int64(7)).WillReturnError(failure)
		if err := a.requireScopedUser(ctx, a.db, adminScope{AgentUserID: 7}, 12); !errors.Is(err, failure) {
			t.Fatalf("error = %v, want database failure", err)
		}
	})
}

func TestNearestAgentSelectsClosestEnabledAncestor(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(14))
	got, err := a.nearestAgent(context.Background(), a.db, 20)
	if err != nil || got != 14 {
		t.Fatalf("nearest agent = %d, error = %v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNearestAgentReturnsZeroWithoutMentor(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path`").WithArgs(int64(20)).
		WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}))
	got, err := a.nearestAgent(context.Background(), a.db, 20)
	if err != nil || got != 0 {
		t.Fatalf("nearest agent = %d, error = %v", got, err)
	}
}

func TestNearestAgentDoesNotSelectSelfLoop(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p .*p.ancestor_user_id<>p.descendant_user_id").
		WithArgs(int64(20)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(20))
	got, err := a.nearestAgent(context.Background(), a.db, 20)
	if err != nil || got != 0 {
		t.Fatalf("self path selected as mentor: id=%d error=%v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminAccessInfoReportsEffectiveAgentMode(t *testing.T) {
	ctx := context.Background()
	a, _ := testApp(t)
	for _, user := range []M{{"user_id": int64(1), "admin_role_id": int64(3)}, {"user_id": int64(7), "admin_role_id": int64(0)}} {
		info, err := a.adminAccessInfo(ctx, user)
		mode, present := info["agent_mode"]
		if err != nil || !present || number(mode) != 0 {
			t.Fatalf("access info = %#v, error = %v", info, err)
		}
	}
	for _, tc := range []struct{ status, mode, want int }{{1, 1, 1}, {0, 1, 0}} {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT name,status,agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
			WillReturnRows(sqlmock.NewRows([]string{"name", "status", "agent_mode"}).AddRow("导师", tc.status, tc.mode))
		if tc.status == 1 {
			mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
				WillReturnRows(sqlmock.NewRows([]string{"permission_key"}))
		}
		info, err := a.adminAccessInfo(ctx, M{"user_id": int64(7), "admin_role_id": int64(3)})
		mode, present := info["agent_mode"]
		if err != nil || !present || number(mode) != int64(tc.want) {
			t.Fatalf("access info = %#v, error = %v", info, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
