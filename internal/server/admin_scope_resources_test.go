package server

import (
	"database/sql/driver"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectResourceUser(mock sqlmock.Sqlmock, uid int64, allowed bool) {
	rows := sqlmock.NewRows([]string{"1"})
	if allowed {
		rows.AddRow(1)
	}
	mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=.*scope_path.ancestor_user_id=.*scope_path.descendant_user_id=u.user_id").WithArgs(uid, int64(7), int64(7)).WillReturnRows(rows)
}

func TestAgentScopeResourcesRejectForeignUsersBeforeReading(t *testing.T) {
	for _, tc := range []struct {
		path string
		call func(*App, *request) (any, error)
	}{
		{"/manage/bank/detail", (*App).manageBankCard}, {"/manage/bank/edit", (*App).manageBankCard},
		{"/manage/wallet/account", (*App).manageWallet}, {"/manage/wallet/entries", (*App).manageWallet},
		{"/manage/wallet/credit", (*App).manageWallet}, {"/manage/wallet/recharge", (*App).manageWallet}, {"/manage/wallet/withdraw", (*App).manageWallet},
		{"/manage/message/getContacts", (*App).manageMessage},
	} {
		t.Run(tc.path, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			expectResourceUser(mock, 99, false)
			_, err := tc.call(a, agentMemberRequest(a, tc.path, M{"user_id": 99, "amount": "12.00", "request_id": "request-12345678", "note": "测试入账", "bonus_mode": "none"}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeResourcesGlobalMutationsDenied(t *testing.T) {
	for _, act := range []string{"publishNotice", "delNotice", "clearMessage", "setTaskConfig", "startTask", "stopTask", "clearTaskLog"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			r := agentMemberRequest(a, "/manage/index/"+act, M{})
			var err error
			if act == "publishNotice" || act == "delNotice" || act == "clearMessage" {
				_, err = a.manageIndex(r)
			} else {
				_, err = a.task(r)
			}
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeResourcesListsConstrainCountsAndRows(t *testing.T) {
	for _, tc := range []struct {
		path, table, scope string
		call               func(*App, *request) (any, error)
		args               []driver.Value
	}{
		{"/manage/group/index", "group", "scope_owner.user_id=g.owner_id", (*App).manageGroup, []driver.Value{int64(7), int64(7)}},
		{"/manage/message/index", "message", "scope_group.group_id=.*to_user", (*App).manageMessage, []driver.Value{int64(7), int64(7), int64(7), int64(7), int64(7), int64(7)}},
		{"/manage/files/index", "file", "scope_path.descendant_user_id=f.user_id", (*App).files, []driver.Value{int64(7), int64(7)}},
		{"/manage/bank/index", "imgo_bank_card", "scope_path.descendant_user_id=b.user_id", (*App).manageBankCard, []driver.Value{int64(7), int64(7)}},
		{"/manage/wallet/index", "imgo_withdrawal", "scope_path.descendant_user_id=w.user_id", (*App).manageWallet, []driver.Value{int64(7), int64(7)}},
		{"/manage/wallet/recharges", "imgo_recharge_order", "scope_path.descendant_user_id=o.user_id", (*App).manageWallet, []driver.Value{int64(7), int64(7)}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM `yu_" + tc.table + "`.*" + tc.scope).WithArgs(tc.args...).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
			mock.ExpectQuery("SELECT .*FROM `yu_" + tc.table + "`.*" + tc.scope + ".*LIMIT \\? OFFSET \\?").WithArgs(append(tc.args, int64(20), int64(0))...).WillReturnRows(sqlmock.NewRows([]string{"id"}))
			_, err := tc.call(a, agentMemberRequest(a, tc.path, M{}))
			if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeGroupRejectsForeignOwnerAndTargets(t *testing.T) {
	for _, act := range []string{"changeOwner", "del", "addGroupUser", "delGroupUser", "setManager"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT owner_id FROM `yu_group` WHERE group_id=").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(99))
			expectResourceUser(mock, 99, false)
			_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/"+act, M{"group_id": 9, "user_id": 12, "user_ids": []any{12}, "role": 2}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, act := range []string{"changeOwner", "addGroupUser", "delGroupUser", "setManager"} {
		t.Run(act+" foreign target", func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT owner_id FROM `yu_group` WHERE group_id=").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
			expectResourceUser(mock, 12, true)
			expectResourceUser(mock, 99, false)
			_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/"+act, M{"group_id": 9, "user_id": 99, "user_ids": []any{99}, "role": 2}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeMessageDealReadsOnlyScopedMessage(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT \\* FROM `yu_message` WHERE id=.*scope_group.group_id=.*to_user").WithArgs("foreign-message", int64(7), int64(7), int64(7), int64(7), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"msg_id"}))
	_, err := a.manageMessage(agentMemberRequest(a, "/manage/message/dealMsg", M{"id": "foreign-message"}))
	assertDenied(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeWalletReviewRechecksOwnerAfterLock(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id FROM `yu_imgo_withdrawal` WHERE withdrawal_id=?")).WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	expectResourceUser(mock, 12, true)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal` WHERE withdrawal_id=.*FOR UPDATE").WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(99, 100, 1))
	expectResourceUser(mock, 99, false)
	mock.ExpectRollback()
	_, err := a.manageWallet(agentMemberRequest(a, "/manage/wallet/review", M{"withdrawal_id": 51, "status": 1}))
	assertDenied(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeWalletWritesRecheckAfterWalletLock(t *testing.T) {
	for _, act := range []string{"credit", "recharge", "withdraw"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			expectResourceUser(mock, 12, true)
			if act == "credit" {
				mock.ExpectQuery("SELECT entry_id,available_delta FROM `yu_imgo_wallet_entry`").WillReturnRows(sqlmock.NewRows([]string{"entry_id"}))
				mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
			}
			mock.ExpectBegin()
			if act != "credit" {
				mock.ExpectQuery("SELECT user_id FROM `yu_user` WHERE user_id=.*FOR SHARE").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
			}
			mock.ExpectExec("INSERT IGNORE INTO `yu_imgo_wallet`").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT available_cents.*FROM `yu_imgo_wallet` WHERE user_id=.*FOR UPDATE").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(100000, 0))
			expectResourceUser(mock, 12, false)
			mock.ExpectRollback()
			_, err := a.manageWallet(agentMemberRequest(a, "/manage/wallet/"+act, M{"user_id": 12, "amount": "12.00", "request_id": "request-12345678", "note": "测试入账", "bonus_mode": "none"}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeGroupSharedAdminReadsAndAvatarRejectForeignOwner(t *testing.T) {
	for _, act := range []string{"groupInfo", "groupUserList", "editGroupAvatar"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT \\* FROM `yu_group` WHERE group_id=").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "status"}).AddRow(9, 99, 1))
			expectAgentMemberScope(mock)
			expectResourceUser(mock, 99, false)
			_, err := a.group(agentMemberRequest(a, "/enterprise/group/"+act, M{"group_id": 9}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeMessageContactsConstrainGroupAndLatestQueries(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	expectResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE user_id=").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	mock.ExpectQuery("SELECT value FROM `yu_config`").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{}`))
	mock.ExpectQuery("SELECT u.user_id,u.realname.*FROM `yu_user`").WithArgs(int64(12), int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectQuery("SELECT from_user,COUNT").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"from_user", "n"}))
	mock.ExpectQuery("SELECT g.*scope_owner.user_id=g.owner_id.*scope_path").WithArgs(int64(12), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"group_id"}))
	mock.ExpectQuery("SELECT m.*scope_group.group_id=.*to_user.*scope_path").WithArgs(int64(12), int64(12), int64(12), int64(12), int64(12), int64(7), int64(7), int64(7), int64(7), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"msg_id"}))
	mock.ExpectQuery("SELECT COUNT.*FROM `yu_friend`").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	_, err := a.manageMessage(agentMemberRequest(a, "/manage/message/getContacts", M{"user_id": 12}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeGroupChangeOwnerRechecksLockedOwner(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
	expectResourceUser(mock, 12, true)
	expectResourceUser(mock, 13, true)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `yu_group` WHERE group_id=.*FOR UPDATE").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(9, 99))
	expectResourceUser(mock, 99, false)
	mock.ExpectRollback()
	_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/changeOwner", M{"group_id": 9, "user_id": 13}))
	assertDenied(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeFileDownloadAndPreviewMatrix(t *testing.T) {
	for _, preview := range []bool{false, true} {
		for _, tc := range []struct {
			name      string
			uid, mode int64
			allowed   bool
		}{{"descendant", 7, 1, true}, {"foreign", 7, 1, false}, {"super", 1, 0, true}, {"global role", 7, 0, true}} {
			t.Run(tc.name+fmt.Sprint(preview), func(t *testing.T) {
				a, mock := testApp(t)
				if err := os.MkdirAll(filepath.Join(a.cfg.PublicDir, "storage/file"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(a.cfg.PublicDir, "storage/file/test.txt"), []byte("file-content"), 0600); err != nil {
					t.Fatal(err)
				}
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				path := "/filedown/" + a.hashID(5)
				if preview {
					path = "/storage/file/test.txt"
				}
				c.Request = httptest.NewRequest("GET", path, nil)
				c.Request.Header.Set("Authorization", signToken(a.cfg.JWTKey, tc.uid, "scope-file", time.Now().Add(time.Hour)))
				mock.ExpectQuery("SELECT \\* FROM `yu_file`").WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "src", "name", "ext", "disk", "parent_id"}).AddRow(5, 12, "/storage/file/test.txt", "test", "txt", "local", 0))
				if preview {
					mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("sysInfo").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{}`))
				}
				mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WithArgs("scope-file", tc.uid, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(tc.uid))
				mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(tc.uid).WillReturnRows(sqlmock.NewRows([]string{"user_id", "admin_role_id"}).AddRow(tc.uid, 3))
				if tc.uid != 1 {
					mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(tc.mode))
					if tc.mode == 1 {
						expectResourceUser(mock, 12, tc.allowed)
					}
				}
				if tc.allowed && tc.uid != 1 {
					mock.ExpectQuery("SELECT id FROM `yu_emoji`").WillReturnRows(sqlmock.NewRows([]string{"id"}))
					mock.ExpectQuery("SELECT m.msg_id FROM `yu_message`").WillReturnRows(sqlmock.NewRows([]string{"msg_id"}).AddRow(2))
				}
				if tc.allowed {
					mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("fileUpload").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"disk":"local"}`))
					mock.ExpectQuery("SELECT disk,object_key FROM `yu_imgo_object`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"disk", "object_key"}))
				}
				if preview {
					if a.authorizeStorage(c, "storage/file/test.txt") {
						c.Status(200)
					}
				} else {
					a.download(c)
				}
				c.Writer.WriteHeaderNow()
				want := 200
				if !tc.allowed {
					want = 403
				}
				if recorder.Code != want {
					t.Fatalf("status=%d want %d", recorder.Code, want)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestAgentScopeGroupMemberWritesIncludeOwnerAndTargetPredicates(t *testing.T) {
	for _, act := range []string{"setManager", "delGroupUser"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
			expectResourceUser(mock, 12, true)
			expectResourceUser(mock, 13, true)
			if act == "delGroupUser" {
				mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
			}
			query := "UPDATE `yu_group_user`.*scope_path.*scope_path"
			args := []driver.Value{int64(2), int64(9), int64(13), int64(7), int64(7), int64(7), int64(7)}
			if act == "delGroupUser" {
				query = "DELETE FROM `yu_group_user`.*scope_path.*scope_path"
				args = args[1:]
			}
			mock.ExpectExec(query).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))
			_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/"+act, M{"group_id": 9, "user_id": 13, "role": 2}))
			if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeGroupAddAndDeleteRecheckAfterGroupLock(t *testing.T) {
	for _, act := range []string{"addGroupUser", "del"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
			expectResourceUser(mock, 12, true)
			if act == "addGroupUser" {
				expectResourceUser(mock, 13, true)
				mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(9, 12))
			}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT .* FROM `yu_group` WHERE group_id=.*FOR UPDATE").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(9, 99))
			expectResourceUser(mock, 99, false)
			mock.ExpectRollback()
			_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/"+act, M{"group_id": 9, "user_ids": []any{13}}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeBankEditGuardsUpdate(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	expectResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT user_id,receipt_name.*FROM `yu_imgo_bank_card`").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "receipt_name", "bank_name", "branch_name", "account_last4", "status", "remark", "version", "created_at", "updated_at"}).AddRow(12, "测试用户", "中国银行", "测试支行", "1234", 0, "", 3, 10, 20))
	mock.ExpectExec("UPDATE `yu_imgo_bank_card` SET receipt_name=.*WHERE user_id=.*scope_path").WithArgs("测试用户", "中国银行", "测试支行", int64(1), "已核对", sqlmock.AnyArg(), int64(12), int64(3), int64(7), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	result, err := a.manageBankCard(agentMemberRequest(a, "/manage/bank/edit", M{"user_id": 12, "receipt_name": "测试用户", "status": 1, "version": 3, "remark": "已核对"}))
	if err != nil || result.(M)["saved"] != true {
		t.Fatalf("result=%v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeResourcesGlobalRolesKeepUnrestrictedLists(t *testing.T) {
	for _, uid := range []int64{1, 7} {
		for _, tc := range []struct {
			path, table string
			call        func(*App, *request) (any, error)
		}{
			{"/manage/group/index", "group", (*App).manageGroup}, {"/manage/message/index", "message", (*App).manageMessage}, {"/manage/files/index", "file", (*App).files}, {"/manage/bank/index", "imgo_bank_card", (*App).manageBankCard}, {"/manage/wallet/index", "imgo_withdrawal", (*App).manageWallet}, {"/manage/wallet/recharges", "imgo_recharge_order", (*App).manageWallet},
		} {
			t.Run(fmt.Sprint(uid)+tc.path, func(t *testing.T) {
				a, mock := testApp(t)
				if uid != 1 {
					mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(0))
				}
				mock.ExpectQuery("SELECT COUNT.*FROM `yu_" + tc.table + "`").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
				mock.ExpectQuery("SELECT .*FROM `yu_"+tc.table+"`.*LIMIT \\? OFFSET \\?").WithArgs(int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
				r := bankRequest(a, tc.path, uid, M{})
				r.user["admin_role_id"] = int64(3)
				_, err := tc.call(a, r)
				if err != nil {
					t.Fatal(err)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestAgentScopeWalletAllowsDescendantCreditRetry(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	expectResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT entry_id,available_delta FROM `yu_imgo_wallet_entry`").WithArgs(int64(12), "request-12345678").WillReturnRows(sqlmock.NewRows([]string{"entry_id", "available_delta"}).AddRow(51, 1200))
	result, err := a.manageWallet(agentMemberRequest(a, "/manage/wallet/credit", M{"user_id": 12, "amount": "12.00", "request_id": "request-12345678", "note": "测试入账"}))
	if err != nil || number(result.(M)["entry_id"]) != 51 {
		t.Fatalf("result=%v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeMessageAllowsDescendantAndScopesUpdate(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	pattern := ".*is_group=0.*scope_from.user_id=`yu_message`.from_user.* OR EXISTS.*scope_to.user_id=`yu_message`.to_user.*is_group=1.*scope_group.group_id=`yu_message`.to_user"
	args := []driver.Value{"scoped-message", int64(7), int64(7), int64(7), int64(7), int64(7), int64(7)}
	mock.ExpectQuery("SELECT \\* FROM `yu_message` WHERE id=" + pattern).WithArgs(args...).WillReturnRows(sqlmock.NewRows([]string{"id", "msg_id", "is_group", "from_user", "to_user"}).AddRow("scoped-message", 42, 0, 12, 99))
	mock.ExpectExec("UPDATE `yu_message` SET `status`=\\? WHERE id=" + pattern).WithArgs(append([]driver.Value{int64(0)}, args...)...).WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := a.manageMessage(agentMemberRequest(a, "/manage/message/dealMsg", M{"id": "scoped-message", "dealType": 1}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeWalletDetailRejectsOwnerChangedAfterPrecheck(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_withdrawal`").WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	expectResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT w.withdrawal_id.*WHERE w.withdrawal_id=.*scope_path.descendant_user_id=w.user_id").WithArgs(int64(51), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"withdrawal_id"}))
	_, err := a.manageWallet(agentMemberRequest(a, "/manage/wallet/detail", M{"withdrawal_id": 51}))
	assertDenied(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeGroupCandidateListContainsOnlyDescendants(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
	expectResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT user_id,realname,avatar,name_py FROM `yu_user`.*scope_path.descendant_user_id=u.user_id").WithArgs(int64(7), int64(9), int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "realname"}).AddRow(13, "下级"))
	result, err := a.group(agentMemberRequest(a, "/enterprise/group/getAllUser", M{"group_id": 9}))
	if err != nil || len(result.([]M)) != 1 {
		t.Fatalf("result=%v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeGroupAvatarRejectsForeignOwner(t *testing.T) {
	a, mock := testApp(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/avatar/group-9/120/9", nil)
	c.Request.Header.Set("Authorization", signToken(a.cfg.JWTKey, 7, "scope-avatar", time.Now().Add(time.Hour)))
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WithArgs("scope-avatar", int64(7), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "admin_role_id"}).AddRow(7, 3))
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(99))
	expectResourceUser(mock, 99, false)
	a.groupAvatar(c, 9)
	c.Writer.WriteHeaderNow()
	if recorder.Code != 403 {
		t.Fatalf("status=%d", recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeGroupAvatarUpdateHasOwnerPredicate(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "status"}).AddRow(9, 12, 1))
	expectAgentMemberScope(mock)
	expectResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow("manage.groups"))
	mock.ExpectQuery("SELECT gu.*").WithArgs(int64(9), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"role"}))
	mock.ExpectQuery("SELECT \\* FROM `yu_file`").WithArgs(int64(5), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "size", "src"}).AddRow(5, 7, 2, 100, "/storage/image/avatar.png"))
	mock.ExpectExec("UPDATE `yu_group` SET `avatar`=.*scope_owner.user_id=`yu_group`.owner_id.*scope_path").WithArgs("/storage/image/avatar.png", int64(9), int64(7), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	_, err := a.group(agentMemberRequest(a, "/enterprise/group/editGroupAvatar", M{"group_id": 9, "file_id": 5}))
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeFileLegacyPreviewPreservesForbiddenStatus(t *testing.T) {
	a, mock := testApp(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/view", nil)
	c.Request.Header.Set("Authorization", signToken(a.cfg.JWTKey, 7, "scope-view", time.Now().Add(time.Hour)))
	mock.ExpectQuery("SELECT \\* FROM `yu_file`").WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "src", "cate"}).AddRow(5, 99, "/storage/file/test.txt", 1))
	mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("sysInfo").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{}`))
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "admin_role_id"}).AddRow(7, 3))
	expectAgentMemberScope(mock)
	expectResourceUser(mock, 99, false)
	_, err := a.index(&request{app: a, c: c, p: M{"src": "/storage/file/test.txt"}})
	if err != nil {
		t.Fatalf("legacy preview replaced scope denial: %v", err)
	}
	if recorder.Code != 403 {
		t.Fatalf("status=%d", recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
