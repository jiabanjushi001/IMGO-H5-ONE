package server

import (
	"database/sql"
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
