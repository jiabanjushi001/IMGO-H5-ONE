package server

import (
	"net/http/httptest"
	"regexp"
	"testing"

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
