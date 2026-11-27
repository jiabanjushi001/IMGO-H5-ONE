package server

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectWithdrawalConfig(mock sqlmock.Sqlmock, enabled int, minYuan int) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM `yu_config` WHERE name=? AND status=1")).
		WithArgs("withdrawal").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(js(M{"enabled": enabled, "minYuan": minYuan})))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT withdrawal_enabled FROM `yu_user` WHERE user_id=? AND delete_time=0")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_enabled"}).AddRow(nil))
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE u.user_id=\\? AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	expectNoWithdrawalMentor(mock, 7)
}

func expectNoWithdrawalMentor(mock sqlmock.Sqlmock, userID int64) {
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}))
}

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
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(1234, 8000))
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_cents\\),0\\) frozen_cents FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND status=\\?").
		WithArgs(int64(7), withdrawalStatusFrozen).
		WillReturnRows(sqlmock.NewRows([]string{"frozen_cents"}).AddRow(3000))
	expectWithdrawalConfig(mock, 1, 100)
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/status", 7, M{"user_id": 99}))
	if err != nil {
		t.Fatal(err)
	}
	data := result.(M)
	if data["available_cents"] != int64(1234) || data["pending_cents"] != int64(8000) || data["frozen_cents"] != int64(3000) {
		t.Fatalf("wrong balance: %#v", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalIdempotent(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}).AddRow(41, 1234, 0, "bank"))
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-1234567890"}))
	if err != nil || number(result.(M)["withdrawal_id"]) != 41 {
		t.Fatalf("idempotent request failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalRespectsDisabledSetting(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-disabled-123").
		WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 0, 100)
	_, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "100.00", "request_id": "wd-disabled-123", "payment_type": "bank"}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 403 {
		t.Fatalf("disabled withdrawal must reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalRespectsMinimumAmount(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-minimum-123").
		WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 1, 100)
	_, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "99.99", "request_id": "wd-minimum-123", "payment_type": "alipay"}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 400 {
		t.Fatalf("below-minimum withdrawal must reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalSnapshotsAlipay(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-alipay-123").
		WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 1, 100)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(20000, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-alipay-123").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalMethods(mock, 7, nil)
	mock.ExpectQuery("SELECT receipt_name,account_cipher,account_last4,qr_file_id FROM `yu_imgo_payment_method` WHERE user_id=\\? AND payment_type=\\? AND status=1 FOR UPDATE").
		WithArgs(int64(7), "alipay").WillReturnRows(sqlmock.NewRows([]string{"receipt_name", "account_cipher", "account_last4", "qr_file_id"}).AddRow("测试用户", "", "", 30))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents-\\?,pending_cents=pending_cents\\+\\?,updated_at=\\? WHERE user_id=\\? AND available_cents>=\\?").
		WithArgs(int64(10000), int64(10000), sqlmock.AnyArg(), int64(7), int64(10000)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_withdrawal`").
		WithArgs(int64(7), "wd-alipay-123", int64(10000), "alipay", "测试用户", "支付宝收款码", "", "", "", int64(30), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(61, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "withdraw", "wd-alipay-123", int64(-10000), int64(10000), int64(61), int64(7), "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	expectNoWithdrawalMentor(mock, 7)
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "100.00", "request_id": "wd-alipay-123", "payment_type": "alipay"}))
	if err != nil || str(result.(M)["payment_type"]) != "alipay" {
		t.Fatalf("alipay withdrawal failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletWithdrawalConcurrentRetryReturnsExistingRequest(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 1, 1)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(0, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}).AddRow(51, 1234, 0, "bank"))
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
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 1, 1)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(500, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
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
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 1, 1)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(5000, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-1234567890").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalMethods(mock, 7, nil)
	mock.ExpectQuery("SELECT receipt_name,bank_name,branch_name,account_cipher,account_last4,0 AS qr_file_id FROM `yu_imgo_bank_card` WHERE user_id=\\? AND status=1 FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"receipt_name", "bank_name", "branch_name", "account_cipher", "account_last4", "qr_file_id"}).AddRow("测试用户", "中国银行", "朝阳支行", "v1:cipher", "1234", 0))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents-\\?,pending_cents=pending_cents\\+\\?,updated_at=\\? WHERE user_id=\\? AND available_cents>=\\?").
		WithArgs(int64(1234), int64(1234), sqlmock.AnyArg(), int64(7), int64(1234)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_withdrawal`").
		WithArgs(int64(7), "wd-1234567890", int64(1234), "bank", "测试用户", "中国银行", "朝阳支行", "v1:cipher", "1234", int64(0), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(51, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "withdraw", "wd-1234567890", int64(-1234), int64(1234), int64(51), int64(7), "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	expectNoWithdrawalMentor(mock, 7)
	result, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-1234567890"}))
	if err != nil || number(result.(M)["withdrawal_id"]) != 51 {
		t.Fatalf("withdrawal failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithdrawalNotifiesOnlyNearestMentor(t *testing.T) {
	a, mock := testApp(t)
	mentor := &peer{
		id: "mentor-device", uid: 9,
		claims: claims{UID: 9, Exp: time.Now().Add(time.Hour).Unix()},
		send:   make(chan []byte, 1), done: make(chan struct{}),
	}
	a.hub.peers[mentor.id] = mentor
	defer delete(a.hub.peers, mentor.id)

	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id WHERE p.descendant_user_id=\\? AND p.ancestor_user_id<>p.descendant_user_id AND u.status=1 AND u.delete_time=0 AND r.status=1 AND r.agent_mode=1 ORDER BY p.depth ASC,p.ancestor_user_id ASC LIMIT 1").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}).AddRow(9))
	a.notifyMentorWithdrawal(context.Background(), 7, 61, 1234, M{"account": "member7", "realname": "成员七"})

	select {
	case raw := <-mentor.send:
		var event M
		if err := json.Unmarshal(raw, &event); err != nil {
			t.Fatal(err)
		}
		data := obj(event["data"])
		if event["type"] != "withdrawalSubmitted" || number(data["withdrawal_id"]) != 61 || number(data["amount_cents"]) != 1234 || str(data["account"]) != "member7" {
			t.Fatalf("unexpected mentor notification: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("mentor did not receive withdrawal notification")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletHistoryUsesAuthenticatedUser(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) n FROM `yu_imgo_withdrawal` WHERE user_id=\\?").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type,bank_name,account_last4,created_at,processed_at,remark FROM `yu_imgo_withdrawal` WHERE user_id=\\? ORDER BY withdrawal_id DESC LIMIT \\? OFFSET \\?").
		WithArgs(int64(7), int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type", "bank_name", "account_last4", "created_at", "processed_at", "remark"}).AddRow(51, 1234, 0, "bank", "中国银行", "1234", 100, 0, ""))
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

func TestManageWalletDebitDoesNotRequireBankCard(t *testing.T) {
	a, mock := testApp(t)
	if route := a.routes[normalizedPath("/manage/wallet/debit")]; route.super || route.permission != "manage.finance" {
		t.Fatal("debit must require finance permission")
	}
	mock.ExpectQuery("SELECT entry_id,available_delta FROM `yu_imgo_wallet_entry` WHERE user_id=\\? AND event='debit' AND request_id=\\?").
		WithArgs(int64(7), "debit-1234567890").WillReturnRows(sqlmock.NewRows([]string{"entry_id", "available_delta"}))
	mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(5000, 0))
	mock.ExpectExec("UPDATE `yu_imgo_wallet` SET available_cents=available_cents-\\?,updated_at=\\? WHERE user_id=\\? AND available_cents>=\\?").
		WithArgs(int64(1200), sqlmock.AnyArg(), int64(7), int64(1200)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").
		WithArgs(int64(7), "debit-1234567890", int64(-1200), int64(1), "纠正重复充值", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(62, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/debit", 1, M{"user_id": 7, "amount": "12.00", "request_id": "debit-1234567890", "note": "纠正重复充值"}))
	if err != nil || result.(M)["debited"] != true {
		t.Fatalf("debit failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageWalletDebitRejectsNegativeBalance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT entry_id,available_delta FROM `yu_imgo_wallet_entry` WHERE user_id=\\? AND event='debit' AND request_id=\\?").
		WithArgs(int64(7), "debit-too-much-1").WillReturnRows(sqlmock.NewRows([]string{"entry_id", "available_delta"}))
	mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=\\? AND delete_time=0").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(500, 0))
	mock.ExpectRollback()
	_, err := a.manageWallet(bankRequest(a, "/manage/wallet/debit", 1, M{"user_id": 7, "amount": "12.00", "request_id": "debit-too-much-1", "note": "纠正重复充值"}))
	var client clientError
	if !errors.As(err, &client) || client.code != 409 {
		t.Fatalf("overdraft debit must be rejected: %v", err)
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

func TestManageWalletFreezeKeepsPendingBalance(t *testing.T) {
	a, mock := testApp(t)
	if route := a.routes[normalizedPath("/manage/wallet/freeze")]; route.super || route.permission != "manage.finance" {
		t.Fatal("freeze must require finance permission")
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE withdrawal_id=\\? FOR UPDATE").
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(7, 1234, 0))
	mock.ExpectExec("UPDATE `yu_imgo_withdrawal` SET status=\\?,processed_at=\\?,processed_by=\\?,remark=\\? WHERE withdrawal_id=\\? AND status=0").
		WithArgs(withdrawalStatusFrozen, sqlmock.AnyArg(), int64(1), "资料待复核", int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/freeze", 1, M{"withdrawal_id": 51, "frozen": 1, "remark": "资料待复核"}))
	if err != nil || result.(M)["processed"] != true || number(result.(M)["status"]) != withdrawalStatusFrozen {
		t.Fatalf("freeze failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageWalletUnfreezeKeepsPendingBalance(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE withdrawal_id=\\? FOR UPDATE").
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(7, 1234, withdrawalStatusFrozen))
	mock.ExpectExec("UPDATE `yu_imgo_withdrawal` SET status=0,processed_at=0,processed_by=0,remark='' WHERE withdrawal_id=\\? AND status=\\?").
		WithArgs(int64(51), withdrawalStatusFrozen).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(bankRequest(a, "/manage/wallet/freeze", 1, M{"withdrawal_id": 51, "frozen": 0}))
	if err != nil || result.(M)["processed"] != true || number(result.(M)["status"]) != 0 {
		t.Fatalf("unfreeze failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManageWalletReviewRejectsFrozenOrder(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE withdrawal_id=\\? FOR UPDATE").
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(7, 1234, withdrawalStatusFrozen))
	mock.ExpectRollback()
	_, err := a.manageWallet(bankRequest(a, "/manage/wallet/review", 1, M{"withdrawal_id": 51, "status": 1, "remark": "已转账"}))
	var ce clientError
	if !errors.As(err, &ce) || ce.code != 409 {
		t.Fatalf("frozen order review must reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
