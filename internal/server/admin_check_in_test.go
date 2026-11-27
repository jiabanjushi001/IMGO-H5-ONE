package server

import (
	"database/sql/driver"
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestManageUserListFiltersReferralDescendants(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scope    string
		keyword  string
		depthSQL string
	}{
		{name: "direct", scope: "direct", depthSQL: " AND p.depth=1"},
		{name: "all with member keyword", scope: "all", keyword: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			where := "delete_time=0 AND user_id IN (SELECT p.descendant_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` inviter ON inviter.user_id=p.ancestor_user_id WHERE inviter.account=? AND inviter.delete_time=0" + tc.depthSQL + ")"
			args := []driver.Value{"wuhuA1"}
			params := M{"referral_scope": tc.scope, "keywords": " wuhuA1 "}
			mock.ExpectQuery("SELECT COUNT.*account=\\?").WithArgs("wuhuA1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
			if tc.keyword != "" {
				where += " AND (realname LIKE ? OR account LIKE ? OR email LIKE ?)"
				params["keywords"] = tc.keyword
				for range 3 {
					args = append(args, "%"+tc.keyword+"%")
				}
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` WHERE " + where)).WithArgs(args...).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `yu_user` WHERE " + where + " ORDER BY user_id DESC LIMIT ? OFFSET ?")).WithArgs(append(args, int64(20), int64(0))...).WillReturnRows(sqlmock.NewRows([]string{"user_id", "account"}))
			r := bankRequest(a, "/manage/user/index", 1, params)
			result, err := a.manageUser(r)
			if err != nil || r.count != 0 || len(result.([]M)) != 0 {
				t.Fatalf("unexpected filtered result: count=%d result=%#v err=%v", r.count, result, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManageUserListRejectsInvalidReferralScope(t *testing.T) {
	a, _ := testApp(t)
	for _, params := range []M{
		{"referral_scope": "unknown", "referrer_account": "wuhuA1"},
		{"referral_scope": "direct"},
		{"mentor_only": 2},
	} {
		if _, err := a.manageUser(bankRequest(a, "/manage/user/index", 1, params)); err == nil {
			t.Fatalf("expected invalid filter to be rejected: %#v", params)
		}
	}
}

func TestManageUserListIncludesCheckInSummary(t *testing.T) {
	a, mock := testApp(t)
	today := checkInDate(time.Now())
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_user` WHERE delete_time=0").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE delete_time=0 ORDER BY user_id DESC LIMIT \\? OFFSET \\?").
		WithArgs(int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "realname", "account", "password", "salt"}).
		AddRow(8, "用户八", "u8", "hash", "salt").AddRow(7, "用户七", "u7", "hash", "salt"))
	mock.ExpectQuery("SELECT user_id,COUNT\\(\\*\\) total_days,DATE_FORMAT\\(MAX\\(sign_date\\),'%Y-%m-%d'\\) last_date,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN 1 ELSE 0 END\\),0\\) signed_today,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN reward_cents ELSE 0 END\\),0\\) today_reward_cents,COALESCE\\(SUM\\(reward_cents\\),0\\) total_reward_cents FROM `yu_imgo_check_in` WHERE user_id IN \\(\\?,\\?\\) GROUP BY user_id").
		WithArgs(today, today, int64(8), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "total_days", "last_date", "signed_today", "today_reward_cents", "total_reward_cents"}).
		AddRow(7, 3, "2026-09-18", 1, 300, 1800))
	mock.ExpectQuery("SELECT w.user_id,w.available_cents,w.pending_cents,COALESCE\\(SUM\\(d.amount_cents\\),0\\) frozen_cents FROM `yu_imgo_wallet` w LEFT JOIN `yu_imgo_withdrawal` d ON d.user_id=w.user_id AND d.status=\\? WHERE w.user_id IN \\(\\?,\\?\\) GROUP BY w.user_id,w.available_cents,w.pending_cents").
		WithArgs(withdrawalStatusFrozen, int64(8), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "available_cents", "pending_cents", "frozen_cents"}).
		AddRow(8, 10000, 5000, 3000).AddRow(7, 2000, 0, 0))
	mock.ExpectQuery("SELECT user_id,invite_code FROM `yu_imgo_referral` WHERE user_id IN \\(\\?,\\?\\)").
		WithArgs(int64(8), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "invite_code"}).AddRow(8, "ABCDE").AddRow(7, "FGHIJ"))
	mock.ExpectQuery("SELECT p.ancestor_user_id,COUNT\\(\\*\\) team_count,COALESCE\\(SUM\\(p.depth=1\\),0\\) direct_count FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.descendant_user_id WHERE p.ancestor_user_id IN \\(\\?,\\?\\) AND u.delete_time=0 GROUP BY p.ancestor_user_id").
		WithArgs(int64(8), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id", "team_count", "direct_count"}).AddRow(8, 3, 2))
	mock.ExpectQuery("SELECT p.descendant_user_id,parent.user_id parent_user_id,parent.account parent_account,parent.realname parent_name").WithArgs(int64(8), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "parent_user_id", "parent_account", "parent_name"}).AddRow(8, 7, "u7", "用户七"))
	mock.ExpectQuery("SELECT p.descendant_user_id,mentor.user_id mentor_user_id").WithArgs(int64(8), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "mentor_user_id", "mentor_account", "mentor_name"}))
	result, err := a.manageUser(bankRequest(a, "/manage/user/index", 1, M{}))
	if err != nil {
		t.Fatal(err)
	}
	users := result.([]M)
	if users[0]["checkin_days"] != int64(0) || users[0]["checkin_today"] != false || users[0]["checkin_last_date"] != "" || number(users[0]["checkin_today_reward_cents"]) != 0 || number(users[0]["checkin_total_reward_cents"]) != 0 {
		t.Fatalf("unsigned user: %#v", users[0])
	}
	if users[1]["checkin_days"] != int64(3) || users[1]["checkin_today"] != true || users[1]["checkin_last_date"] != "2026-09-18" || number(users[1]["checkin_today_reward_cents"]) != 300 || number(users[1]["checkin_total_reward_cents"]) != 1800 {
		t.Fatalf("signed user: %#v", users[1])
	}
	if number(users[0]["available_cents"]) != 10000 || number(users[0]["pending_cents"]) != 5000 || number(users[0]["frozen_cents"]) != 3000 || number(users[0]["balance_cents"]) != 13000 || number(users[1]["balance_cents"]) != 2000 {
		t.Fatalf("wallet summaries: %#v", users)
	}
	if users[0]["invite_code"] != "ABCDE" || users[0]["direct_invite_count"] != int64(2) || users[0]["team_count"] != int64(3) || users[1]["invite_code"] != "FGHIJ" {
		t.Fatalf("referral summaries: %#v", users)
	}
	if number(users[0]["parent_user_id"]) != 7 || users[0]["parent_account"] != "u7" || users[0]["parent_name"] != "用户七" || number(users[1]["parent_user_id"]) != 0 {
		t.Fatalf("parent summaries: %#v", users)
	}
	if users[1]["password"] != nil || users[1]["salt"] != nil {
		t.Fatal("member listing leaked secrets")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageUserDetailIncludesCheckInSummary(t *testing.T) {
	a, mock := testApp(t)
	today := checkInDate(time.Now())
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE user_id=\\? AND delete_time=0").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "realname", "password", "salt"}).AddRow(7, "用户七", "hash", "salt"))
	mock.ExpectQuery("SELECT user_id,COUNT\\(\\*\\) total_days,DATE_FORMAT\\(MAX\\(sign_date\\),'%Y-%m-%d'\\) last_date,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN 1 ELSE 0 END\\),0\\) signed_today,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN reward_cents ELSE 0 END\\),0\\) today_reward_cents,COALESCE\\(SUM\\(reward_cents\\),0\\) total_reward_cents FROM `yu_imgo_check_in` WHERE user_id IN \\(\\?\\) GROUP BY user_id").
		WithArgs(today, today, int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "total_days", "last_date", "signed_today", "today_reward_cents", "total_reward_cents"}).AddRow(7, 5, "2026-09-17", 0, 0, 2500))
	mock.ExpectQuery("SELECT w.user_id,w.available_cents,w.pending_cents,COALESCE\\(SUM\\(d.amount_cents\\),0\\) frozen_cents FROM `yu_imgo_wallet` w LEFT JOIN `yu_imgo_withdrawal` d ON d.user_id=w.user_id AND d.status=\\? WHERE w.user_id IN \\(\\?\\) GROUP BY w.user_id,w.available_cents,w.pending_cents").
		WithArgs(withdrawalStatusFrozen, int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "available_cents", "pending_cents", "frozen_cents"}).AddRow(7, 1000, 500, 500))
	mock.ExpectQuery("SELECT user_id,invite_code FROM `yu_imgo_referral` WHERE user_id IN \\(\\?\\)").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "invite_code"}).AddRow(7, "FGHIJ"))
	mock.ExpectQuery("SELECT p.ancestor_user_id,COUNT\\(\\*\\) team_count,COALESCE\\(SUM\\(p.depth=1\\),0\\) direct_count FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.descendant_user_id WHERE p.ancestor_user_id IN \\(\\?\\) AND u.delete_time=0 GROUP BY p.ancestor_user_id").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id", "team_count", "direct_count"}).AddRow(7, 5, 2))
	mock.ExpectQuery("SELECT p.descendant_user_id,parent.user_id parent_user_id,parent.account parent_account,parent.realname parent_name").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "parent_user_id", "parent_account", "parent_name"}).AddRow(7, 6, "u6", "用户六"))
	mock.ExpectQuery("SELECT p.descendant_user_id,mentor.user_id mentor_user_id").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"descendant_user_id", "mentor_user_id", "mentor_account", "mentor_name"}))
	result, err := a.manageUser(bankRequest(a, "/manage/user/detail", 1, M{"user_id": 7}))
	if err != nil {
		t.Fatal(err)
	}
	u := result.(M)
	if u["checkin_days"] != int64(5) || u["checkin_today"] != false || u["checkin_last_date"] != "2026-09-17" || number(u["checkin_today_reward_cents"]) != 0 || number(u["checkin_total_reward_cents"]) != 2500 {
		t.Fatalf("detail: %#v", u)
	}
	if number(u["available_cents"]) != 1000 || number(u["pending_cents"]) != 500 || number(u["frozen_cents"]) != 500 || number(u["balance_cents"]) != 1500 {
		t.Fatalf("wallet detail: %#v", u)
	}
	if u["invite_code"] != "FGHIJ" || u["direct_invite_count"] != int64(2) || u["team_count"] != int64(5) {
		t.Fatalf("referral detail: %#v", u)
	}
	if number(u["parent_user_id"]) != 6 || u["parent_account"] != "u6" || u["parent_name"] != "用户六" {
		t.Fatalf("parent detail: %#v", u)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageUserCheckInHistoryReturnsNewestPage(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_imgo_check_in` WHERE user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(3))
	mock.ExpectQuery("SELECT DATE_FORMAT\\(sign_date,'%Y-%m-%d'\\) sign_date,created_at signed_at,cycle_day,reward_cents FROM `yu_imgo_check_in` WHERE user_id=\\? ORDER BY sign_date DESC LIMIT \\? OFFSET \\?").
		WithArgs(int64(7), int64(2), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"sign_date", "signed_at", "cycle_day", "reward_cents"}).AddRow("2026-09-16", int64(1789560000), 4, 500))
	r := bankRequest(a, "/manage/user/checkInHistory", 1, M{"user_id": 7, "page": 2, "limit": 2})
	result, err := a.manageUser(r)
	if err != nil {
		t.Fatal(err)
	}
	entries := result.([]M)
	if r.count != 3 || r.page != 2 || len(entries) != 1 || entries[0]["sign_date"] != "2026-09-16" || number(entries[0]["signed_at"]) != 1789560000 || number(entries[0]["cycle_day"]) != 4 || number(entries[0]["reward_cents"]) != 500 {
		t.Fatalf("unexpected history: count=%d page=%d entries=%#v", r.count, r.page, entries)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageUserCheckInHistoryRejectsInvalidUserID(t *testing.T) {
	a, mock := testApp(t)
	_, err := a.manageUser(bankRequest(a, "/manage/user/checkInHistory", 1, M{"user_id": 0}))
	if ce, ok := err.(clientError); !ok || ce.code != 400 {
		t.Fatalf("invalid user ID should be rejected: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageUserCheckInHistoryRequiresAdminLogin(t *testing.T) {
	a, _ := testApp(t)
	w := httptest.NewRecorder()
	a.Router().ServeHTTP(w, httptest.NewRequest("POST", "/manage/user/checkInHistory", strings.NewReader(`{"user_id":7}`)))
	var response M
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || number(response["code"]) != -1 {
		t.Fatalf("history should require authentication: %s (%v)", w.Body.String(), err)
	}
}

func TestMemberUsernameExactFirst(t *testing.T) {
	for _, tc := range []struct {
		name, scope, predicate, value string
		exact                         int
	}{
		{"exact user", "", "account=?", "wuhu", 1},
		{"fuzzy user", "", "account LIKE ? ESCAPE '!'", "%wuhu%", 0},
		{"fuzzy direct", "direct", "account LIKE ? ESCAPE '!'", "%wuhu%", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` WHERE delete_time=0 AND account=?")).WithArgs("wuhu").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(tc.exact))
			where := "delete_time=0 AND " + tc.predicate
			if tc.scope != "" {
				where = "delete_time=0 AND user_id IN (SELECT p.descendant_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` inviter ON inviter.user_id=p.ancestor_user_id WHERE inviter." + tc.predicate + " AND inviter.delete_time=0 AND p.depth=1)"
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_user` WHERE " + where)).WithArgs(tc.value).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `yu_user` WHERE "+where+" ORDER BY user_id DESC LIMIT ? OFFSET ?")).WithArgs(tc.value, int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
			_, err := a.manageUser(bankRequest(a, "/manage/user/index", 1, M{"keywords": " wuhu ", "referral_scope": tc.scope}))
			if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
