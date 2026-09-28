package server

import (
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestClientDownloadURL(t *testing.T) {
	for _, tc := range []struct {
		url   string
		valid bool
	}{
		{"", true}, {"https://example.com/app.apk?version=2", true}, {"http://127.0.0.1:8088/app.apk", true},
		{"javascript:alert(1)", false}, {"//example.com/app.apk", false}, {"https://", false}, {"https://user:password@example.com/", false},
		{"https://example.com/downapp", false}, {"https://example.com/\r\nheader", false},
	} {
		if err := validateClientDownloadURL(tc.url); (err == nil) != tc.valid {
			t.Errorf("%q: %v", tc.url, err)
		}
	}
}

func TestClientDownloadEntry(t *testing.T) {
	for _, configured := range []bool{false, true} {
		a, mock := testApp(t)
		value := `{}`
		if configured {
			value = `{"clientDownloadUrl":"https://example.com/app.apk"}`
		}
		mock.ExpectQuery("SELECT value FROM `yu_config`").WithArgs("sysInfo").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(value))
		r := agentMemberRequest(a, "/index/index/downapp", M{})
		if _, err := a.index(r); err != nil {
			t.Fatal(err)
		}
		if configured && r.c.Writer.Header().Get("Location") != "https://example.com/app.apk" {
			t.Fatal("missing configured redirect")
		}
		if !configured && r.c.Writer.Status() != 200 {
			t.Fatal("default download page unavailable")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
