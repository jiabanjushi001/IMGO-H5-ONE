package server

import (
	"bytes"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadValidationFailureLogsFileMetadataAndStage(t *testing.T) {
	a, mock := testApp(t)
	var logs bytes.Buffer
	a.log = slog.New(slog.NewJSONHandler(&logs, nil))
	a.routes["/common/upload/test"] = endpoint{public: true, handler: a.upload}
	mock.ExpectQuery("SELECT value FROM `yu_config`").WillReturnRows(
		sqlmock.NewRows([]string{"value"}).AddRow(`{"disk":"local","size":50,"fileExt":["png"]}`),
	)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "not-allowed.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write([]byte("not an image")); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/common/upload/test", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-ID", "upload-request-123")
	a.Router().ServeHTTP(w, req)

	var response M
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if number(response["code"]) != 400 || !strings.Contains(str(response["msg"]), "upload-request-123") {
		t.Fatalf("unexpected upload response: %s", w.Body.String())
	}
	line := logs.String()
	for _, value := range []string{`"request_id":"upload-request-123"`, `"stage":"upload_validate_type"`, `"file_name":"not-allowed.exe"`, `"file_size":12`, `"message":"不允许上传此文件类型"`} {
		if !strings.Contains(line, value) {
			t.Fatalf("upload log missing %s: %s", value, line)
		}
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequestFailureIncludesRequestIDAndStructuredLog(t *testing.T) {
	a, _ := testApp(t)
	var logs bytes.Buffer
	a.log = slog.New(slog.NewJSONHandler(&logs, nil))
	a.routes["/test/failure"] = endpoint{
		public: true,
		handler: func(r *request) (any, error) {
			return nil, r.fail("测试失败")
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test/failure", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request-123")
	a.Router().ServeHTTP(w, req)

	var body M
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if number(body["code"]) != 400 || str(body["request_id"]) != "test-request-123" {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
	if w.Header().Get("X-Request-ID") != "test-request-123" {
		t.Fatal("missing request id response header")
	}
	line := logs.String()
	for _, value := range []string{`"msg":"request failed"`, `"request_id":"test-request-123"`, `"business_code":400`, `"message":"测试失败"`} {
		if !strings.Contains(line, value) {
			t.Fatalf("log missing %s: %s", value, line)
		}
	}
}

func TestRequestPanicIsLoggedAndReturnsReference(t *testing.T) {
	a, _ := testApp(t)
	var logs bytes.Buffer
	a.log = slog.New(slog.NewJSONHandler(&logs, nil))
	a.routes["/test/panic"] = endpoint{
		public: true,
		handler: func(*request) (any, error) {
			panic("test panic")
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test/panic", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	a.Router().ServeHTTP(w, req)

	var body M
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if number(body["code"]) != 500 || str(body["request_id"]) == "" {
		t.Fatalf("unexpected panic response: %s", w.Body.String())
	}
	line := logs.String()
	if !strings.Contains(line, `"msg":"request panic"`) || !strings.Contains(line, `"panic":"test panic"`) || !strings.Contains(line, `"stack":`) {
		t.Fatalf("panic was not fully logged: %s", line)
	}
}
