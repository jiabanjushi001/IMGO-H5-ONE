package server

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func TestBankCardValidationAndEncryption(t *testing.T) {
	if _, ok := bankStatus("garbage"); ok {
		t.Fatal("non-numeric status accepted")
	}
	for _, account := range []string{"", "123", "1234abc567890", strings.Repeat("1", 31)} {
		if validBankAccount(account) {
			t.Fatalf("accepted invalid account %q", account)
		}
	}
	if !validBankAccount("6222020202020202") || !validBankName("测试用户") || validBankName("\n") {
		t.Fatal("bank input validation")
	}
	if !validBankInstitution("中国银行") || !validBankInstitution("北京朝阳支行") || validBankInstitution("A") || validBankInstitution(strings.Repeat("支", 121)) || validBankInstitution("支行\n名称") {
		t.Fatal("bank institution validation")
	}
	key := strings.Repeat("k", 32)
	ciphertext, err := encryptBankAccount(key, "6222020202020202")
	if err != nil || strings.Contains(ciphertext, "6222020202020202") {
		t.Fatalf("encryption failed: %v", err)
	}
	got, err := decryptBankAccount(key, ciphertext)
	if err != nil || got != "6222020202020202" {
		t.Fatalf("decryption failed: %v", err)
	}
	if _, err := decryptBankAccount(strings.Repeat("x", 32), ciphertext); err == nil {
		t.Fatal("different key decrypted bank account")
	}
}

func TestAdminBankEditUpdatesReviewedCard(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "bank_name", "branch_name", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(7, "测试用户", "旧银行", "旧支行", "0202", 0, "", 3, 10, 20))
	mock.ExpectExec("UPDATE `yu_imgo_bank_card` SET receipt_name").WithArgs("测试用户", "中国银行", "北京朝阳支行", int64(1), "已核对", sqlmock.AnyArg(), int64(7), int64(3)).WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.manageBankCard(bankRequest(a, "/manage/bank/edit", 1, M{"user_id": 7, "version": 3, "status": 1, "receipt_name": "测试用户", "bank_name": "中国银行", "branch_name": "北京朝阳支行", "remark": "已核对"}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("admin edit failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBankDetailReturnsFullAccountWithoutCipher(t *testing.T) {
	a, mock := testApp(t)
	const account = "6222020202020202"
	ciphertext, err := encryptBankAccount(a.cfg.JWTKey, account)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT user_id,receipt_name,bank_name,branch_name,account_cipher,account_last4,status,remark,version,created_at,updated_at FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "bank_name", "branch_name", "account_cipher", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(7, "测试用户", "中国银行", "北京朝阳支行", ciphertext, "0202", 0, "", 3, 10, 20))
	result, err := a.manageBankCard(bankRequest(a, "/manage/bank/detail", 1, M{"user_id": 7}))
	if err != nil {
		t.Fatal(err)
	}
	card := result.(M)
	if card["receipt_account"] != account || card["account_cipher"] != nil || card["account_last4"] != nil || card["bank_name"] != "中国银行" || card["branch_name"] != "北京朝阳支行" {
		t.Fatal("admin detail must contain only the decrypted account, not its ciphertext")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type encryptedBankAccountArg struct {
	key, account string
}

func (want encryptedBankAccountArg) Match(value driver.Value) bool {
	encrypted, ok := value.(string)
	if !ok || encrypted == want.account {
		return false
	}
	account, err := decryptBankAccount(want.key, encrypted)
	return err == nil && account == want.account
}

func TestAdminBankEditCanReplaceAccount(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(7, "测试用户", "0202", 0, "", 3, 10, 20))
	mock.ExpectExec("UPDATE `yu_imgo_bank_card` SET receipt_name=.*account_cipher=\\?,account_last4=\\? WHERE user_id=\\? AND version=\\?").
		WithArgs("测试用户", "", "", int64(1), "已核对", sqlmock.AnyArg(), encryptedBankAccountArg{a.cfg.JWTKey, "6222020202027878"}, "7878", int64(7), int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.manageBankCard(bankRequest(a, "/manage/bank/edit", 1, M{"user_id": 7, "version": 3, "status": 1, "receipt_name": "测试用户", "remark": "已核对", "receipt_account": "6222 0202 0202 7878"}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("admin should be able to replace an encrypted card number: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBankEditRejectsInvalidAccount(t *testing.T) {
	for _, account := range []string{"", "1234", "62220202abc20202"} {
		t.Run(account, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(7, "测试用户", "0202", 0, "", 3, 10, 20))
			_, err := a.manageBankCard(bankRequest(a, "/manage/bank/edit", 1, M{"user_id": 7, "version": 3, "status": 1, "receipt_name": "测试用户", "receipt_account": account}))
			if ce, ok := err.(clientError); !ok || ce.code != 400 {
				t.Fatalf("invalid card number should be rejected: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func bankRequest(a *App, path string, uid int64, params M) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", path, nil)
	return &request{app: a, c: c, user: M{"user_id": uid}, p: params}
}

func TestUserBankCardOnlyReturnsMaskedNumber(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "bank_name", "branch_name", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(7, "测试用户", "中国银行", "北京朝阳支行", "0202", 0, "", 1, 10, 20))
	result, err := a.bankCard(bankRequest(a, "/enterprise/bank/get", 7, M{}))
	if err != nil {
		t.Fatal(err)
	}
	card := result.(M)
	if card["receipt_account_masked"] != "•••• •••• •••• 0202" || card["account_cipher"] != nil || card["account_last4"] != nil || card["bank_name"] != "中国银行" || card["branch_name"] != "北京朝阳支行" {
		t.Fatalf("unexpected card response: %#v", card)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserSaveBankCardPendingAndOwnRecord(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}))
	mock.ExpectExec("INSERT INTO `yu_imgo_bank_card`").WithArgs(int64(7), "测试用户", "中国银行", "北京朝阳支行", sqlmock.AnyArg(), "0202", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "测试用户", "card_number": "6222020202020202", "bank_name": "中国银行", "branch_name": "北京朝阳支行"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUserBankSaveRequiresBankAndBranch(t *testing.T) {
	for _, fields := range []M{
		{"bank_name": "", "branch_name": "北京朝阳支行"},
		{"bank_name": "中国银行", "branch_name": ""},
		{"bank_name": "中国银行", "branch_name": "X"},
		{"bank_name": "中\n国银行", "branch_name": "北京朝阳支行"},
	} {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}))
		params := M{"name": "测试用户", "card_number": "6222020202020202", "bank_name": fields["bank_name"], "branch_name": fields["branch_name"]}
		_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, params))
		if ce, ok := err.(clientError); !ok || ce.code != 400 {
			t.Fatalf("invalid bank details should be rejected: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUserCannotResubmitUntilAdminRejects(t *testing.T) {
	for _, status := range []int64{0, 1} {
		t.Run(str(status), func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(status))
			_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "测试用户", "card_number": "6222020202020202"}))
			if ce, ok := err.(clientError); !ok || ce.code != 409 {
				t.Fatalf("status %d must reject resubmission, got %v", status, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUserBankSaveHTTPRejectsPendingAndApprovedRegardlessOfPayload(t *testing.T) {
	for _, status := range []int64{0, 1} {
		t.Run(str(status), func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WithArgs("session", int64(7), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
			mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE user_id").WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id", "role", "status"}).AddRow(7, 0, 1))
			mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(status))
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/enterprise/bank/save", strings.NewReader(`{"user_id":999,"status":2,"name":"X","card_number":"bad"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "bearer "+signToken(a.cfg.JWTKey, 7, "session", time.Now().Add(time.Hour)))
			a.Router().ServeHTTP(w, req)
			var body M
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || number(body["code"]) != 409 {
				t.Fatalf("status %d must block any self-service update: %s (%v)", status, w.Body.String(), err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRejectedBankCardCanBeResubmittedOnlyOnce(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(2))
	mock.ExpectExec("UPDATE `yu_imgo_bank_card` SET receipt_name=.* WHERE user_id=\\? AND status=2").WithArgs("新收款姓名", "建设银行", "上海浦东支行", sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "新收款姓名", "card_number": "", "bank_name": "建设银行", "branch_name": "上海浦东支行"}))
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(0))
	_, err = a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "再次修改", "card_number": ""}))
	if ce, ok := err.(clientError); !ok || ce.code != 409 {
		t.Fatalf("second submission should wait for another rejection: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedBankCardConcurrentReviewBlocksUpdate(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(2))
	mock.ExpectExec("UPDATE `yu_imgo_bank_card` SET receipt_name=.* WHERE user_id=\\? AND status=2").WithArgs("新收款姓名", "建设银行", "上海浦东支行", sqlmock.AnyArg(), int64(7)).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "新收款姓名", "card_number": "", "bank_name": "建设银行", "branch_name": "上海浦东支行"}))
	if ce, ok := err.(clientError); !ok || ce.code != 409 {
		t.Fatalf("stale rejection must not overwrite admin review: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedBankCardCanReplaceNumber(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(2))
	mock.ExpectExec("UPDATE `yu_imgo_bank_card` SET receipt_name=.*account_cipher=\\?,account_last4=\\? WHERE user_id=\\? AND status=2").WithArgs("新收款姓名", "建设银行", "上海浦东支行", sqlmock.AnyArg(), sqlmock.AnyArg(), "7878", int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "新收款姓名", "card_number": "6222020202027878", "bank_name": "建设银行", "branch_name": "上海浦东支行"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFirstSubmissionRaceDoesNotOverwriteExistingCard(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT status FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"status"}))
	mock.ExpectExec("INSERT INTO `yu_imgo_bank_card`").WithArgs(int64(7), "测试用户", "中国银行", "北京朝阳支行", sqlmock.AnyArg(), "0202", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate"})
	_, err := a.bankCard(bankRequest(a, "/enterprise/bank/save", 7, M{"name": "测试用户", "card_number": "6222020202020202", "bank_name": "中国银行", "branch_name": "北京朝阳支行"}))
	if ce, ok := err.(clientError); !ok || ce.code != 409 {
		t.Fatalf("concurrent first submission should return conflict: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminBankEditRejectsStaleVersion(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT user_id,receipt_name,bank_name,branch_name,account_last4,status,remark,version,created_at,updated_at FROM `yu_imgo_bank_card` WHERE user_id").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(7, "测试用户", "0202", 0, "", 3, 10, 20))
	_, err := a.manageBankCard(bankRequest(a, "/manage/bank/edit", 1, M{"user_id": 7, "version": 2, "status": 1, "receipt_name": "测试用户"}))
	if ce, ok := err.(clientError); !ok || ce.code != 409 {
		t.Fatalf("expected version conflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBankCardRoutesRequireAuthentication(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/enterprise/bank/get", "/enterprise/bank/save", "/manage/bank/index", "/manage/bank/detail", "/manage/bank/edit"} {
		w := httptest.NewRecorder()
		a.Router().ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader("{}")))
		var body M
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || number(body["code"]) != -1 {
			t.Fatalf("%s: %s (%v)", path, w.Body.String(), err)
		}
	}
}

func TestOrdinaryUserCannotManageBankCards(t *testing.T) {
	a, mock := testApp(t)
	for _, path := range []string{"/manage/bank/index", "/manage/bank/detail", "/manage/bank/edit"} {
		mock.ExpectQuery("SELECT user_id FROM .*imgo_session").WithArgs("session", int64(7), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
		mock.ExpectQuery("SELECT \\* FROM .*user.* WHERE user_id").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "role", "status"}).AddRow(7, 0, 1))
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "bearer "+signToken(a.cfg.JWTKey, 7, "session", time.Now().Add(time.Hour)))
		a.Router().ServeHTTP(w, req)
		var body M
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || number(body["code"]) != 403 {
			t.Fatalf("%s: %s (%v)", path, w.Body.String(), err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
