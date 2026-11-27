package server

import (
	"bytes"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSaveUploadedVideoCover(t *testing.T) {
	a, mock := testApp(t)
	a.cfg.PublicDir = t.TempDir()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("poster", "video-cover.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if err = jpeg.Encode(part, image.NewRGBA(image.Rect(0, 0, 12, 8)), &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/upload/uploadFile", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	mock.ExpectExec("INSERT INTO `yu_file`").WillReturnResult(sqlmock.NewResult(91, 1))

	metadata, uploaded, err := a.saveUploadedVideoCover(&request{app: a, c: c, p: M{}, user: M{"user_id": 2}}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if !uploaded || number(metadata["width"]) != 12 || number(metadata["height"]) != 8 {
		t.Fatalf("unexpected metadata: uploaded=%v metadata=%v", uploaded, metadata)
	}
	poster := str(metadata["poster"])
	if !strings.HasPrefix(poster, "/storage/cover/") || strings.Contains(poster, "://") {
		t.Fatalf("unexpected poster path: %s", poster)
	}
	if _, err = os.Stat(a.cfg.PublicDir + poster); err != nil {
		t.Fatal("saved poster missing:", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveUploadedVideoCoverFallsBackWhenAbsent(t *testing.T) {
	a, _ := testApp(t)
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/upload/uploadFile", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	metadata, uploaded, err := a.saveUploadedVideoCover(&request{app: a, c: c, p: M{}, user: M{"user_id": 2}}, 42)
	if err != nil || uploaded || metadata != nil {
		t.Fatalf("expected FFmpeg fallback: uploaded=%v metadata=%v err=%v", uploaded, metadata, err)
	}
}

func TestClaimUploadedVideoCover(t *testing.T) {
	a, mock := testApp(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/upload/uploadFile", nil)
	r := &request{app: a, c: c, p: M{}, user: M{"user_id": 2}}
	poster := "/storage/image/2026-11-21/2/poster.jpg"
	mock.ExpectQuery("SELECT file_id,src,cate,user_id,parent_id FROM `yu_file`").
		WithArgs(poster, int64(2), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "src", "cate", "user_id", "parent_id"}).
			AddRow(91, poster, 2, 2, 0))
	mock.ExpectExec("UPDATE `yu_file` SET `parent_id`=\\? WHERE file_id=\\? AND user_id=\\? AND parent_id=0").
		WithArgs(int64(42), int64(91), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	metadata, claimed, err := a.claimUploadedVideoCover(r, 42, "https://api.example"+poster)
	if err != nil {
		t.Fatal(err)
	}
	if !claimed || str(metadata["poster"]) != poster {
		t.Fatalf("unexpected claim result: claimed=%v metadata=%v", claimed, metadata)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClaimUploadedVideoCoverIsIdempotentForSameVideo(t *testing.T) {
	a, mock := testApp(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/upload/completeDirect", nil)
	r := &request{app: a, c: c, p: M{}, user: M{"user_id": 2}}
	poster := "/storage/image/2026-11-21/2/poster.jpg"
	mock.ExpectQuery("SELECT file_id,src,cate,user_id,parent_id FROM `yu_file`").
		WithArgs(poster, int64(2), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "src", "cate", "user_id", "parent_id"}).
			AddRow(91, poster, 2, 2, 42))

	metadata, claimed, err := a.claimUploadedVideoCover(r, 42, poster)
	if err != nil {
		t.Fatal(err)
	}
	if !claimed || str(metadata["poster"]) != poster {
		t.Fatalf("same video must retain its claimed poster: claimed=%v metadata=%v", claimed, metadata)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClaimUploadedVideoCoverRejectsExternalPath(t *testing.T) {
	a, _ := testApp(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/common/upload/uploadFile", nil)
	metadata, claimed, err := a.claimUploadedVideoCover(
		&request{app: a, c: c, p: M{}, user: M{"user_id": 2}},
		42,
		"https://cdn.example/poster.jpg",
	)
	if err != nil || claimed || metadata != nil {
		t.Fatalf("external poster must not be claimed: claimed=%v metadata=%v err=%v", claimed, metadata, err)
	}
}
