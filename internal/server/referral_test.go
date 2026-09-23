package server

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInviteCodeFormat(t *testing.T) {
	for _, tc := range []struct {
		code  string
		valid bool
	}{
		{"000000", true}, {"123456", true}, {"999999", true},
		{"12345", false}, {"1234567", false}, {"ABCDEF", false},
		{"ABCDE", false}, {"12A456", false}, {"１２３４５６", false}, {"123456 ", false},
	} {
		if validInviteCode(tc.code) != tc.valid {
			t.Fatalf("validInviteCode(%q)=%v", tc.code, !tc.valid)
		}
	}
	for i := 0; i < 100; i++ {
		code, err := newInviteCode()
		if err != nil || !validInviteCode(code) {
			t.Fatalf("generated code %q: %v", code, err)
		}
	}
}

func TestLegacyInviteCodeFormat(t *testing.T) {
	for _, tc := range []struct {
		code  string
		valid bool
	}{
		{"ABCDE", true}, {"ZZZZZ", true}, {"abcde", false}, {"123456", false},
	} {
		if validLegacyInviteCode(tc.code) != tc.valid {
			t.Fatalf("validLegacyInviteCode(%q)=%v, want %v", tc.code, !tc.valid, tc.valid)
		}
	}
}

func TestResolveLegacyInviteCode(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT l.user_id FROM `yu_imgo_referral_legacy_code` l JOIN `yu_user` u ON u.user_id=l.user_id WHERE l.invite_code=? AND u.status=1 AND u.delete_time=0")).
		WithArgs("ABCDE").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_user` WHERE user_id=? AND status=1 AND delete_time=0")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	if inviterID, err := a.resolveInviter(context.Background(), a.db, "ABCDE"); err != nil || inviterID != 7 {
		t.Fatalf("legacy invite code resolved to %d: %v", inviterID, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInviteStatusOnlyReturnsDirectCount(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT invite_code FROM `yu_imgo_referral` WHERE user_id=?")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"invite_code"}).AddRow("123456"))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_imgo_referral_path` p JOIN `yu_user` u").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
	mock.ExpectQuery("SELECT DATE_FORMAT\\(sign_date,'%Y-%m-%d'\\) sign_date FROM `yu_imgo_check_in` WHERE user_id=\\? ORDER BY sign_date DESC").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"sign_date"}).AddRow(checkInDate(time.Now())).AddRow(checkInDate(time.Now().AddDate(0, 0, -1))).AddRow(checkInDate(time.Now().AddDate(0, 0, -3))))
	result, err := a.inviteStatus(bankRequest(a, "/enterprise/invite/status", 7, M{}))
	if err != nil {
		t.Fatal(err)
	}
	status := result.(M)
	if status["invite_code"] != "123456" || status["direct_count"] != int64(2) || status["streak_days"] != int64(2) || status["earned_points"] != nil || status["surpassed_percent"] != nil || status["team_count"] != nil {
		t.Fatalf("unexpected public summary: %#v", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInviteStreak(t *testing.T) {
	today := time.Date(2026, time.September, 19, 0, 0, 0, 0, checkInZone)
	for _, tc := range []struct {
		name string
		days []string
		want int64
	}{
		{"today and yesterday", []string{"2026-09-19", "2026-09-18", "2026-09-16"}, 2},
		{"yesterday still live", []string{"2026-09-18", "2026-09-17"}, 2},
		{"missed two days", []string{"2026-09-17", "2026-09-16"}, 0},
		{"never signed", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := inviteStreak(tc.days, today); got != tc.want {
				t.Fatalf("inviteStreak(%v)=%d, want %d", tc.days, got, tc.want)
			}
		})
	}
}
