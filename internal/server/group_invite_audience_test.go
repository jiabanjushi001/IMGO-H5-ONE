package server

import (
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestGroupInviteAudiencePredicateCombinesFriendsAndDescendants(t *testing.T) {
	a, _ := testApp(t)
	scope := adminScope{AgentUserID: 7, referralTable: a.t("imgo_referral_path")}
	predicate, args := a.groupInviteAudiencePredicate(groupInviteAudience{
		InviterID:       7,
		DescendantScope: &scope,
	}, "u", false)
	if !strings.Contains(predicate, "`yu_friend` invite_friend") || !strings.Contains(predicate, "`yu_imgo_referral_path` scope_path") || !strings.Contains(predicate, " OR ") {
		t.Fatalf("predicate must combine friends and descendants: %s", predicate)
	}
	if !reflect.DeepEqual(args, []any{int64(7), int64(7), int64(7)}) {
		t.Fatalf("args=%v", args)
	}
}

func TestSuperAdminGroupCandidateListIncludesAnyActiveUser(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT group_id,setting FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"group_id", "setting"}).AddRow(5, `{}`),
	)
	mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(1)).WillReturnRows(
		sqlmock.NewRows([]string{"role"}).AddRow(1),
	)
	mock.ExpectQuery("SELECT user_id,account,realname,avatar,name_py FROM `yu_user` u WHERE status=1").
		WithArgs(int64(1), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname"}).AddRow(99, "any99", "任意用户"))

	r := bankRequest(a, "/enterprise/group/getAllUser", 1, M{"group_id": "group-5"})
	result, err := a.group(r)
	if err != nil || len(result.([]M)) != 1 || number(result.([]M)[0]["user_id"]) != 99 {
		t.Fatalf("result=%v err=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGroupAddRejectsUserOutsideFriendsAndDescendants(t *testing.T) {
	a, mock := testApp(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/enterprise/group/addGroupUser", nil)
	mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"group_id", "owner_id", "name", "setting"}).AddRow(5, 8, "测试群", `{"manager_invite":1}`),
	)
	mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"role"}).AddRow(2),
	)
	expectAgentMemberScope(mock)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT group_id,owner_id,setting FROM `yu_group`.*FOR UPDATE").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"group_id", "owner_id", "setting"}).AddRow(5, 8, `{"manager_invite":1}`),
	)
	mock.ExpectQuery("SELECT role FROM `yu_group_user`.*FOR SHARE").WithArgs(int64(5), int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"role"}).AddRow(2),
	)
	mock.ExpectQuery("SELECT 1 FROM `yu_user` u.*invite_friend.*OR.*scope_path.*FOR SHARE").
		WithArgs(int64(99), int64(7), int64(7), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))
	mock.ExpectRollback()

	_, err := a.group(&request{
		app: a,
		c:   c,
		p:   M{"group_id": "group-5", "user_ids": []any{99}},
		user: M{
			"user_id":       int64(7),
			"admin_role_id": int64(3),
		},
	})
	var denied clientError
	if !errors.As(err, &denied) || denied.code != 403 {
		t.Fatalf("expected 403 for unrelated target, err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGroupCandidateSearchPaginatesAndReturnsTotal(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT group_id,setting FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"group_id", "setting"}).AddRow(5, "{}"),
	)
	mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(1)).WillReturnRows(
		sqlmock.NewRows([]string{"role"}).AddRow(1),
	)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_user`.*u.realname LIKE.*u.account LIKE").
		WithArgs(int64(1), int64(5), "%beta%", "%beta%").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(3))
	mock.ExpectQuery("SELECT user_id,account,realname,avatar,name_py FROM `yu_user`.*LIMIT \\? OFFSET \\?").
		WithArgs(int64(1), int64(5), "%beta%", "%beta%", int64(2), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname"}).AddRow(99, "beta99", "成员"))

	r := bankRequest(a, "/enterprise/group/getalluser", 1, M{"group_id": "group-5", "keywords": "beta", "page": 2, "limit": 2})
	result, err := a.group(r)
	if err != nil || r.count != 3 || r.page != 2 || len(result.([]M)) != 1 {
		t.Fatalf("result=%v count=%v page=%v err=%v", result, r.count, r.page, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
