package server

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWithdrawalPasswordFormat(t *testing.T) {
	for _, password := range []string{"123456", "000000", "987654"} {
		if !validWithdrawalPassword(password) {
			t.Fatalf("valid password rejected: %q", password)
		}
	}
	for _, password := range []string{"", "12345", "1234567", "abcdef", "１２３４５６", "12 456"} {
		if validWithdrawalPassword(password) {
			t.Fatalf("invalid password accepted: %q", password)
		}
	}
}

func TestSetWithdrawalPasswordStoresOnlyHash(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO `yu_imgo_withdrawal_password` (user_id,password_hash,failed_attempts,locked_until,updated_at) VALUES (?,?,0,0,?)")).
		WithArgs(int64(7), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/setWithdrawalPassword", 7, M{"password": "123456", "password_confirmation": "123456"}))
	if err != nil || result.(M)["withdrawal_password_set"] != true {
		t.Fatalf("set withdrawal password failed: %#v %v", result, err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetWithdrawalPasswordRejectsInvalidConfirmation(t *testing.T) {
	a, mock := testApp(t)
	_, err := a.wallet(bankRequest(a, "/enterprise/wallet/setWithdrawalPassword", 7, M{"password": "123456", "password_confirmation": "654321"}))
	var client clientError
	if !errors.As(err, &client) || client.code != 400 {
		t.Fatalf("mismatched confirmation must fail: %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyWithdrawalPasswordCountsFailuresAndClearsThem(t *testing.T) {
	a, mock := testApp(t)
	hash, err := hashPassword("123456")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT password_hash,failed_attempts,locked_until FROM `yu_imgo_withdrawal_password` WHERE user_id=?")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash", "failed_attempts", "locked_until"}).AddRow(hash, 0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_imgo_withdrawal_password` SET failed_attempts=?,locked_until=?,updated_at=? WHERE user_id=?")).
		WithArgs(int64(1), int64(0), sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	err = a.verifyWithdrawalPassword(context.Background(), a.db, 7, "654321")
	var client clientError
	if !errors.As(err, &client) || client.code != 400 || !strings.Contains(client.Error(), "还可尝试4次") {
		t.Fatalf("wrong password must increment failure count: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT password_hash,failed_attempts,locked_until FROM `yu_imgo_withdrawal_password` WHERE user_id=?")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash", "failed_attempts", "locked_until"}).AddRow(hash, 2, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `yu_imgo_withdrawal_password` SET failed_attempts=0,locked_until=0,updated_at=? WHERE user_id=?")).
		WithArgs(sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err = a.verifyWithdrawalPassword(context.Background(), a.db, 7, "123456"); err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletStatusReportsRequiredButUnsetPassword(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(1234, 0))
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_cents\\),0\\) frozen_cents FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND status=\\?").
		WithArgs(int64(7), withdrawalStatusFrozen).
		WillReturnRows(sqlmock.NewRows([]string{"frozen_cents"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("withdrawal").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"enabled":1,"minYuan":100,"passwordRequired":1}`))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled"}).AddRow(nil))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	expectNoWithdrawalMentor(mock, 7)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_withdrawal_password` WHERE user_id=?")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))

	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/status", 7, M{}))
	if err != nil {
		t.Fatal(err)
	}
	data := result.(M)
	if data["withdrawal_password_required"] != true || data["withdrawal_password_set"] != false {
		t.Fatalf("wrong password status: %#v", data)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalRequiresPasswordBeforeOpeningTransaction(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-password-123").
		WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("withdrawal").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"enabled":1,"minYuan":100,"passwordRequired":1}`))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled"}).AddRow(nil))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	expectNoWithdrawalMentor(mock, 7)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT password_hash,failed_attempts,locked_until FROM `yu_imgo_withdrawal_password` WHERE user_id=?")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"password_hash", "failed_attempts", "locked_until"}))

	_, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "100.00", "request_id": "wd-password-123", "payment_type": "bank"}))
	var client clientError
	if !errors.As(err, &client) || client.code != 409 {
		t.Fatalf("unset required password must reject: %v", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
