package server

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func expectMemberInviteCodeLookup(mock sqlmock.Sqlmock, oldCode string) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT invite_code FROM `yu_imgo_referral` WHERE user_id=?")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"invite_code"}).AddRow(oldCode))
}

func TestAdminCanChangeInviteCode(t *testing.T) {
	a, mock := testApp(t)
	expectMemberInviteCodeLookup(mock, "123456")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_referral` WHERE invite_code=?")).
		WithArgs("654321").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_imgo_referral` SET invite_code=? WHERE user_id=?")).
		WithArgs("654321", int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.manageUser(bankRequest(a, "/manage/user/setInviteCode", 1, M{"user_id": 7, "invite_code": "654321"}))
	if err != nil || result.(M)["invite_code"] != "654321" {
		t.Fatalf("change invite code: result=%v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCannotReuseInviteCode(t *testing.T) {
	a, mock := testApp(t)
	expectMemberInviteCodeLookup(mock, "123456")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_referral` WHERE invite_code=?")).
		WithArgs("654321").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(8))
	_, err := a.manageUser(bankRequest(a, "/manage/user/setInviteCode", 1, M{"user_id": 7, "invite_code": "654321"}))
	if err == nil || err.Error() != "邀请码已被其他成员使用" {
		t.Fatalf("duplicate code error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInviteCodeUniqueIndexHandlesConcurrentSave(t *testing.T) {
	a, mock := testApp(t)
	expectMemberInviteCodeLookup(mock, "123456")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_referral` WHERE invite_code=?")).
		WithArgs("654321").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_imgo_referral` SET invite_code=? WHERE user_id=?")).
		WithArgs("654321", int64(7)).WillReturnError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"})
	_, err := a.manageUser(bankRequest(a, "/manage/user/setInviteCode", 1, M{"user_id": 7, "invite_code": "654321"}))
	if err == nil || err.Error() != "邀请码已被其他成员使用" {
		t.Fatalf("concurrent duplicate code error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminInviteCodeRequiresSixDigits(t *testing.T) {
	a, mock := testApp(t)
	for _, code := range []string{"12345", "1234567", "ABCDEF", "12A456"} {
		_, err := a.manageUser(bankRequest(a, "/manage/user/setInviteCode", 1, M{"user_id": 7, "invite_code": code}))
		if err == nil || !errors.As(err, new(clientError)) {
			t.Fatalf("code %q error = %v", code, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
