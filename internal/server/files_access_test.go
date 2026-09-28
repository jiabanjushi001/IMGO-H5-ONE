package server

import (
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func filesRequest(a *App, uid, roleID int64, params M) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/enterprise/files/index", nil)
	return &request{app: a, c: c, user: M{"user_id": uid, "admin_role_id": roleID}, p: params}
}

func TestEnterpriseFilesAllReturnsAllFilesForAuthorizedRole(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT p.permission_key FROM `yu_imgo_admin_role`").WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"permission_key"}).AddRow("manage.files"))
	mock.ExpectQuery("SELECT agent_mode FROM `yu_imgo_admin_role`").WithArgs(int64(3)).WillReturnRows(sqlmock.NewRows([]string{"agent_mode"}).AddRow(0))
	where := "f.status=1 AND COALESCE(f.delete_time,0)=0"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_file` f WHERE " + where)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT f.* FROM `yu_file` f WHERE "+where+" ORDER BY f.file_id DESC LIMIT ? OFFSET ?")).
		WithArgs(int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "name", "ext", "src"}).AddRow(8, "图片", "jpg", "/storage/image/a.jpg"))

	result, err := a.files(filesRequest(a, 7, 3, M{"is_all": 1}))
	if err != nil || len(result.([]M)) != 1 {
		t.Fatalf("authorized all-files request failed: %#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnterpriseFilesAllRequiresManageFilesPermission(t *testing.T) {
	a, mock := testApp(t)
	assertDenied(t, func() error {
		_, err := a.files(filesRequest(a, 7, 0, M{"is_all": 1}))
		return err
	}())
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentCanReadOwnUploadedMedia(t *testing.T) {
	a, mock := testApp(t)
	if !a.canReadFile(httptest.NewRequest("GET", "/", nil).Context(), 7, M{"user_id": int64(7)}) {
		t.Fatal("mentor must be able to read their own uploaded media")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentMediaUsesChatAndAvatarAccessInsteadOfAdminScope(t *testing.T) {
	for _, tc := range []struct {
		name     string
		inChat   bool
		isAvatar bool
		allowed  bool
	}{
		{name: "foreign uploader in chat", inChat: true, allowed: true},
		{name: "foreign profile avatar", isAvatar: true, allowed: true},
		{name: "unrelated foreign media", allowed: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			path := "storage/image/foreign.png"
			c.Request = httptest.NewRequest("GET", "/"+path, nil)
			c.Request.Header.Set("Authorization", signToken(a.cfg.JWTKey, 7, "media-chat", time.Now().Add(time.Hour)))

			mock.ExpectQuery("SELECT \\* FROM `yu_file`").WithArgs("/"+path, path).
				WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "src", "cate", "parent_id"}).AddRow(5, 99, "/"+path, 2, 0))
			mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("sysInfo").
				WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{}`))
			mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").WithArgs("media-chat", int64(7), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
			mock.ExpectQuery("SELECT \\* FROM `yu_user`").WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id", "admin_role_id"}).AddRow(7, 3))
			mock.ExpectQuery("SELECT id FROM `yu_emoji`").WithArgs(int64(5), int64(7)).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))
			messageRows := sqlmock.NewRows([]string{"msg_id"})
			if tc.inChat {
				messageRows.AddRow(11)
			}
			mock.ExpectQuery("SELECT m.msg_id FROM `yu_message`").WithArgs(int64(5), int64(7), int64(7), int64(7), int64(7)).
				WillReturnRows(messageRows)
			if !tc.inChat {
				avatarRows := sqlmock.NewRows([]string{"user_id"})
				if tc.isAvatar {
					avatarRows.AddRow(99)
				}
				mock.ExpectQuery("SELECT user_id FROM `yu_user`").WithArgs("/" + path).WillReturnRows(avatarRows)
			}
			if tc.allowed {
				mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("fileUpload").
					WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"disk":"local"}`))
				mock.ExpectQuery("SELECT disk,object_key FROM `yu_imgo_object`").WithArgs(int64(5)).
					WillReturnRows(sqlmock.NewRows([]string{"disk", "object_key"}))
			}

			if got := a.authorizeStorage(c, path); got != tc.allowed {
				t.Fatalf("authorizeStorage()=%v want %v, status=%d", got, tc.allowed, recorder.Code)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
