package server

import (
	"bytes"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadReturnsRelativePaths(t *testing.T) {
	for _, action := range []string{"uploadFile", "uploadAvatar", "uploadImage", "uploadEmoji"} {
		t.Run(action, func(t *testing.T) {
			a, mock := testApp(t)
			a.cfg.BaseURL = "https://api.example"
			a.cfg.PublicDir = t.TempDir()
			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			part, err := w.CreateFormFile("file", "test.png")
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(part, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			w.Close()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/common/upload/"+action, &b)
			c.Request.Header.Set("Content-Type", w.FormDataContentType())
			mock.ExpectQuery("SELECT value FROM `yu_config`").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"disk":"local","size":50,"fileExt":["png"]}`))
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO `yu_file`").WillReturnResult(sqlmock.NewResult(42, 1))
			mock.ExpectCommit()
			if action == "uploadAvatar" {
				mock.ExpectExec("UPDATE `yu_user`").WillReturnResult(sqlmock.NewResult(0, 1))
			}
			if action == "uploadEmoji" {
				mock.ExpectExec("INSERT INTO `yu_emoji`").WillReturnResult(sqlmock.NewResult(1, 1))
			}
			result, err := a.upload(&request{app: a, c: c, p: M{}, user: M{"user_id": 2}})
			if err != nil {
				t.Fatal(err)
			}
			path := str(result)
			if action == "uploadFile" {
				m := result.(M)
				path = str(m["url"])
				if path != str(m["src"]) {
					t.Fatal("src/url differ", m)
				}
			}
			if !strings.HasPrefix(path, "/storage/image/") || strings.Contains(path, "://") {
				t.Fatal("not relative", path)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
