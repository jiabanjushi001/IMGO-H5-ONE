package server

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestParseWalletAmount(t *testing.T) {
	for _, tc := range []struct {
		input string
		cents int64
		valid bool
	}{
		{"0.01", 1, true}, {"12", 1200, true}, {"12.3", 1230, true}, {"999999999.99", 99999999999, true},
		{"", 0, false}, {"0", 0, false}, {"0.00", 0, false}, {"-1", 0, false}, {"1.234", 0, false}, {"1e2", 0, false}, {"1000000000", 0, false},
	} {
		got, ok := parseWalletAmount(tc.input)
		if got != tc.cents || ok != tc.valid {
			t.Errorf("%q: got %d %t, want %d %t", tc.input, got, ok, tc.cents, tc.valid)
		}
	}
}

func TestWalletStatusUsesAuthenticatedUser(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(1234, 200))
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/status", 7, M{"user_id": 99}))
	if err != nil {
		t.Fatal(err)
	}
	data := result.(M)
	if data["available_cents"] != int64(1234) || data["pending_cents"] != int64(200) {
		t.Fatalf("wrong balance: %#v", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalIdempotent(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}).AddRow(41, 1234, 0))
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-1234567890"}))
	if err != nil || number(result.(M)["withdrawal_id"]) != 41 {
		t.Fatalf("idempotent request failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalConcurrentRetryReturnsExistingRequest(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(0, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}).AddRow(51, 1234, 0))
	mock.ExpectRollback()
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-1234567890"}))
	if err != nil || number(result.(M)["withdrawal_id"]) != 51 {
		t.Fatalf("concurrent retry should return existing request: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalRejectsInsufficientBalance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(500, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}))
	mock.ExpectRollback()
	_, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-1234567890"}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 409 {
		t.Fatalf("insufficient balance must reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalFreezesBalanceAndSnapshotsCard(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(5000, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status"}))
	mock.ExpectQuery("SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4 FROM `yu_imgo_bank_card` WHERE user_id=\\? AND status=1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"receipt_name", "bank_name", "branch_name", "account_cipher", "account_last4"}).AddRow("测试用户", "中国银行", "朝阳支行", "v1:cipher", "1234"))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents-\\?,pending_cents=pending_cents\\+\\?,updated_at=\\? WHERE user_id=\\? AND available_cents>=\\?").
		WithArgs(int64(1234), int64(1234), sqlmock.AnyArg(), int64(7), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_withdrawal`").
		WithArgs(int64(7), "wd-1234567890", int64(1234), "测试用户", "中国银行", "朝阳支行", "v1:cipher", "1234", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(51, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "withdraw", "wd-1234567890", int64(-1234), int64(1234), int64(51), int64(7), "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-1234567890"}))
	if err != nil || number(result.(M)["withdrawal_id"]) != 51 {
		t.Fatalf("withdrawal failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletHistoryUsesAuthenticatedUser(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_imgo_withdrawal` WHERE user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,bank_name,account_last4,created_at,processed_at,remark FROM `yu_imgo_withdrawal` WHERE user_id=\\? ORDER BY withdrawal_id DESC LIMIT \\? OFFSET \\?").
		WithArgs(int64(7), int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "bank_name", "account_last4", "created_at", "processed_at", "remark"}).AddRow(51, 1234, 0, "中国银行", "1234", 100, 0, ""))
	r := bankRequest(a, "/enterprise/wallet/history", 7, M{"user_id": 99})
	result, err := a.wallet(r)
	if err != nil || r.count != 1 || len(result.([]M)) != 1 {
		t.Fatalf("wrong withdrawal history: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageWalletCreditIsAuditedAndIdempotent(t *testing.T) {
	a, mock := testApp(t)
	if route := a.routes[normalizedPath("/manage/wallet/credit")]; route.super || route.permission != "manage.finance" {
		t.Fatal("credit must require finance permission")
	}
	mock.ExpectQuery("SELECT entry_id,available_delta FROM `yu_imgo_wallet_entry` WHERE user_id=\\? AND event='credit' AND request_id=\\?").
		WithArgs(int64(7), "credit-1234567890").WillReturnRows(sqlmock.NewRows([]string{"entry_id", "available_delta"}))
	mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(100, 0))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents\\+\\?,updated_at=\\? WHERE user_id=\\?").
		WithArgs(int64(1200), sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "credit-1234567890", int64(1200), int64(1), "后台测试入账", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(61, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/credit", 1, M{"user_id": 7, "amount": "12.00", "request_id": "credit-1234567890", "note": "后台测试入账"}))
	if err != nil || result.(M)["credited"] != true {
		t.Fatalf("credit failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageWalletRejectRefundsFrozenBalance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE withdrawal_id=\\? FOR UPDATE").
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(7, 1234, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(500, 1234))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents\\+\\?,pending_cents=pending_cents-\\?,updated_at=\\? WHERE user_id=\\? AND pending_cents>=\\?").
		WithArgs(int64(1234), int64(1234), sqlmock.AnyArg(), int64(7), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `yu_imgo_withdrawal` SET status=\\?,processed_at=\\?,processed_by=\\?,remark=\\? WHERE withdrawal_id=\\? AND status=0").
		WithArgs(int64(2), sqlmock.AnyArg(), int64(1), "资料不符", int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "refund", "wd-51", int64(1234), int64(-1234), int64(51), int64(1), "资料不符", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(62, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/review", 1, M{"withdrawal_id": 51, "status": 2, "remark": "资料不符"}))
	if err != nil || result.(M)["processed"] != true {
		t.Fatalf("rejection failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageWalletPaidClearsFrozenBalance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE withdrawal_id=\\? FOR UPDATE").
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(7, 1234, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(500, 1234))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents\\+\\?,pending_cents=pending_cents-\\?,updated_at=\\? WHERE user_id=\\? AND pending_cents>=\\?").
		WithArgs(int64(0), int64(1234), sqlmock.AnyArg(), int64(7), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `yu_imgo_withdrawal` SET status=\\?,processed_at=\\?,processed_by=\\?,remark=\\? WHERE withdrawal_id=\\? AND status=0").
		WithArgs(int64(1), sqlmock.AnyArg(), int64(1), "已通过银行转账", int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "paid", "wd-51", int64(0), int64(-1234), int64(51), int64(1), "已通过银行转账", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(63, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/review", 1, M{"withdrawal_id": 51, "status": 1, "remark": "已通过银行转账"}))
	if err != nil || result.(M)["processed"] != true {
		t.Fatalf("paid review failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
