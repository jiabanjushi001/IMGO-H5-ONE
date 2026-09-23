package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func dispatchScopeRequest(a *App, path, body string, uid int64) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", signToken(a.cfg.JWTKey, uid, "scope-dispatch", time.Now().Add(time.Hour)))
	a.dispatch(c)
	return rec
}
func expectScopeDispatchAuth(mock sqlmock.Sqlmock, permission string, uid int64) {
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WithArgs("scope-dispatch", uid, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(uid))
	mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(uid).WillReturnRows(sqlmock.NewRows([]string{"user_id", "admin_role_id"}).AddRow(uid, 3))
	if uid != 1 {
		mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow(permission))
	}
}
func responseScopeCode(t *testing.T, rec *httptest.ResponseRecorder) int64 {
	t.Helper()
	var body M
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return number(body["code"])
}

func TestAgentScopeFilePHPDispatchUsesManagementScope(t *testing.T) {
	for _, mode := range []int{0, 1} {
		t.Run(string(rune('0'+mode)), func(t *testing.T) {
			a, mock := testApp(t)
			expectScopeDispatchAuth(mock, "manage.files", 7)
			mock.ExpectQuery("SELECT agent_mode").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(mode))
			if mode == 1 {
				mock.ExpectQuery("SELECT COUNT.*FROM `yu_file` f WHERE .*scope_path.descendant_user_id=f.user_id").WithArgs(int64(7), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
				mock.ExpectQuery("SELECT f.*FROM `yu_file` f WHERE .*scope_path.descendant_user_id=f.user_id").WithArgs(int64(7), int64(7), int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "name", "ext", "src"}).AddRow(5, 12, "下级文件", "txt", "/storage/file/a.txt"))
			} else {
				mock.ExpectQuery("SELECT COUNT.*FROM `yu_file` f WHERE f.status=1 AND COALESCE\\(f.delete_time,0\\)=0$").WithoutArgs().WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
				mock.ExpectQuery("SELECT f.*FROM `yu_file` f WHERE f.status=1 AND COALESCE\\(f.delete_time,0\\)=0 ORDER BY").WithArgs(int64(20), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "name", "ext", "src"}).AddRow(5, 99, "全局文件", "txt", "/storage/file/a.txt"))
			}
			rec := dispatchScopeRequest(a, "/index.php?s=/manage/files/index", "{}", 7)
			if code := responseScopeCode(t, rec); code != 0 {
				t.Fatalf("response=%s", rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeResourcesSettingsPermissionDoesNotPermitGlobalWrites(t *testing.T) {
	for _, act := range []string{"setConfig", "sendTestEmail", "getInviteLink"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			expectScopeDispatchAuth(mock, "manage.settings", 7)
			expectAgentMemberScope(mock)
			rec := dispatchScopeRequest(a, "/index.php?s=/manage/config/"+act, `{"name":"chatInfo","value":{"groupChat":0},"email":"scope-test@example.com"}`, 7)
			if code := responseScopeCode(t, rec); code != 403 {
				t.Fatalf("response=%s", rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeGroupSharedMemberIdentityDoesNotRequireAdminMenu(t *testing.T) {
	for _, act := range []string{"groupInfo", "groupUserList", "editGroupAvatar"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "status"}).AddRow(9, 99, 1))
			mock.ExpectQuery("SELECT gu.*").WithArgs(int64(9), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"role", "owner_id"}).AddRow(2, 99))
			switch act {
			case "groupInfo":
				mock.ExpectQuery("SELECT user_id,realname,avatar FROM `yu_user`").WithArgs(int64(99)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "realname"}).AddRow(99, "群主"))
				mock.ExpectQuery("SELECT COUNT.*FROM `yu_group_user`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
			case "groupUserList":
				mock.ExpectQuery("SELECT COUNT.*FROM `yu_group_user`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
				mock.ExpectQuery("SELECT \\* FROM `yu_group_user`").WithArgs(int64(9), int64(2000), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
			case "editGroupAvatar":
				mock.ExpectQuery("SELECT \\* FROM `yu_file`").WithArgs(int64(5), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "size", "src"}).AddRow(5, 7, 2, 100, "/storage/image/avatar.png"))
				mock.ExpectExec("UPDATE `yu_group` SET `avatar`").WithArgs("/storage/image/avatar.png", int64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
			}
			_, err := a.group(agentMemberRequest(a, "/enterprise/group/"+act, M{"group_id": 9, "file_id": 5}))
			if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeGroupAvatarAllowsMemberWithoutAdminMenu(t *testing.T) {
	a, mock := testApp(t)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/avatar/group-9/120/9", nil)
	c.Request.Header.Set("Authorization", signToken(a.cfg.JWTKey, 7, "member-avatar", time.Now().Add(time.Hour)))
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "admin_role_id"}).AddRow(7, 3))
	mock.ExpectQuery("SELECT gu.*").WithArgs(int64(9), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(3))
	mock.ExpectQuery("SELECT avatar FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"avatar"}).AddRow(""))
	mock.ExpectQuery("SELECT u.user_id,u.realname,u.avatar FROM `yu_user`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "realname", "avatar"}).AddRow(7, "用户", ""))
	a.groupAvatar(c, 9)
	c.Writer.WriteHeaderNow()
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("response status=%d type=%s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeGroupMissingSharedResourceUsesDenied(t *testing.T) {
	for _, act := range []string{"groupInfo", "groupUserList", "editGroupAvatar"} {
		t.Run(act, func(t *testing.T) {
			a, mock := testApp(t)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id"}))
			expectAgentMemberScope(mock)
			_, err := a.group(agentMemberRequest(a, "/enterprise/group/"+act, M{"group_id": 9}))
			assertDenied(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func expectLockedResourceUser(mock sqlmock.Sqlmock, uid int64, allowed bool) {
	rows := sqlmock.NewRows([]string{"1"})
	if allowed {
		rows.AddRow(1)
	}
	mock.ExpectQuery("SELECT 1 FROM `yu_user` u WHERE u.user_id=.*scope_path.*FOR SHARE.*FOR SHARE$").WithArgs(uid, int64(7), int64(7)).WillReturnRows(rows)
}

func TestAgentScopeWalletReviewLocksAuthorizationBeforeWalletWait(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_withdrawal`").WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	expectResourceUser(mock, 12, true)
	mock.ExpectBegin()
	expectLockedResourceUser(mock, 12, true)
	mock.ExpectQuery("SELECT user_id,amount_cents,status FROM `yu_imgo_withdrawal`.*FOR UPDATE").WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount_cents", "status"}).AddRow(12, 100, 0))
	mock.ExpectQuery("SELECT available_cents,pending_cents FROM `yu_imgo_wallet`.*FOR UPDATE").WithArgs(int64(12)).WillDelayFor(10 * time.Millisecond).WillReturnRows(sqlmock.NewRows([]string{"available_cents", "pending_cents"}).AddRow(100, 100))
	expectLockedResourceUser(mock, 12, true)
	mock.ExpectExec("UPDATE `yu_imgo_wallet`").WithArgs(int64(0), int64(100), sqlmock.AnyArg(), int64(12), int64(100)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `yu_imgo_withdrawal`").WithArgs(int64(1), sqlmock.AnyArg(), int64(7), "已转账", int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_wallet_entry`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	result, err := a.manageWallet(agentMemberRequest(a, "/manage/wallet/review", M{"withdrawal_id": 51, "status": 1, "remark": "已转账"}))
	if err != nil || result.(M)["processed"] != true {
		t.Fatalf("result=%v error=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentScopeWalletReviewRejectsDeletionAfterPrecheck(t *testing.T) {
	a, mock := testApp(t)
	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_withdrawal`").WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(12))
	expectResourceUser(mock, 12, true)
	mock.ExpectBegin()
	expectLockedResourceUser(mock, 12, false)
	mock.ExpectRollback()
	_, err := a.manageWallet(agentMemberRequest(a, "/manage/wallet/review", M{"withdrawal_id": 51, "status": 1}))
	assertDenied(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func resourceEventObserver(t *testing.T, a *App, uid int64) <-chan []byte {
	t.Helper()
	p := &peer{id: "scope-event", uid: uid, claims: claims{Exp: time.Now().Add(time.Hour).Unix()}, send: make(chan []byte, 8), done: make(chan struct{})}
	a.hub.peers[p.id] = p
	t.Cleanup(func() { a.hub.mu.Lock(); delete(a.hub.peers, p.id); a.hub.mu.Unlock() })
	return p.send
}
func assertNoResourceEvent(t *testing.T, events <-chan []byte) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected event: %s", event)
	default:
	}
}

func TestAgentScopeMessageZeroRowsRechecksCurrentScopeWithoutEvent(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			a, mock := testApp(t)
			events := resourceEventObserver(t, a, 99)
			expectAgentMemberScope(mock)
			mock.ExpectQuery("SELECT \\* FROM `yu_message` WHERE id=.*scope_path").WillReturnRows(sqlmock.NewRows([]string{"id", "msg_id", "is_group", "from_user", "to_user"}).AddRow("message-zero", 42, 0, 12, 99))
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE `yu_message` SET `status`=.*scope_path").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT from_user,to_user,is_group FROM `yu_message` WHERE id=.*FOR UPDATE").WithArgs("message-zero").WillReturnRows(sqlmock.NewRows([]string{"from_user", "to_user", "is_group"}).AddRow(12, 99, 0))
			expectLockedResourceUser(mock, 12, allowed)
			if allowed {
				mock.ExpectCommit()
			} else {
				expectLockedResourceUser(mock, 99, false)
				mock.ExpectRollback()
			}
			_, err := a.manageMessage(agentMemberRequest(a, "/manage/message/dealMsg", M{"id": "message-zero", "dealType": 1}))
			if allowed {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertDenied(t, err)
			}
			assertNoResourceEvent(t, events)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAgentScopeGroupZeroMemberWritesRecheckWithoutEvent(t *testing.T) {
	for _, act := range []string{"delGroupUser", "setManager"} {
		for _, allowed := range []bool{false, true} {
			t.Run(act+fmt.Sprint(allowed), func(t *testing.T) {
				a, mock := testApp(t)
				events := resourceEventObserver(t, a, 13)
				expectAgentMemberScope(mock)
				mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
				expectResourceUser(mock, 12, true)
				expectResourceUser(mock, 13, true)
				if act == "delGroupUser" {
					mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
				}
				mock.ExpectBegin()
				query := "DELETE FROM `yu_group_user`.*scope_path"
				if act == "setManager" {
					query = "UPDATE `yu_group_user`.*scope_path"
				}
				mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery("SELECT owner_id FROM `yu_group`.*FOR UPDATE").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
				expectLockedResourceUser(mock, 12, allowed)
				if allowed {
					expectLockedResourceUser(mock, 13, true)
					mock.ExpectCommit()
				} else {
					mock.ExpectRollback()
				}
				_, err := a.manageGroup(agentMemberRequest(a, "/manage/group/"+act, M{"group_id": 9, "user_id": 13, "role": 2}))
				if allowed {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					assertDenied(t, err)
				}
				assertNoResourceEvent(t, events)
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestAgentScopeGroupZeroAvatarUpdateRechecksWithoutEvent(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			a, mock := testApp(t)
			events := resourceEventObserver(t, a, 13)
			var logs bytes.Buffer
			a.log = slog.New(slog.NewTextHandler(&logs, nil))
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "status"}).AddRow(9, 12, 1))
			mock.ExpectQuery("SELECT gu.*").WithArgs(int64(9), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"role"}))
			expectAgentMemberScope(mock)
			expectResourceUser(mock, 12, true)
			mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow("manage.groups"))
			mock.ExpectQuery("SELECT \\* FROM `yu_file`").WithArgs(int64(5), int64(7)).WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "size", "src"}).AddRow(5, 7, 2, 100, "/storage/image/avatar.png"))
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE `yu_group` SET `avatar`=.*scope_path").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT owner_id FROM `yu_group`.*FOR UPDATE").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(12))
			expectLockedResourceUser(mock, 12, allowed)
			if allowed {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			_, err := a.group(agentMemberRequest(a, "/enterprise/group/editGroupAvatar", M{"group_id": 9, "file_id": 5}))
			if allowed {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertDenied(t, err)
			}
			assertNoResourceEvent(t, events)
			if logs.Len() != 0 {
				t.Fatalf("unexpected side effect log: %s", logs.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
