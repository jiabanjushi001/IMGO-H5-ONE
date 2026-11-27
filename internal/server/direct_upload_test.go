package server

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

func TestDirectUploadCapabilityCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		config  M
		enabled bool
	}{
		{"local", M{"disk": "local", "directUpload": "1"}, false},
		{"legacy aliyun config", M{"disk": "aliyun"}, false},
		{"explicitly disabled", M{"disk": "aliyun", "directUpload": "0"}, false},
		{"enabled without cname", M{"disk": "aliyun", "directUpload": "1", "aliyun": M{}}, false},
		{"explicitly enabled", M{"disk": "aliyun", "directUpload": "1", "aliyun": M{"domain": "https://oss.example.com"}}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capability := directUploadCapability(test.config)
			if got := capability["enabled"]; got != test.enabled {
				t.Fatalf("enabled=%v, want %v", got, test.enabled)
			}
			if _, leaked := capability["accessSecret"]; leaked {
				t.Fatal("public capability leaked accessSecret")
			}
		})
	}
}

func TestDirectUploadPublicEndpointUsesESAHost(t *testing.T) {
	got, err := directUploadPublicEndpoint("oss.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://oss.example.com/" {
		t.Fatalf("endpoint=%q", got)
	}
	for _, invalid := range []string{"http://example.com", "https://localhost", "https://127.0.0.1", "https://oss-cn-hangzhou.aliyuncs.com", "https://oss.example.com/path", "https://user@example.com"} {
		if _, err = directUploadPublicEndpoint(invalid); err == nil {
			t.Fatalf("invalid ESA/CNAME domain accepted: %s", invalid)
		}
	}
}

func TestDirectUploadTicketRoundTripAndTamperProtection(t *testing.T) {
	a := &App{cfg: Config{JWTKey: strings.Repeat("k", 32)}}
	want := directUploadTicket{UID: 7, Action: "uploadfile", Key: "storage/image/2026-11-22/7/a.jpg", FileName: "a.jpg", Ext: "jpg", MIME: "image/jpeg", Kind: "image", Category: 2, MaxSize: 1024, DeclaredSize: 100, ExpiresAt: time.Now().Add(time.Minute).Unix()}
	token, err := a.signDirectUploadTicket(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.parseDirectUploadTicket(token)
	if err != nil {
		t.Fatal(err)
	}
	if got.UID != want.UID || got.Key != want.Key || got.MaxSize != want.MaxSize {
		t.Fatalf("round trip mismatch: %#v", got)
	}
	parts := strings.Split(token, ".")
	parts[0] = "A" + parts[0][1:]
	if _, err = a.parseDirectUploadTicket(strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered ticket was accepted")
	}

	expired := want
	expired.ExpiresAt = time.Now().Add(-time.Second).Unix()
	token, err = a.signDirectUploadTicket(expired)
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.parseDirectUploadTicket(token)
	var client clientError
	if !errors.As(err, &client) || client.code != 410 {
		t.Fatalf("expired ticket error=%v", err)
	}
}

func TestUploadExtensionAllowedRejectsExecutableContent(t *testing.T) {
	config := M{"fileExt": []any{"jpg", "mp4", "js"}}
	if !uploadExtensionAllowed(config, "JPG") {
		t.Fatal("configured safe extension rejected")
	}
	if uploadExtensionAllowed(config, "js") {
		t.Fatal("active content extension accepted")
	}
}

func TestDirectUploadContentTypeRemovesParameters(t *testing.T) {
	if got := directUploadContentType("txt"); got != "text/plain" {
		t.Fatalf("txt content type=%q", got)
	}
	if got := directUploadContentType("MP4"); got != "video/mp4" {
		t.Fatalf("mp4 content type=%q", got)
	}
	if got := directUploadContentType("unknown-extension"); got != "application/octet-stream" {
		t.Fatalf("unknown content type=%q", got)
	}
}

func TestDirectObjectStatFromRange(t *testing.T) {
	etag := `"d41d8cd98f00b204e9800998ecf8427e"`
	rangeValue := "bytes 0-0/7766425"
	stat, err := directObjectStatFromRange(&oss.GetObjectResult{ContentLength: 1, ContentRange: &rangeValue, ETag: &etag, Body: io.NopCloser(strings.NewReader("x"))})
	if err != nil {
		t.Fatal(err)
	}
	if stat.ContentLength != 7766425 || stat.ETag == nil || *stat.ETag != etag {
		t.Fatalf("unexpected ranged stat: %#v", stat)
	}
	badRange := "bytes 0-0/*"
	if _, err = directObjectStatFromRange(&oss.GetObjectResult{ContentLength: 1, ContentRange: &badRange}); err == nil {
		t.Fatal("invalid total size accepted")
	}
	stat, err = directObjectStatFromRange(&oss.GetObjectResult{ContentLength: 42, ETag: &etag})
	if err != nil || stat.ContentLength != 42 {
		t.Fatalf("full response fallback failed: %#v, %v", stat, err)
	}
}
