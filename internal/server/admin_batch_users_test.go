package server

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"unicode"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBatchUserInputValidation(t *testing.T) {
	valid, err := batchUserInputFromParams(M{
		"count": 12, "account_prefix": "vip_", "password": "123456",
		"parent_invite_code": "123456", "customer_user_id": 9,
	})
	if err != nil || valid.Count != 12 || valid.Prefix != "vip_" || valid.CustomerID != 9 {
		t.Fatalf("input=%#v err=%v", valid, err)
	}
	for _, params := range []M{
		{"count": 0, "password": "123456"},
		{"count": 201, "password": "123456"},
		{"count": 1, "password": "123"},
		{"count": 1, "password": "123456", "account_prefix": "bad prefix"},
		{"count": 1, "password": "123456", "account_prefix": "abcdefghijklmnopqrstu"},
	} {
		if _, err := batchUserInputFromParams(params); err == nil {
			t.Fatalf("expected invalid input: %#v", params)
		}
	}
}

func TestBatchRandomIdentityShape(t *testing.T) {
	accounts := map[string]bool{}
	accountPattern := regexp.MustCompile(`^u[a-z2-9]{8}$`)
	for i := 0; i < 100; i++ {
		account, err := randomBatchAccount("")
		if err != nil || !accountPattern.MatchString(account) || accounts[account] {
			t.Fatalf("account=%q duplicate=%v err=%v", account, accounts[account], err)
		}
		accounts[account] = true
		name, err := randomChineseName()
		if err != nil {
			t.Fatal(err)
		}
		runes := []rune(name)
		if len(runes) < 2 || len(runes) > 4 {
			t.Fatalf("name length=%d name=%q", len(runes), name)
		}
		for _, ch := range runes {
			if !unicode.Is(unicode.Han, ch) {
				t.Fatalf("non-Chinese generated name %q", name)
			}
		}
		sex, err := randomBatchSex()
		if err != nil || sex < 0 || sex > 1 {
			t.Fatalf("sex=%d err=%v", sex, err)
		}
	}
}

func TestSequentialBatchAccount(t *testing.T) {
	for _, test := range []struct {
		sequence int
		want     string
	}{
		{1, "wuhu001"},
		{2, "wuhu002"},
		{999, "wuhu999"},
		{1000, "wuhu1000"},
	} {
		if got := sequentialBatchAccount("wuhu", test.sequence); got != test.want {
			t.Fatalf("sequence=%d got=%q want=%q", test.sequence, got, test.want)
		}
	}
}

func TestNextBatchAccountNumberContinuesExistingPrefix(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT account FROM `yu_user` WHERE LEFT\\(account,\\?\\)=\\?").
		WithArgs(4, "wuhu").
		WillReturnRows(sqlmock.NewRows([]string{"account"}).
			AddRow("wuhu001").
			AddRow("WUHU009").
			AddRow("wuhu11").
			AddRow("wuhunot-a-number").
			AddRow("wuhu008extra"))
	next, err := a.nextBatchAccountNumber(context.Background(), a.db, "wuhu")
	if err != nil || next != 10 {
		t.Fatalf("next=%d err=%v", next, err)
	}
}

func TestBatchMemberInviterScope(t *testing.T) {
	t.Run("mentor requires an invite code", func(t *testing.T) {
		a, _ := testApp(t)
		_, err := a.batchMemberInviter(agentMemberRequest(a, "/manage/user/batchAdd", M{}), a.db, adminScope{AgentUserID: 7, referralTable: "`yu_imgo_referral_path`"})
		if err == nil || err.Error() != "导师批量创建账号必须填写上级邀请码" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("mentor may use own invite code", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT r.user_id FROM `yu_imgo_referral`").WithArgs("123456").
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
		mock.ExpectQuery("SELECT user_id FROM `yu_user`.*status=1 AND delete_time=0").WithArgs(int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
		uid, err := a.batchMemberInviter(agentMemberRequest(a, "/manage/user/batchAdd", M{"parent_invite_code": "123456"}), a.db, adminScope{AgentUserID: 7, referralTable: "`yu_imgo_referral_path`"})
		if err != nil || uid != 7 {
			t.Fatalf("uid=%d err=%v", uid, err)
		}
	})

	t.Run("mentor may use a team invite code", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT r.user_id FROM `yu_imgo_referral`").WithArgs("654321").
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
		mock.ExpectQuery("SELECT user_id FROM `yu_user`.*status=1 AND delete_time=0").WithArgs(int64(12)).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
		mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").WithArgs(int64(12), int64(7), int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
		uid, err := a.batchMemberInviter(agentMemberRequest(a, "/manage/user/batchAdd", M{"parent_invite_code": "654321"}), a.db, adminScope{AgentUserID: 7, referralTable: "`yu_imgo_referral_path`"})
		if err != nil || uid != 12 {
			t.Fatalf("uid=%d err=%v", uid, err)
		}
	})

	t.Run("mentor rejects an outside invite code", func(t *testing.T) {
		a, mock := testApp(t)
		mock.ExpectQuery("SELECT r.user_id FROM `yu_imgo_referral`").WithArgs("654321").
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(99))
		mock.ExpectQuery("SELECT user_id FROM `yu_user`.*status=1 AND delete_time=0").WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(99))
		mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=").WithArgs(int64(99), int64(7), int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"1"}))
		_, err := a.batchMemberInviter(agentMemberRequest(a, "/manage/user/batchAdd", M{"parent_invite_code": "654321"}), a.db, adminScope{AgentUserID: 7, referralTable: "`yu_imgo_referral_path`"})
		if err == nil || err.Error() != "上级邀请码只能选择本人或自己团队的成员" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestBatchAddRouteUsesMemberPermission(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/manage/user/batchadd", "/manage/user/batchstatus"} {
		route, ok := a.routes[path]
		if !ok || route.permission != "manage.users" || route.super {
			t.Fatalf("path=%s route=%#v ok=%v", path, route, ok)
		}
	}
}

func TestBatchTaskCodecKeepsPasswordEncryptedAtRest(t *testing.T) {
	a := &App{cfg: Config{ChatKey: "1234567890123456"}}
	want := batchTaskPayload{
		Input: batchUserInput{Count: 3, Prefix: "vip_", Password: "secret123", InviteCode: "123456"},
		Scope: adminScope{Global: false, AgentUserID: 7},
		IP:    "198.51.100.8",
	}
	ciphertext, err := a.encodeBatchTaskPayload(want)
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "" || regexp.MustCompile(`secret123`).MatchString(ciphertext) {
		t.Fatalf("password leaked in task payload: %q", ciphertext)
	}
	got, err := a.decodeBatchTaskPayload(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
}
