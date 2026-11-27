package server

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectNoPaymentMethodMentor(mock sqlmock.Sqlmock, userID int64) {
	mock.ExpectQuery("SELECT p.ancestor_user_id FROM `yu_imgo_referral_path` p JOIN `yu_user` u ON u.user_id=p.ancestor_user_id JOIN `yu_imgo_admin_role` r ON r.role_id=u.admin_role_id").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"ancestor_user_id"}))
}

func TestSaveAlipayPaymentMethodRequiresOwnedImage(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status,qr_file_id FROM `yu_imgo_payment_method` WHERE user_id=\\? AND payment_type=\\?").
		WithArgs(int64(7), "alipay").
		WillReturnRows(sqlmock.NewRows([]string{"status", "qr_file_id"}))
	mock.ExpectQuery("SELECT file_id,user_id,cate,src FROM `yu_file` WHERE file_id=\\? AND status=1").
		WithArgs(int64(30)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "src"}).AddRow(30, 7, 2, "/storage/alipay.png"))
	mock.ExpectExec("INSERT INTO `yu_imgo_payment_method`").
		WithArgs(int64(7), "alipay", "测试用户", "", "", int64(30), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPaymentMethodMentor(mock, 7)
	result, err := a.paymentMethods(bankRequest(a, "/enterprise/paymentMethod/save", 7, M{
		"payment_type": "alipay", "receipt_name": "测试用户", "qr_file_id": 30,
	}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("save alipay failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveWechatPaymentMethodRequiresOwnedImage(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status,qr_file_id FROM `yu_imgo_payment_method` WHERE user_id=\\? AND payment_type=\\?").
		WithArgs(int64(7), "wechat").
		WillReturnRows(sqlmock.NewRows([]string{"status", "qr_file_id"}))
	mock.ExpectQuery("SELECT file_id,user_id,cate,src FROM `yu_file` WHERE file_id=\\? AND status=1").
		WithArgs(int64(31)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "src"}).AddRow(31, 7, 2, "/storage/wx.png"))
	mock.ExpectExec("INSERT INTO `yu_imgo_payment_method`").
		WithArgs(int64(7), "wechat", "测试用户", "", "", int64(31), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPaymentMethodMentor(mock, 7)
	result, err := a.paymentMethods(bankRequest(a, "/enterprise/paymentMethod/save", 7, M{
		"payment_type": "wechat", "receipt_name": "测试用户", "qr_file_id": 31,
	}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("save wechat failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSavePaymentMethodRejectsPendingUpdate(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status,qr_file_id FROM `yu_imgo_payment_method` WHERE user_id=\\? AND payment_type=\\?").
		WithArgs(int64(7), "alipay").
		WillReturnRows(sqlmock.NewRows([]string{"status", "qr_file_id"}).AddRow(0, 30))
	_, err := a.paymentMethods(bankRequest(a, "/enterprise/paymentMethod/save", 7, M{
		"payment_type": "alipay", "receipt_name": "测试用户", "qr_file_id": 30,
	}))
	if err == nil {
		t.Fatal("pending payment method must not be editable")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAlipayPaymentMethodMigratesLegacyAccountToQRCode(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status,qr_file_id FROM `yu_imgo_payment_method` WHERE user_id=\\? AND payment_type=\\?").
		WithArgs(int64(7), "alipay").
		WillReturnRows(sqlmock.NewRows([]string{"status", "qr_file_id"}).AddRow(1, 0))
	mock.ExpectQuery("SELECT file_id,user_id,cate,src FROM `yu_file` WHERE file_id=\\? AND status=1").
		WithArgs(int64(30)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "src"}).AddRow(30, 7, 2, "/storage/alipay.png"))
	mock.ExpectExec("UPDATE `yu_imgo_payment_method` SET receipt_name=\\?,account_cipher=\\?,account_last4=\\?,qr_file_id=\\?,status=0").
		WithArgs("测试用户", "", "", int64(30), sqlmock.AnyArg(), int64(7), "alipay").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPaymentMethodMentor(mock, 7)
	result, err := a.paymentMethods(bankRequest(a, "/enterprise/paymentMethod/save", 7, M{
		"payment_type": "alipay", "receipt_name": "测试用户", "qr_file_id": 30,
	}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("migrate legacy alipay failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminReviewsAlipayPaymentMethod(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT p.user_id,p.payment_type,p.receipt_name,p.account_cipher,p.account_last4,p.qr_file_id,p.status,p.remark,p.version,p.created_at,p.updated_at,f.src AS qr_image FROM `yu_imgo_payment_method` p").
		WithArgs(int64(7), "alipay").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "payment_type", "receipt_name", "account_cipher", "account_last4", "qr_file_id", "status", "remark", "version", "created_at", "updated_at", "qr_image"}).AddRow(7, "alipay", "测试用户", "cipher", "5678", 0, 0, "", 3, 10, 20, ""))
	mock.ExpectExec("UPDATE `yu_imgo_payment_method` SET receipt_name=\\?,status=\\?,remark=\\?,version=version\\+1,updated_at=\\?").
		WithArgs("测试用户", int64(1), "已核对", sqlmock.AnyArg(), int64(7), "alipay", int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.manageBankCard(bankRequest(a, "/manage/bank/edit", 1, M{
		"user_id": 7, "payment_type": "alipay", "version": 3, "status": 1, "receipt_name": "测试用户", "remark": "已核对",
	}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("admin review failed: %#v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMentorReviewsTeamPaymentMethods(t *testing.T) {
	for _, typeName := range []string{"alipay", "wechat"} {
		t.Run(typeName, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			expectResourceUser(mock, 12, true)
			mock.ExpectQuery("SELECT p.user_id,p.payment_type,p.receipt_name,p.account_cipher,p.account_last4,p.qr_file_id,p.status,p.remark,p.version,p.created_at,p.updated_at,f.src AS qr_image FROM `yu_imgo_payment_method` p").
				WithArgs(int64(12), typeName).
				WillReturnRows(sqlmock.NewRows([]string{"user_id", "payment_type", "receipt_name", "account_cipher", "account_last4", "qr_file_id", "status", "remark", "version", "created_at", "updated_at", "qr_image"}).
					AddRow(12, typeName, "测试用户", "", "", 30, 0, "", 3, 10, 20, "/storage/pay.png"))
			mock.ExpectExec("UPDATE `yu_imgo_payment_method` SET receipt_name=.*scope_path").
				WithArgs("测试用户", int64(1), "导师已核对", sqlmock.AnyArg(), int64(12), typeName, int64(3), int64(7), int64(7)).
				WillReturnResult(sqlmock.NewResult(0, 1))
			result, err := a.manageBankCard(agentMemberRequest(a, "/manage/bank/edit", M{
				"user_id": 12, "payment_type": typeName, "version": 3, "status": 1, "receipt_name": "测试用户", "remark": "导师已核对",
			}))
			if err != nil || result.(M)["saved"] != true {
				t.Fatalf("mentor review failed: %#v %v", result, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
