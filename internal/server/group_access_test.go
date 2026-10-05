package server

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestGroupReadAccess(t *testing.T) {
	for _, tc := range []struct {
		name, action    string
		uid, userRole   int
		member, allowed bool
	}{
		{"admin outside group", "groupuserlist", 8, 1, false, true},
		{"superadmin outside group", "groupuserlist", 1, 0, false, true},
		{"ordinary outsider", "groupuserlist", 8, 0, false, false},
		{"ordinary member", "groupuserlist", 8, 0, true, true},
		{"admin cannot mutate as nonmember", "editgroupname", 8, 1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/group/"+tc.action, nil)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(5, 2))
			memberQuery := mock.ExpectQuery("SELECT gu.*").WithArgs(int64(5), int64(tc.uid))
			if tc.member {
				memberQuery.WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(3))
			} else {
				memberQuery.WillReturnError(sql.ErrNoRows)
			}
			if tc.allowed {
				mock.ExpectQuery("SELECT COUNT").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
				mock.ExpectQuery("SELECT \\* FROM `yu_group_user`").WithArgs(int64(5), 2000, 0).WillReturnRows(sqlmock.NewRows([]string{"user_id", "role"}))
			}
			_, err := a.group(&request{app: a, c: c, p: M{"group_id": "group-5"}, user: M{"user_id": tc.uid, "role": tc.userRole}})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, err=%v", tc.allowed, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEditGroupAvatarAccess(t *testing.T) {
	for _, tc := range []struct {
		name     string
		role     int
		userRole int
		cate     int
		allowed  bool
	}{
		{"owner", 1, 0, 2, true},
		{"manager", 2, 0, 2, true},
		{"member denied", 3, 0, 2, false},
		{"system admin member", 3, 1, 2, true},
		{"system admin outsider", 0, 1, 2, true},
		{"outsider denied", 0, 0, 2, false},
		{"non-image denied", 1, 0, 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/group/editGroupAvatar", nil)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(5, 2))
			membership := mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(2))
			if tc.role == 0 {
				membership.WillReturnError(sql.ErrNoRows)
			} else {
				membership.WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(tc.role))
			}
			if tc.userRole > 0 || tc.role == 1 || tc.role == 2 {
				mock.ExpectQuery("SELECT \\* FROM `yu_file`").WithArgs(int64(9), int64(2)).WillReturnRows(sqlmock.NewRows([]string{"file_id", "src", "cate", "size"}).AddRow(9, "/storage/image/test/2/avatar.png", tc.cate, 1024))
			}
			if tc.allowed {
				mock.ExpectExec("UPDATE `yu_group` SET `avatar`=\\? WHERE group_id=\\?").WithArgs("/storage/image/test/2/avatar.png", int64(5)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(2))
			}
			result, err := a.group(&request{app: a, c: c, p: M{"group_id": "group-5", "file_id": 9}, user: M{"user_id": 2, "role": tc.userRole}})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, result=%v, err=%v", tc.allowed, result, err)
			}
			if tc.allowed && str(result.(M)["avatar"]) == "" {
				t.Fatal("missing avatar URL", result)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSetNoSpeakOwnerAndManager(t *testing.T) {
	for _, tc := range []struct {
		name    string
		role    int
		allowed bool
	}{
		{"owner allowed", 1, true},
		{"manager allowed", 2, true},
		{"member denied", 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/group/setNoSpeak", nil)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id"}).AddRow(5, 2))
			mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(2)).WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(tc.role))
			if tc.allowed {
				mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(9)).WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(3))
				mock.ExpectExec("UPDATE `yu_group_user` SET `no_speak_time`=\\? WHERE group_id=\\? AND user_id=\\?").WithArgs(sqlmock.AnyArg(), int64(5), int64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(2).AddRow(9))
			}
			_, err := a.group(&request{app: a, c: c, p: M{"id": "group-5", "user_id": 9, "noSpeakTimer": 1}, user: M{"user_id": 2}})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, err=%v", tc.allowed, err)
			}
			if !tc.allowed {
				var denied clientError
				if !errors.As(err, &denied) || denied.code != 403 {
					t.Fatalf("ordinary member should receive 403, err=%v", err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCanInviteGroupMember(t *testing.T) {
	for _, tc := range []struct {
		name          string
		role          int64
		managerInvite int64
		allowed       bool
	}{
		{"owner remains allowed when switch is off", 1, 0, true},
		{"owner remains allowed when switch is on", 1, 1, true},
		{"manager allowed when enabled", 2, 1, true},
		{"manager denied when disabled", 2, 0, false},
		{"ordinary member denied when enabled", 3, 1, false},
		{"ordinary member denied when disabled", 3, 0, false},
		{"outsider denied", 0, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canInviteGroupMember(tc.role, tc.managerInvite); got != tc.allowed {
				t.Fatalf("canInviteGroupMember(%d, %d)=%v, want %v", tc.role, tc.managerInvite, got, tc.allowed)
			}
		})
	}
}

func TestCanViewGroupMemberAccount(t *testing.T) {
	tests := []struct {
		name      string
		role      int64
		adminRead bool
		want      bool
	}{
		{"group owner", 1, false, true},
		{"group manager", 2, false, true},
		{"ordinary member", 3, false, false},
		{"system administrator", 0, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canViewGroupMemberAccount(tt.role, tt.adminRead); got != tt.want {
				t.Fatalf("canViewGroupMemberAccount(%d, %v) = %v, want %v", tt.role, tt.adminRead, got, tt.want)
			}
		})
	}
}

func TestManageGroupOwnerCanSetAnyMemberAsManager(t *testing.T) {
	a, mock := testApp(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/manage/Group/setManager", nil)

	expectAgentMemberScope(mock)
	mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"owner_id"}).AddRow(7),
	)
	expectResourceUser(mock, 9, false)
	mock.ExpectQuery("SELECT owner_id FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"owner_id"}).AddRow(7),
	)
	mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"group_id", "owner_id", "setting", "status"}).AddRow(5, 7, `{}`, 1),
	)
	mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"role"}).AddRow(1),
	)
	mock.ExpectExec("UPDATE `yu_group_user` SET .*role.*WHERE group_id=\\? AND user_id=\\? AND status=1").
		WithArgs(int64(2), int64(5), int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"user_id"}).AddRow(7).AddRow(9),
	)

	_, err := a.manageGroup(&request{
		app:  a,
		c:    c,
		p:    M{"group_id": 5, "user_id": 9, "role": 2},
		user: M{"user_id": 7, "admin_role_id": 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAddGroupUserRejectsUnauthorizedInviters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		role    int
		setting string
	}{
		{"ordinary member cannot invite", 3, `{"manager_invite":1}`},
		{"manager cannot invite when disabled", 2, `{"manager_invite":0}`},
		{"legacy invite flag does not enable managers", 2, `{"invite":1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/group/addGroupUser", nil)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(
				sqlmock.NewRows([]string{"group_id", "owner_id", "setting"}).
					AddRow(5, 8, tc.setting),
			)
			mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(2)).WillReturnRows(
				sqlmock.NewRows([]string{"role"}).AddRow(tc.role),
			)

			_, err := a.group(&request{
				app:  a,
				c:    c,
				p:    M{"group_id": "group-5", "user_ids": []int64{9}},
				user: M{"user_id": 2},
			})
			var denied clientError
			if !errors.As(err, &denied) || denied.code != 403 {
				t.Fatalf("expected 403, err=%v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGroupMuteSettingOwnerAndManager(t *testing.T) {
	current := `{"manage":1,"invite":0,"nospeak":0,"history":1,"number_join":2}`
	for _, tc := range []struct {
		name    string
		role    int
		setting M
		allowed bool
	}{
		{"owner allowed", 1, M{"manage": 1, "invite": 0, "nospeak": 1, "history": 1}, true},
		{"manager mute allowed", 2, M{"manage": 1, "invite": 0, "nospeak": 1, "history": 1, "number_join": 2}, true},
		{"manager other setting denied", 2, M{"manage": 0, "invite": 0, "nospeak": 1, "history": 1, "number_join": 2}, false},
		{"member denied", 3, M{"manage": 1, "invite": 0, "nospeak": 1, "history": 1, "number_join": 2}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/group/groupSetting", nil)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "setting"}).AddRow(5, 2, current))
			mock.ExpectQuery("SELECT gu.\\*").WithArgs(int64(5), int64(2)).WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(tc.role))
			if tc.allowed {
				mock.ExpectExec("UPDATE `yu_group` SET `setting`=\\? WHERE group_id=\\?").WithArgs(sqlmock.AnyArg(), int64(5)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery("SELECT user_id FROM `yu_group_user`").WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(2))
			}
			_, err := a.group(&request{app: a, c: c, p: M{"group_id": "group-5", "setting": tc.setting}, user: M{"user_id": 2}})
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, err=%v", tc.allowed, err)
			}
			if !tc.allowed {
				var denied clientError
				if !errors.As(err, &denied) || denied.code != 403 {
					t.Fatalf("expected 403, err=%v", err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGroupInvitationPreview(t *testing.T) {
	for _, mode := range []string{"valid", "expired", "other-group", "tampered"} {
		t.Run(mode, func(t *testing.T) {
			a, mock := testApp(t)
			gid := int64(5)
			tokenGid := gid
			expiry := time.Now().Add(time.Hour).Unix()
			if mode == "expired" {
				expiry = time.Now().Add(-time.Hour).Unix()
			}
			if mode == "other-group" {
				tokenGid = 6
			}
			payload := fmt.Sprintf("%d.%d", tokenGid, expiry)
			token := payload + "." + a.signLink(payload)
			if mode == "tampered" {
				token += "x"
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/group/groupInfo", nil)
			mock.ExpectQuery("SELECT \\* FROM `yu_group`").WithArgs(gid).WillReturnRows(sqlmock.NewRows([]string{"group_id", "owner_id", "name", "setting"}).AddRow(gid, 2, "Test", `{"invite":1}`))
			mock.ExpectQuery("SELECT gu.*").WithArgs(gid, int64(9)).WillReturnError(sql.ErrNoRows)
			if mode == "valid" {
				mock.ExpectQuery("SELECT user_id,realname,avatar").WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "realname", "avatar"}).AddRow(2, "Owner", ""))
				mock.ExpectQuery("SELECT COUNT").WithArgs(gid).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
			}
			data, err := a.group(&request{app: a, c: c, p: M{"group_id": "group-5", "token": token}, user: M{"user_id": 9, "role": 0}})
			if mode == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				info := data.(M)
				if number(info["isJoin"]) != 0 || info["qrUrl"] != nil || info["userInfo"] != nil {
					t.Fatal("preview leaks member fields", info)
				}
			} else if err == nil {
				t.Fatal("invalid invite accepted")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
