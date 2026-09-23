package server

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWalletRechargeBonus(t *testing.T) {
	for _, tc := range []struct {
		mode, value string
		principal   int64
		want        int64
		valid       bool
	}{
		{"none", "", 12345, 0, true},
		{"percent", "10", 12345, 1235, true},
		{"percent", "2.5", 10000, 250, true},
		{"fixed", "5.25", 10000, 525, true},
		{"percent", "0", 10000, 0, false},
		{"percent", "1000.01", 10000, 0, false},
		{"fixed", "0", 10000, 0, false},
		{"unknown", "10", 10000, 0, false},
	} {
		got, valid := walletRechargeBonus(tc.principal, tc.mode, tc.value)
		if got != tc.want || valid != tc.valid {
			t.Errorf("%s %q on %d: got %d %t; want %d %t", tc.mode, tc.value, tc.principal, got, valid, tc.want, tc.valid)
		}
	}
}

func TestAdminRechargeCreatesOrderPrincipalBonusAndLedgerAtomically(t *testing.T) {
	a, mock := testApp(t)
	for _, action := range []string{"recharge", "withdraw"} {
		if route := a.routes[normalizedPath("/manage/wallet/"+action)]; route.super || route.permission != "manage.finance" {
			t.Fatalf("%s must require finance permission", action)
		}
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents"}).AddRow(1000))
	mock.ExpectQuery("SELECT order_id,amount_cents,bonus_mode,bonus_value,bonus_cents,total_cents,note,status FROM `yu_imgo_recharge_order` WHERE user_id=\\? AND request_id=\\?").WithArgs(int64(7), "recharge-1234567890").WillReturnRows(sqlmock.NewRows([]string{"order_id"}))
	mock.ExpectExec("INSERT INTO `yu_imgo_recharge_order`").WithArgs(int64(7), "recharge-1234567890", int64(12345), "percent", "10", int64(1235), int64(13580), "后台测试充值", sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(22, 1))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents\\+\\?,updated_at=\\? WHERE user_id=\\?").WithArgs(int64(13580), sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WithArgs(int64(7), "recharge-1234567890", int64(12345), int64(22), int64(1), "后台测试充值", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(31, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WithArgs(int64(7), "recharge-1234567890", int64(1235), int64(22), int64(1), "后台测试充值", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(32, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/recharge", 1, M{"user_id": 7, "request_id": "recharge-1234567890", "amount": "123.45", "bonus_mode": "percent", "bonus_value": "10", "note": "后台测试充值"}))
	if err != nil || number(result.(M)["total_cents"]) != 13580 || number(result.(M)["order_id"]) != 22 {
		t.Fatalf("wrong recharge result: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminRechargeRejectsChangedDuplicateWithoutCrediting(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents FROM `yu_imgo_wallet`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents"}).AddRow(1000))
	mock.ExpectQuery("SELECT order_id,amount_cents,bonus_mode,bonus_value,bonus_cents,total_cents,note,status FROM `yu_imgo_recharge_order`").WithArgs(int64(7), "recharge-1234567890").WillReturnRows(sqlmock.NewRows([]string{"order_id", "amount_cents", "bonus_mode", "bonus_value", "bonus_cents", "total_cents", "note", "status"}).AddRow(22, 1000, "none", "", 0, 1000, "已入账", 1))
	mock.ExpectRollback()
	_, err := a.manageWallet(bankRequest(a, "/manage/wallet/recharge", 1, M{"user_id": 7, "request_id": "recharge-1234567890", "amount": "20", "bonus_mode": "none", "bonus_value": "", "note": "已入账"}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 409 {
		t.Fatalf("changed duplicate must be rejected: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminWithdrawFreezesBalanceCreatesOrderAndLedger(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(5000, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,request_note FROM `yu_imgo_withdrawal`").WithArgs(int64(7), "admin-withdraw-12345").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id"}))
	mock.ExpectQuery("SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4 FROM `yu_imgo_bank_card`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"receipt_name", "bank_name", "branch_name", "account_cipher", "account_last4"}).AddRow("测试用户", "中国银行", "朝阳支行", "encrypted", "1234"))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents-\\?,pending_cents=pending_cents\\+\\?").WithArgs(int64(1234), int64(1234), sqlmock.AnyArg(), int64(7), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_withdrawal`").WithArgs(int64(7), "admin-withdraw-12345", int64(1234), "测试用户", "中国银行", "朝阳支行", "encrypted", "1234", sqlmock.AnyArg(), int64(1), "后台申请").WillReturnResult(sqlmock.NewResult(41, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WithArgs(int64(7), "admin-withdraw-12345", int64(-1234), int64(1234), int64(41), int64(1), "后台申请", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(99, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/withdraw", 1, M{"user_id": 7, "request_id": "admin-withdraw-12345", "amount": "12.34", "note": "后台申请"}))
	if err != nil || number(result.(M)["withdrawal_id"]) != 41 {
		t.Fatalf("wrong withdrawal: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminRechargeLedgerFailureRollsBackOrderAndBalance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents FROM `yu_imgo_wallet`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents"}).AddRow(100))
	mock.ExpectQuery("SELECT order_id,amount_cents,bonus_mode,bonus_value,bonus_cents,total_cents,note,status FROM `yu_imgo_recharge_order`").WithArgs(int64(7), "recharge-failure-123").WillReturnRows(sqlmock.NewRows([]string{"order_id"}))
	mock.ExpectExec("INSERT INTO `yu_imgo_recharge_order`").WithArgs(int64(7), "recharge-failure-123", int64(1000), "fixed", "2", int64(200), int64(1200), "测试原子事务", sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec("UPDATE `yu_imgo_wallet`").WithArgs(int64(1200), sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WithArgs(int64(7), "recharge-failure-123", int64(1000), int64(42), int64(1), "测试原子事务", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(51, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WithArgs(int64(7), "recharge-failure-123", int64(200), int64(42), int64(1), "测试原子事务", sqlmock.AnyArg()).WillReturnError(errors.New("ledger unavailable"))
	mock.ExpectRollback()
	_, err := a.manageWallet(bankRequest(a, "/manage/wallet/recharge", 1, M{"user_id": 7, "request_id": "recharge-failure-123", "amount": "10", "bonus_mode": "fixed", "bonus_value": "2", "note": "测试原子事务"}))
	if err == nil {
		t.Fatal("failed ledger must fail the entire recharge")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminWithdrawRequiresApprovedBankCardBeforeFreezing(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(5000, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,request_note FROM `yu_imgo_withdrawal`").WithArgs(int64(7), "admin-withdraw-12345").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id"}))
	mock.ExpectQuery("SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4 FROM `yu_imgo_bank_card`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"receipt_name"}))
	mock.ExpectRollback()
	_, err := a.manageWallet(bankRequest(a, "/manage/wallet/withdraw", 1, M{"user_id": 7, "request_id": "admin-withdraw-12345", "amount": "12.34", "note": "后台申请"}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 409 {
		t.Fatalf("unapproved card must prevent withdrawal: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletEntriesStayWithinAuthenticatedUser(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_imgo_wallet_entry` WHERE user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery("SELECT entry_id,event,available_delta,pending_delta,reference_id,note,created_at FROM `yu_imgo_wallet_entry` WHERE user_id=\\? ORDER BY entry_id DESC LIMIT \\? OFFSET \\?").
		WithArgs(int64(7), int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"entry_id", "event", "available_delta", "pending_delta", "reference_id", "note", "created_at"}).AddRow(3, "recharge", 1000, 0, 2, "测试", 100))
	r := bankRequest(a, "/enterprise/wallet/entries", 7, M{"user_id": 99})
	result, err := a.wallet(r)
	if err != nil || r.count != 1 || len(result.([]M)) != 1 {
		t.Fatalf("wrong wallet entries: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRechargeListRequiresFinancePermission(t *testing.T) {
	a, mock := testApp(t)
	if a.routes[normalizedPath("/manage/wallet/recharges")].permission != "manage.finance" || a.routes[normalizedPath("/manage/wallet/entries")].permission != "manage.finance" {
		t.Fatal("financial audit routes must require finance permission")
	}
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_imgo_recharge_order` o LEFT JOIN `yu_user` u ON u.user_id=o.user_id WHERE 1=1 AND o.user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectQuery("SELECT o.order_id,o.user_id,o.amount_cents,o.bonus_mode,o.bonus_value,o.bonus_cents,o.total_cents,o.note,o.status,o.created_at,o.created_by,u.account,u.realname FROM `yu_imgo_recharge_order` o LEFT JOIN `yu_user` u ON u.user_id=o.user_id WHERE 1=1 AND o.user_id=\\? ORDER BY o.order_id DESC LIMIT \\? OFFSET \\?").
		WithArgs(int64(7), int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"order_id"}))
	r := bankRequest(a, "/manage/wallet/recharges", 1, M{"user_id": 7})
	result, err := a.manageWallet(r)
	if err != nil || r.count != 0 || len(result.([]M)) != 0 {
		t.Fatalf("wrong recharge list: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
