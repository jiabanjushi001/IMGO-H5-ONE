package server

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectWithdrawalMethods(mock sqlmock.Sqlmock, userID int64, stored any) {
	query := mock.ExpectQuery("SELECT s.withdrawal_methods FROM `yu_imgo_referral_path` p")
	query.WithArgs(userID)
	rows := sqlmock.NewRows([]string{"withdrawal_methods"})
	if stored != nil {
		rows.AddRow(stored)
	}
	query.WillReturnRows(rows)
}

func TestStoredWithdrawalMethodsDefaultAndOverride(t *testing.T) {
	defaults := storedWithdrawalMethods(nil)
	for _, typeName := range withdrawalMethodTypes {
		if !withdrawalMethodEnabled(defaults, typeName) {
			t.Fatalf("legacy settings must enable %s: %#v", typeName, defaults)
		}
	}
	override := storedWithdrawalMethods(`{"bank":true,"alipay":false,"wechat":true}`)
	if !withdrawalMethodEnabled(override, paymentTypeBank) || withdrawalMethodEnabled(override, paymentTypeAlipay) || !withdrawalMethodEnabled(override, paymentTypeWechat) {
		t.Fatalf("withdrawal method override = %#v", override)
	}
}

func TestStoredWithdrawalMethodsAcceptsDecodedJSONObjects(t *testing.T) {
	methods := storedWithdrawalMethods(map[string]any{
		paymentTypeBank:   true,
		paymentTypeAlipay: false,
		paymentTypeWechat: false,
	})
	if !withdrawalMethodEnabled(methods, paymentTypeBank) ||
		withdrawalMethodEnabled(methods, paymentTypeAlipay) ||
		withdrawalMethodEnabled(methods, paymentTypeWechat) {
		t.Fatalf("decoded database JSON switches were not preserved: %#v", methods)
	}
}

func TestNormalizeWithdrawalMethodsRequiresAllBooleanSwitches(t *testing.T) {
	got, err := normalizeWithdrawalMethods(M{"bank": true, "alipay": false, "wechat": "true"})
	if err != nil || !withdrawalMethodEnabled(got, paymentTypeBank) || withdrawalMethodEnabled(got, paymentTypeAlipay) || !withdrawalMethodEnabled(got, paymentTypeWechat) {
		t.Fatalf("normalized methods = %#v, %v", got, err)
	}
	if _, err = normalizeWithdrawalMethods(M{"bank": true, "alipay": false}); err == nil {
		t.Fatal("missing method switch must be rejected")
	}
}

func TestWalletWithdrawalRejectsMentorDisabledMethod(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\?").
		WithArgs(int64(7), "wd-method-off-123").
		WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalConfig(mock, 1, 1)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WithArgs(int64(7), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet` WHERE user_id=\\? FOR UPDATE").
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(5000, 0))
	mock.ExpectQuery("SELECT withdrawal_id,amount_cents,status,payment_type FROM `yu_imgo_withdrawal` WHERE user_id=\\? AND request_id=\\? FOR UPDATE").
		WithArgs(int64(7), "wd-method-off-123").WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id", "amount_cents", "status", "payment_type"}))
	expectWithdrawalMethods(mock, 7, `{"bank":true,"alipay":false,"wechat":true}`)
	mock.ExpectRollback()

	_, err := a.wallet(bankRequest(a, "/enterprise/wallet/withdraw", 7, M{"amount": "12.34", "request_id": "wd-method-off-123", "payment_type": "alipay"}))
	var client clientError
	if !errors.As(err, &client) || client.code != 403 {
		t.Fatalf("disabled mentor method must reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
