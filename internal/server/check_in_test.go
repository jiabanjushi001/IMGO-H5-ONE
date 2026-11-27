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

type centsBetween struct{ minimum, maximum int64 }

func (want centsBetween) Match(value driver.Value) bool {
	got, ok := value.(int64)
	return ok && got >= want.minimum && got <= want.maximum && got%100 == 0
}

const checkInTestConfig = `{"rewardEnabled":1,"cycleDays":10,"highDays":3,"highMinYuan":7,"highMaxYuan":10,"normalMinYuan":3,"normalMaxYuan":5}`

func expectCheckInConfig(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("checkIn").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(checkInTestConfig))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}))
}

func TestCheckInChinaDateBoundary(t *testing.T) {
	before := time.Date(2026, 9, 18, 15, 59, 59, 0, time.UTC)
	if got := checkInDate(before); got != "2026-09-18" {
		t.Fatalf("before midnight: %s", got)
	}
	if got := checkInDate(before.Add(time.Second)); got != "2026-09-19" {
		t.Fatalf("after midnight: %s", got)
	}
}

func TestCheckInStatusUsesAuthenticatedUser(t *testing.T) {
	a, mock := testApp(t)
	day := checkInDate(time.Now())
	expectCheckInConfig(mock)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) total,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN 1 ELSE 0 END\\),0\\) signed_today,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN reward_cents ELSE 0 END\\),0\\) today_reward_cents,COALESCE\\(MAX\\(CASE WHEN sign_date=\\? THEN cycle_day ELSE 0 END\\),0\\) cycle_day,COALESCE\\(SUM\\(reward_cents\\),0\\) total_reward_cents FROM `yu_imgo_check_in` WHERE user_id=\\?").
		WithArgs(day, day, day, int64(7)).WillReturnRows(sqlmock.NewRows([]string{"total", "signed_today", "today_reward_cents", "cycle_day", "total_reward_cents"}).AddRow(3, 1, 800, 3, 2100))
	result, err := a.checkIn(bankRequest(a, "/enterprise/checkin/status", 7, M{"user_id": 99}))
	if err != nil {
		t.Fatal(err)
	}
	status := result.(M)
	if status["date"] != day || status["signed_today"] != true || status["total_days"] != int64(3) || status["cycle_day"] != int64(3) || status["today_reward_cents"] != int64(800) || status["total_reward_cents"] != int64(2100) {
		t.Fatalf("bad status: %#v", status)
	}
	for _, hidden := range []string{"reward_min_cents", "reward_max_cents", "high_days", "high_min_cents", "high_max_cents", "normal_min_cents", "normal_max_cents"} {
		if _, exposed := status[hidden]; exposed {
			t.Fatalf("user response exposes %s: %#v", hidden, status)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckInRewardBands(t *testing.T) {
	settings, err := decodeCheckInSettings(defaultConfig("checkIn"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ day, min, max int64 }{{1, 700, 1000}, {3, 700, 1000}, {4, 300, 500}, {10, 300, 500}} {
		minimum, maximum := checkInRewardBounds(settings, tc.day)
		if minimum != tc.min || maximum != tc.max {
			t.Fatalf("day %d bounds = %d..%d", tc.day, minimum, maximum)
		}
		for i := 0; i < 20; i++ {
			reward, randomErr := randomCheckInReward(settings, tc.day)
			if randomErr != nil || reward < tc.min || reward > tc.max || reward%100 != 0 {
				t.Fatalf("day %d reward = %d (%v)", tc.day, reward, randomErr)
			}
		}
	}
	if _, err = normalizeCheckInConfig(M{"rewardEnabled": 1, "cycleDays": 10, "highDays": 3, "highMinYuan": 10, "highMaxYuan": 7, "normalMinYuan": 3, "normalMaxYuan": 5}); err == nil {
		t.Fatal("accepted an inverted reward range")
	}
}

func TestCheckInCreditsWalletOnce(t *testing.T) {
	a, mock := testApp(t)
	day := checkInDate(time.Now())
	expectCheckInConfig(mock)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT available_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents"}).AddRow(0))
	mock.ExpectQuery("SELECT reward_cents,cycle_day FROM `yu_imgo_check_in` WHERE user_id=\\? AND sign_date=\\? FOR UPDATE").WithArgs(int64(7), day).WillReturnRows(sqlmock.NewRows([]string{"reward_cents", "cycle_day"}))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) total,COALESCE\\(SUM\\(reward_cents\\),0\\) total_reward_cents FROM `yu_imgo_check_in` WHERE user_id=\\?").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"total", "total_reward_cents"}).AddRow(3, 2400))
	mock.ExpectExec("INSERT INTO `yu_imgo_check_in`").WithArgs(int64(7), day, sqlmock.AnyArg(), int64(4), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents\\+\\?,updated_at=\\? WHERE user_id=\\?").WithArgs(centsBetween{300, 500}, sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WithArgs(int64(7), "checkin-"+strings.ReplaceAll(day, "-", ""), centsBetween{300, 500}, int64(7), "第4天签到奖励（10天周期）", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	result, err := a.checkIn(bankRequest(a, "/enterprise/checkin/submit", 7, M{}))
	if err != nil {
		t.Fatal(err)
	}
	status := result.(M)
	if status["already_signed"] != false || status["total_days"] != int64(4) || status["cycle_day"] != int64(4) {
		t.Fatalf("unexpected first sign response: %#v", status)
	}
	todayReward := number(status["today_reward_cents"])
	if todayReward < 300 || todayReward > 500 || todayReward%100 != 0 || number(status["total_reward_cents"]) != 2400+todayReward {
		t.Fatalf("unexpected reward totals: %#v", status)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckInDuplicateReturnsOriginalReward(t *testing.T) {
	a, mock := testApp(t)
	day := checkInDate(time.Now())
	expectCheckInConfig(mock)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents"}).AddRow(800))
	mock.ExpectQuery("SELECT reward_cents,cycle_day FROM `yu_imgo_check_in` WHERE user_id=\\? AND sign_date=\\? FOR UPDATE").WithArgs(int64(7), day).WillReturnRows(sqlmock.NewRows([]string{"reward_cents", "cycle_day"}).AddRow(800, 3))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) total,COALESCE\\(SUM\\(reward_cents\\),0\\) total_reward_cents FROM `yu_imgo_check_in` WHERE user_id=\\?").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"total", "total_reward_cents"}).AddRow(3, 2100))
	mock.ExpectCommit()
	result, err := a.checkIn(bankRequest(a, "/enterprise/checkin/submit", 7, M{}))
	if err != nil {
		t.Fatal(err)
	}
	status := result.(M)
	if status["already_signed"] != true || status["cycle_day"] != int64(3) || status["today_reward_cents"] != int64(800) || status["total_reward_cents"] != int64(2100) {
		t.Fatalf("unexpected duplicate response: %#v", status)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckInRoutesRequireLogin(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/enterprise/checkin/status", "/enterprise/checkin/submit"} {
		w := httptest.NewRecorder()
		a.Router().ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(`{"user_id":1}`)))
		var body M
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || number(body["code"]) != -1 {
			t.Fatalf("%s: %s (%v)", path, w.Body.String(), err)
		}
	}
}
