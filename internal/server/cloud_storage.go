package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/qiniu/go-sdk/v7/auth"
	qiniu "github.com/qiniu/go-sdk/v7/storage"
	cos "github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type objectStore interface {
	Put(context.Context, string, string, string) error
	SignedGet(context.Context, string) (string, error)
}
type ossStore struct {
	client *oss.Client
	bucket string
}

func (s *ossStore) Put(ctx context.Context, key, file, mime string) error {
	f, e := os.Open(file)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = s.client.PutObject(ctx, &oss.PutObjectRequest{Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(key), Body: f, ContentType: oss.Ptr(mime)})
	return e
}
func (s *ossStore) SignedGet(ctx context.Context, key string) (string, error) {
	result, e := s.client.Presign(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(key)}, oss.PresignExpires(5*time.Minute))
	if e != nil {
		return "", e
	}
	return result.URL, nil
}

type cosStore struct{ client *cos.Client }

func (s *cosStore) Put(ctx context.Context, key, file, mime string) error {
	_, e := s.client.Object.PutFromFile(ctx, key, file, &cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: mime}})
	return e
}
func (s *cosStore) SignedGet(ctx context.Context, key string) (string, error) {
	u, e := s.client.Object.GetPresignedURL2(ctx, http.MethodGet, key, 5*time.Minute, nil)
	if e != nil {
		return "", e
	}
	return u.String(), nil
}

type qiniuStore struct {
	credentials    *auth.Credentials
	bucket, domain string
}

func (s *qiniuStore) Put(ctx context.Context, key, file, mime string) error {
	policy := qiniu.PutPolicy{Scope: s.bucket + ":" + key, Expires: 600}
	up := qiniu.NewFormUploader(&qiniu.Config{UseHTTPS: true})
	var result qiniu.PutRet
	return up.PutFile(ctx, &result, policy.UploadToken(s.credentials), key, file, &qiniu.PutExtra{MimeType: mime})
}
func (s *qiniuStore) SignedGet(_ context.Context, key string) (string, error) {
	return qiniu.MakePrivateURLv2(s.credentials, s.domain, key, time.Now().Add(5*time.Minute).Unix()), nil
}
func newObjectStore(disk string, c M) (objectStore, error) {
	required := func(keys ...string) error {
		for _, key := range keys {
			if str(c[key]) == "" {
				return clientError{"云存储缺少配置项：" + disk + "." + key, 503}
			}
		}
		return nil
	}
	switch disk {
	case "aliyun":
		if e := required("bucket", "endpoint", "accessId", "accessSecret"); e != nil {
			return nil, e
		}
		endpoint := str(c["endpoint"])
		if !strings.Contains(endpoint, "://") {
			endpoint = "https://" + endpoint
		}
		u, e := url.Parse(endpoint)
		if e != nil || u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".aliyuncs.com") {
			return nil, clientError{"OSS endpoint 必须为 HTTPS 阿里云域名", 400}
		}
		region := str(c["region"])
		if region == "" {
			region = strings.TrimPrefix(strings.Split(u.Hostname(), ".")[0], "oss-")
			region = strings.TrimSuffix(region, "-internal")
		}
		cfg := oss.LoadDefaultConfig().WithRegion(region).WithEndpoint(endpoint).WithCredentialsProvider(credentials.NewStaticCredentialsProvider(str(c["accessId"]), str(c["accessSecret"]))).WithHttpClient(&http.Client{Timeout: 60 * time.Second})
		return &ossStore{oss.NewClient(cfg), str(c["bucket"])}, nil
	case "qcloud":
		if e := required("bucket", "region", "secretId", "secretKey"); e != nil {
			return nil, e
		}
		bucket := str(c["bucket"])
		if str(c["appId"]) != "" && !strings.HasSuffix(bucket, "-"+str(c["appId"])) {
			bucket += "-" + str(c["appId"])
		}
		host := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, str(c["region"]))
		u, e := url.Parse(host)
		if e != nil || strings.ContainsAny(bucket+str(c["region"]), "/\\@?#:") {
			return nil, clientError{"COS 配置无效", 400}
		}
		client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{Timeout: 60 * time.Second, Transport: &cos.AuthorizationTransport{SecretID: str(c["secretId"]), SecretKey: str(c["secretKey"])}})
		return &cosStore{client}, nil
	case "qiniu":
		if e := required("bucket", "url", "accessKey", "secretKey"); e != nil {
			return nil, e
		}
		domain := strings.TrimRight(str(c["url"]), "/")
		u, e := url.Parse(domain)
		if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return nil, clientError{"七牛下载域名必须使用 HTTPS", 400}
		}
		return &qiniuStore{auth.New(str(c["accessKey"]), str(c["secretKey"])), str(c["bucket"]), domain}, nil
	}
	return nil, clientError{"不支持的存储类型", 400}
}
func (a *App) uploadCloud(ctx context.Context, config M, key, path, mime string) error {
	store, e := newObjectStore(str(config["disk"]), obj(config[str(config["disk"])]))
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return store.Put(ctx, key, path, mime)
}
func (a *App) objectURL(ctx context.Context, f M) (string, error) {
	config := a.config(ctx, "fileUpload")
	disk := str(config["disk"])
	key := strings.TrimLeft(str(f["src"]), "/")
	meta, e := one(ctx, a.db, "SELECT disk,object_key FROM "+a.t("imgo_object")+" WHERE file_id=?", f["file_id"])
	if e == nil {
		disk = str(meta["disk"])
		key = str(meta["object_key"])
	} else if !errors.Is(e, sql.ErrNoRows) {
		return "", e
	} else if strings.HasPrefix(key, "storage/") {
		return "", nil
	}
	if disk == "" || disk == "local" {
		return "", nil
	}
	store, e := newObjectStore(disk, obj(config[disk]))
	if e != nil {
		return "", e
	}
	return store.SignedGet(ctx, key)
}
