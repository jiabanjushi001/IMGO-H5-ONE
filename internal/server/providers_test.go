package server

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// No provider traffic: verify request construction and signed URLs with dummy keys.
func TestSMSRequests(t *testing.T) {
	for _, driver := range []string{"aliyun", "qcloud", "ucloud", "qiniu", "upyun", "huawei"} {
		t.Run(driver, func(t *testing.T) {
			c := M{"actions": M{"login": M{"template_id": "1234"}}, "access_key": "dummy", "access_secret": "secret", "sign_name": "测试", "appid": "123", "appkey": "secret", "public_key": "dummy", "private_key": "secret", "project_id": "project", "AccessKey": "dummy", "SecretKey": "secret", "token": "dummy-token", "url": "https://smsapi.cn-north-4.myhuaweicloud.com:443", "appKey": "dummy", "appSecret": "secret", "sender": "10690000"}
			req, e := smsRequest(context.Background(), driver, c, "login", "13800000000", "123456", time.Unix(1700000000, 0), "test-nonce")
			if e != nil {
				t.Fatal(e)
			}
			b, _ := io.ReadAll(req.Body)
			payload := string(b) + req.URL.RawQuery
			if req.URL.Scheme != "https" || !strings.Contains(payload, "13800000000") || !strings.Contains(payload, "123456") {
				t.Fatal("invalid payload", driver)
			}
			if driver == "aliyun" || driver == "ucloud" {
				v, _ := url.ParseQuery(payload)
				if v.Get("Signature") == "" {
					t.Fatal("missing signature")
				}
			}
			if driver == "qiniu" && !strings.HasPrefix(req.Header.Get("Authorization"), "Qiniu ") {
				t.Fatal("missing Qiniu signature")
			}
			if driver == "huawei" && !strings.Contains(req.Header.Get("X-WSSE"), "PasswordDigest=") {
				t.Fatal("missing Huawei signature")
			}
			if _, e = smsRequest(context.Background(), driver, M{}, "login", "13800000000", "123456", time.Now(), "nonce"); e == nil {
				t.Fatal("missing credentials accepted")
			}
		})
	}
}
func TestCloudSignedURLs(t *testing.T) {
	cases := []struct {
		disk            string
		c               M
		host, signature string
	}{
		{"aliyun", M{"bucket": "example", "endpoint": "oss-cn-hangzhou.aliyuncs.com", "accessId": "dummy-id", "accessSecret": "dummy-secret"}, "example.oss-cn-hangzhou.aliyuncs.com", "x-oss-signature"},
		{"qcloud", M{"bucket": "example", "appId": "1250000000", "region": "ap-guangzhou", "secretId": "dummy-id", "secretKey": "dummy-secret"}, "example-1250000000.cos.ap-guangzhou.myqcloud.com", "q-signature"},
		{"qiniu", M{"bucket": "example", "url": "https://cdn.example.com", "accessKey": "dummy-id", "secretKey": "dummy-secret"}, "cdn.example.com", "token"},
	}
	for _, tc := range cases {
		t.Run(tc.disk, func(t *testing.T) {
			s, e := newObjectStore(tc.disk, tc.c)
			if e != nil {
				t.Fatal(e)
			}
			raw, e := s.SignedGet(context.Background(), "image/测试 image.png")
			if e != nil {
				t.Fatal(e)
			}
			u, e := url.Parse(raw)
			if e != nil {
				t.Fatal(e)
			}
			if u.Scheme != "https" || u.Host != tc.host || u.Path != "/image/测试 image.png" || u.Query().Get(tc.signature) == "" {
				t.Fatalf("bad signed URL for %s", tc.disk)
			}
			if strings.Contains(raw, "dummy-secret") {
				t.Fatal("secret leaked in URL")
			}
		})
	}
}
func TestModerationFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		body    string
		status  int
		wantErr bool
	}{{`{"code":0,"data":{"labels":""}}`, 200, false}, {`{"code":0,"data":{"labels":"profanity"}}`, 200, true}, {`{}`, 200, true}, {`{"code":1}`, 200, true}, {`not-json`, 200, true}, {`{"code":0}`, 503, true}} {
		a := &App{cfg: Config{ModerationToken: "test-token"}}
		a.outbound = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("Authorization") != "AppCode test-token" || r.URL.Path != "/green/text" {
				t.Fatal("wrong moderation request")
			}
			r.ParseForm()
			if r.Form.Get("content") != "hello" {
				t.Fatal("HTML not stripped")
			}
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{}}, nil
		})}
		if e := a.moderate(context.Background(), "<b>hello</b>", "chat_detection"); (e != nil) != tc.wantErr {
			t.Errorf("body %s: %v", tc.body, e)
		}
	}
}
func TestBundledIPDatabase(t *testing.T) {
	d, e := loadIPDatabase("../../data/17monipdb.dat")
	if e != nil {
		t.Fatal(e)
	}
	if d.find("8.8.8.8") == "" || d.find("114.114.114.114") == "" {
		t.Fatal("database lookup failed")
	}
	if d.find("bad-ip") != "" || d.find("::1") != "" {
		t.Fatal("invalid IP lookup")
	}
	t.Log("bundled IP database version:", d.find("255.255.255.255"))
}
