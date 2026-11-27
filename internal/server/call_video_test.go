package server

import (
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func writeCallVideoFixture(t *testing.T, a *App, src string) {
	t.Helper()
	a.cfg.PublicDir = t.TempDir()
	path := filepath.Join(a.cfg.PublicDir, filepath.FromSlash(src[1:]))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("video"), 0o640); err != nil {
		t.Fatal(err)
	}
}

func callVideoRequest(a *App, path string, uid int64, params M) *request {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", path, nil)
	return &request{app: a, c: c, user: M{"user_id": uid}, p: params}
}

func TestRecordedAndLiveCallsUseSameStoredContent(t *testing.T) {
	if got := rtcCallingContent(1); got != "视频通话" {
		t.Fatalf("video call content=%q", got)
	}
	if got := rtcCallingContent(0); got != "语音通话" {
		t.Fatalf("voice call content=%q", got)
	}
	for _, legacy := range []string{"预设视频通话", "预录视频通话"} {
		if got := normalizeRTCContent(legacy); got != "视频通话" {
			t.Fatalf("legacy content %q normalized to %q", legacy, got)
		}
	}
	if got := normalizeRTCContent("通话时长 01:05"); got != "通话时长 01:05" {
		t.Fatalf("call result changed to %q", got)
	}
}

func TestCallVideoIndexRequiresMentorAndReturnsOwnVideos(t *testing.T) {
	a, mock := testApp(t)
	writeCallVideoFixture(t, a, "/storage/video/demo.mp4")
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	where := "user_id=? AND status=1 AND COALESCE(delete_time,0)=0 AND cate=4"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_file` WHERE " + where)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT file_id,name,ext,src,size,create_time FROM `yu_file` WHERE "+where+" ORDER BY file_id DESC LIMIT ? OFFSET ?")).
		WithArgs(int64(7), int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "name", "ext", "src", "size", "create_time"}).AddRow(18, "演示", "mp4", "/storage/video/demo.mp4", 1024, 20))

	result, err := a.callVideo(callVideoRequest(a, "/enterprise/callVideo/index", 7, M{}))
	if err != nil {
		t.Fatal(err)
	}
	items := result.([]M)
	if len(items) != 1 || items[0]["name"] != "演示.mp4" || items[0]["url"] != "/storage/video/demo.mp4" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCallVideoIndexAllowsSuperAdministrator(t *testing.T) {
	a, mock := testApp(t)
	where := "user_id=? AND status=1 AND COALESCE(delete_time,0)=0 AND cate=4"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_file` WHERE " + where)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT file_id,name,ext,src,size,create_time FROM `yu_file` WHERE "+where+" ORDER BY file_id DESC LIMIT ? OFFSET ?")).
		WithArgs(int64(1), int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "name", "ext", "src", "size", "create_time"}))

	result, err := a.callVideo(callVideoRequest(a, "/enterprise/callVideo/index", 1, M{}))
	if err != nil {
		t.Fatal(err)
	}
	if items := result.([]M); len(items) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordedCallVideoRejectsOrdinaryUser(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	_, err := a.validateRecordedCallVideo(callVideoRequest(a, "/enterprise/im/sendToMsg", 9, M{"type": 1, "video_file_id": 18}))
	var clientErr clientError
	if !errors.As(err, &clientErr) || clientErr.code != 403 {
		t.Fatalf("error=%#v, want 403", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordedCallVideoAllowsSuperAdministrator(t *testing.T) {
	a, mock := testApp(t)
	writeCallVideoFixture(t, a, "/storage/video/demo.mp4")
	mock.ExpectQuery("SELECT file_id,user_id,cate,status,delete_time,src FROM `yu_file`").
		WithArgs(int64(18), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "status", "delete_time", "src"}).AddRow(18, 1, 4, 1, 0, "/storage/video/demo.mp4"))

	file, err := a.validateRecordedCallVideo(callVideoRequest(a, "/enterprise/im/sendToMsg", 1, M{"type": 1, "video_file_id": 18}))
	if err != nil {
		t.Fatal(err)
	}
	if number(file["file_id"]) != 18 {
		t.Fatalf("unexpected file: %#v", file)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordedCallVideoMustBelongToMentor(t *testing.T) {
	a, mock := testApp(t)
	mock.ExpectQuery("SELECT u.user_id FROM `yu_user` u JOIN `yu_imgo_admin_role` r").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT file_id,user_id,cate,status,delete_time,src FROM `yu_file`").
		WithArgs(int64(18), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}))
	_, err := a.validateRecordedCallVideo(callVideoRequest(a, "/enterprise/im/sendToMsg", 7, M{"type": 1, "video_file_id": 18}))
	var clientErr clientError
	if !errors.As(err, &clientErr) || clientErr.message != "预录视频不存在或不属于当前账号" {
		t.Fatalf("error=%#v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCallVideoIndexHidesMissingLocalVideos(t *testing.T) {
	a, mock := testApp(t)
	a.cfg.PublicDir = t.TempDir()
	where := "user_id=? AND status=1 AND COALESCE(delete_time,0)=0 AND cate=4"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) n FROM `yu_file` WHERE " + where)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT file_id,name,ext,src,size,create_time FROM `yu_file` WHERE "+where+" ORDER BY file_id DESC LIMIT ? OFFSET ?")).
		WithArgs(int64(1), int64(20), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "name", "ext", "src", "size", "create_time"}).AddRow(18, "丢失视频", "mp4", "/storage/video/missing.mp4", 1024, 20))
	mock.ExpectQuery("SELECT file_id FROM `yu_imgo_object`").
		WithArgs(int64(18)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}))

	req := callVideoRequest(a, "/enterprise/callVideo/index", 1, M{})
	result, err := a.callVideo(req)
	if err != nil {
		t.Fatal(err)
	}
	if items := result.([]M); len(items) != 0 || req.count != 0 {
		t.Fatalf("missing local video must be hidden: items=%#v count=%d", items, req.count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordedCallVideoRejectsMissingLocalFile(t *testing.T) {
	a, mock := testApp(t)
	a.cfg.PublicDir = t.TempDir()
	mock.ExpectQuery("SELECT file_id,user_id,cate,status,delete_time,src FROM `yu_file`").
		WithArgs(int64(18), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "user_id", "cate", "status", "delete_time", "src"}).AddRow(18, 1, 4, 1, 0, "/storage/video/missing.mp4"))
	mock.ExpectQuery("SELECT file_id FROM `yu_imgo_object`").
		WithArgs(int64(18)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}))

	_, err := a.validateRecordedCallVideo(callVideoRequest(a, "/enterprise/im/sendToMsg", 1, M{"type": 1, "video_file_id": 18}))
	var clientErr clientError
	if !errors.As(err, &clientErr) || clientErr.message != "预设视频文件已丢失，请重新上传" {
		t.Fatalf("error=%#v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
