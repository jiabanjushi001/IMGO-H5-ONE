package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

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
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) total,COALESCE\\(SUM\\(sign_date=\\?\\),0\\) signed_today FROM `yu_imgo_check_in` WHERE user_id=\\?").
		WithArgs(day, int64(7)).WillReturnRows(sqlmock.NewRows([]string{"total", "signed_today"}).AddRow(3, 0))
	result, err := a.checkIn(bankRequest(a, "/enterprise/checkin/status", 7, M{"user_id": 99}))
	if err != nil {
		t.Fatal(err)
	}
	status := result.(M)
	if status["date"] != day || status["signed_today"] != false || status["total_days"] != int64(3) {
		t.Fatalf("bad status: %#v", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckInOncePerDayAndAccount(t *testing.T) {
	for _, tc := range []struct {
		name          string
		userID        int64
		alreadySigned bool
	}{
		{"first sign", 7, false},
		{"duplicate sign", 7, true},
		{"other account", 8, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			day := checkInDate(time.Now())
			insert := mock.ExpectExec("INSERT INTO `yu_imgo_check_in` \\(user_id,sign_date,created_at\\) VALUES \\(\\?,\\?,\\?\\)").
				WithArgs(tc.userID, day, sqlmock.AnyArg())
			if tc.alreadySigned {
				insert.WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate"})
			} else {
				insert.WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectQuery("SELECT COUNT\\(\\*\\) total,COALESCE\\(SUM\\(sign_date=\\?\\),0\\) signed_today FROM `yu_imgo_check_in` WHERE user_id=\\?").
				WithArgs(day, tc.userID).WillReturnRows(sqlmock.NewRows([]string{"total", "signed_today"}).AddRow(4, 1))
			result, err := a.checkIn(bankRequest(a, "/enterprise/checkin/submit", tc.userID, M{"user_id": 999}))
			if err != nil {
				t.Fatal(err)
			}
			status := result.(M)
			if status["signed_today"] != true || status["total_days"] != int64(4) || status["already_signed"] != tc.alreadySigned {
				t.Fatalf("unexpected response: %#v", status)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
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
