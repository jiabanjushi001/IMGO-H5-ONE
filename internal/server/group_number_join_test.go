package server

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGroupNumberJoinRouteUsesGroupPermission(t *testing.T) {
	a, _ := testApp(t)
	route, ok := a.routes["/manage/group/setnumberjoin"]
	if !ok || route.super || route.permission != "manage.groups" {
		t.Fatalf("number join route = %#v", route)
	}
}

func TestMentorCanSetNumberJoinOnlyForOwnGroup(t *testing.T) {
	t.Run("own group", func(t *testing.T) {
		a, mock := testApp(t)
		r := agentMemberRequest(a, "/manage/group/setNumberJoin", M{"group_id": 17, "enabled": 1})
		expectAgentMemberScope(mock)
		mock.ExpectQuery("SELECT owner_id,setting FROM `yu_group`").WithArgs(int64(17)).
			WillReturnRows(sqlmock.NewRows([]string{"owner_id", "setting"}).AddRow(7, `{"invite":1}`))
		mock.ExpectExec("UPDATE `yu_group` SET `setting`=\\? WHERE group_id=\\?").
			WithArgs(`{"invite":1,"number_join":1}`, int64(17)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(17)).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
		if _, err := a.manageGroup(r); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("another mentors group", func(t *testing.T) {
		a, mock := testApp(t)
		r := agentMemberRequest(a, "/manage/group/setNumberJoin", M{"group_id": 18, "enabled": 1})
		expectAgentMemberScope(mock)
		mock.ExpectQuery("SELECT owner_id,setting FROM `yu_group`").WithArgs(int64(18)).
			WillReturnRows(sqlmock.NewRows([]string{"owner_id", "setting"}).AddRow(8, `{}`))
		_, err := a.manageGroup(r)
		assertDenied(t, err)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSuperAdminCanSetNumberJoinForAnyGroup(t *testing.T) {
	a, mock := testApp(t)
	r := settingRequest(a, "/manage/group/setNumberJoin", M{"group_id": 22, "enabled": 0})
	mock.ExpectQuery("SELECT owner_id,setting FROM `yu_group`").WithArgs(int64(22)).
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "setting"}).AddRow(99, `{"number_join":1}`))
	mock.ExpectExec("UPDATE `yu_group` SET `setting`=\\? WHERE group_id=\\?").
		WithArgs(`{"number_join":0}`, int64(22)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(22)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	if _, err := a.manageGroup(r); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
