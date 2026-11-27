package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"mime"
	"path/filepath"
	"strings"
	"time"

	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

const (
	directUploadProtocol = 1
	directUploadTTL      = 45 * time.Minute
)

type directUploadTicket struct {
	UID          int64  `json:"uid"`
	Action       string `json:"action"`
	Key          string `json:"key"`
	FileName     string `json:"file_name"`
	Ext          string `json:"ext"`
	MIME         string `json:"mime"`
	Kind         string `json:"kind"`
	Category     int    `json:"category"`
	MaxSize      int64  `json:"max_size"`
	DeclaredSize int64  `json:"declared_size,omitempty"`
	ExpiresAt    int64  `json:"expires_at"`
}

type directObjectStat struct {
	ContentLength int64
	ETag          *string
}

func directUploadEnabled(config M) bool {
	if str(config["disk"]) != "aliyun" {
		return false
	}
	raw, exists := config["directUpload"]
	if !exists || str(raw) == "" || (number(raw) != 1 && !strings.EqualFold(str(raw), "true")) {
		return false
	}
	_, err := normalizeAliyunCNAME(str(obj(config["aliyun"])["domain"]))
	return err == nil
}

func directUploadCapability(config M) M {
	enabled := directUploadEnabled(config)
	return M{
		"enabled": enabled,
		"provider": func() string {
			if enabled {
				return "aliyun"
			}
			return ""
		}(),
		"protocol": directUploadProtocol,
	}
}

func (a *App) directUpload(r *request) (any, error) {
	switch action(r) {
	case "preparedirect":
		return a.prepareDirectUpload(r)
	case "completedirect":
		return a.completeDirectUpload(r)
	default:
		return nil, r.fail("未知操作")
	}
}

func directUploadAction(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "uploadfile", "uploadimage", "uploadavatar", "uploademoji":
		return value, true
	default:
		return "", false
	}
}

func uploadExtensionAllowed(config M, ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	if ext == "" || strings.Contains("|html|htm|svg|js|mjs|css|exe|dll|com|bat|cmd|ps1|jar|", "|"+ext+"|") {
		return false
	}
	switch extensions := config["fileExt"].(type) {
	case []any:
		for _, value := range extensions {
			if strings.EqualFold(str(value), ext) {
				return true
			}
		}
	case []string:
		for _, value := range extensions {
			if strings.EqualFold(value, ext) {
				return true
			}
		}
	}
	return false
}

func directUploadContentType(ext string) string {
	contentType := mime.TypeByExtension("." + strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), ".")))
	if contentType == "" {
		return "application/octet-stream"
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType == "" {
		return "application/octet-stream"
	}
	return mediaType
}

func directUploadPublicEndpoint(domain string) (string, error) {
	normalized, err := normalizeAliyunCNAME(domain)
	if err != nil {
		return "", err
	}
	return normalized + "/", nil
}

func (a *App) prepareDirectUpload(r *request) (any, error) {
	config := a.config(r.ctx(), "fileUpload")
	if !directUploadEnabled(config) {
		return nil, clientError{"客户端直传 OSS 已关闭", 409}
	}
	aliyun := obj(config["aliyun"])
	if _, err := newObjectStore("aliyun", aliyun); err != nil {
		return nil, err
	}
	uploadAction, ok := directUploadAction(r.s("uploadAction"))
	if !ok {
		return nil, r.fail("上传类型无效")
	}
	fileName := strings.TrimSpace(r.s("fileName"))
	ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(r.s("ext")), "."))
	if ext == "" {
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	}
	if fileName == "" {
		fileName = "upload." + ext
	}
	if filepath.Ext(fileName) == "" && ext != "" {
		fileName += "." + ext
	}
	if filepath.Base(fileName) != fileName || strings.ContainsAny(fileName, "/\\") || !safeAsset(fileName) || !uploadExtensionAllowed(config, ext) {
		return nil, r.fail("文件名或扩展名不安全")
	}
	voice := number(r.p["voice"]) == 1 || strings.EqualFold(r.s("messageType"), "voice")
	category, kind := fileCategory(ext, voice)
	if (uploadAction == "uploadavatar" || uploadAction == "uploademoji" || uploadAction == "uploadimage") && kind != "image" {
		return nil, r.fail("仅支持图片文件")
	}
	maxSize := uploadLimitMB(config, kind) << 20
	if uploadAction == "uploademoji" && maxSize > 2<<20 {
		maxSize = 2 << 20
	}
	declaredSize := r.n("fileSize")
	if declaredSize < 0 || declaredSize > maxSize {
		return nil, r.fail(fmt.Sprintf("文件超过大小限制（最大 %d MB）", maxSize>>20))
	}
	contentType := directUploadContentType(ext)
	key := filepath.ToSlash(filepath.Join("storage", kind, time.Now().Format("2006-01-02"), fmt.Sprint(r.uid()), randomID()+"."+ext))
	expiresAt := time.Now().Add(directUploadTTL)
	policyDocument := M{
		"expiration": expiresAt.UTC().Format(time.RFC3339),
		"conditions": []any{
			M{"bucket": str(aliyun["bucket"])},
			M{"key": key},
			M{"x-oss-forbid-overwrite": "true"},
			[]any{"content-length-range", 1, maxSize},
			[]any{"eq", "$Content-Type", contentType},
		},
	}
	policyJSON, err := json.Marshal(policyDocument)
	if err != nil {
		return nil, err
	}
	policy := base64.StdEncoding.EncodeToString(policyJSON)
	policyMAC := hmac.New(sha1.New, []byte(str(aliyun["accessSecret"])))
	_, _ = policyMAC.Write([]byte(policy))
	signature := base64.StdEncoding.EncodeToString(policyMAC.Sum(nil))
	ticket, err := a.signDirectUploadTicket(directUploadTicket{
		UID: r.uid(), Action: uploadAction, Key: key, FileName: fileName, Ext: ext,
		MIME: contentType, Kind: kind, Category: category, MaxSize: maxSize,
		DeclaredSize: declaredSize, ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		return nil, err
	}
	uploadURL, err := directUploadPublicEndpoint(str(aliyun["domain"]))
	if err != nil {
		return nil, err
	}
	return M{
		"provider": "aliyun", "protocol": directUploadProtocol, "method": "POST",
		"url": uploadURL, "ticket": ticket, "expiresAt": expiresAt.Unix(), "objectKey": key,
		"fields": M{
			"key": key, "policy": policy, "OSSAccessKeyId": str(aliyun["accessId"]),
			"Signature": signature, "Content-Type": contentType,
			"x-oss-forbid-overwrite": "true", "success_action_status": "200",
		},
	}, nil
}

func (a *App) signDirectUploadTicket(ticket directUploadTicket) (string, error) {
	payload, err := json.Marshal(ticket)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(a.cfg.JWTKey+":direct-upload:v1"))
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (a *App) parseDirectUploadTicket(value string) (directUploadTicket, error) {
	var ticket directUploadTicket
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return ticket, clientError{"上传凭证无效", 400}
	}
	mac := hmac.New(sha256.New, []byte(a.cfg.JWTKey+":direct-upload:v1"))
	_, _ = mac.Write([]byte(parts[0]))
	supplied, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(supplied, mac.Sum(nil)) {
		return ticket, clientError{"上传凭证无效", 400}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(payload, &ticket) != nil || ticket.UID < 1 || ticket.Key == "" || ticket.ExpiresAt <= time.Now().Unix() {
		return directUploadTicket{}, clientError{"上传凭证已失效", 410}
	}
	return ticket, nil
}

func (a *App) directAliyunStore(config M) (*ossStore, error) {
	if str(config["disk"]) != "aliyun" {
		return nil, clientError{"OSS 配置已变更，请重新上传", 409}
	}
	store, err := newObjectStore("aliyun", obj(config["aliyun"]))
	if err != nil {
		return nil, err
	}
	aliyun, ok := store.(*ossStore)
	if !ok {
		return nil, clientError{"OSS 配置无效", 503}
	}
	return aliyun, nil
}

func directObjectStatFromRange(result *oss.GetObjectResult) (directObjectStat, error) {
	if result == nil {
		return directObjectStat{}, errors.New("OSS range result is empty")
	}
	if result.ContentRange != nil {
		_, _, total, err := oss.ParseContentRange(*result.ContentRange)
		if err != nil || total < 1 {
			return directObjectStat{}, errors.New("OSS Content-Range is invalid")
		}
		return directObjectStat{ContentLength: total, ETag: result.ETag}, nil
	}
	if result.ContentLength < 1 {
		return directObjectStat{}, errors.New("OSS object size is invalid")
	}
	return directObjectStat{ContentLength: result.ContentLength, ETag: result.ETag}, nil
}

// ESA can reject signed HEAD requests for selected media extensions while the
// same object remains readable. A one-byte ranged GET is a safe verification
// fallback and Content-Range still exposes the full object size.
func statDirectObject(ctx context.Context, store *ossStore, key string) (directObjectStat, error) {
	head, headErr := store.client.HeadObject(ctx, &oss.HeadObjectRequest{Bucket: oss.Ptr(store.bucket), Key: oss.Ptr(key)})
	if headErr == nil {
		return directObjectStat{ContentLength: head.ContentLength, ETag: head.ETag}, nil
	}
	rangeHeader := "bytes=0-0"
	result, getErr := store.client.GetObject(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(store.bucket), Key: oss.Ptr(key), Range: &rangeHeader})
	if getErr != nil {
		return directObjectStat{}, fmt.Errorf("OSS HEAD failed: %v; ranged GET failed: %w", headErr, getErr)
	}
	defer result.Body.Close()
	stat, err := directObjectStatFromRange(result)
	if err != nil {
		return directObjectStat{}, fmt.Errorf("OSS HEAD failed: %v; ranged GET metadata failed: %w", headErr, err)
	}
	return stat, nil
}

func verifyDirectImage(ctx context.Context, store *ossStore, ticket directUploadTicket) (M, error) {
	rangeHeader := "bytes=0-2097151"
	result, err := store.client.GetObject(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(store.bucket), Key: oss.Ptr(ticket.Key), Range: &rangeHeader})
	if err != nil {
		return nil, clientError{"无法校验 OSS 图片", 502}
	}
	defer result.Body.Close()
	cfg, format, err := image.DecodeConfig(result.Body)
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		return nil, clientError{"图片格式无效或尺寸过大", 400}
	}
	if (format == "jpeg" && ticket.Ext != "jpg" && ticket.Ext != "jpeg") || (format == "png" && ticket.Ext != "png") || (format == "gif" && ticket.Ext != "gif") {
		return nil, clientError{"图片扩展名与内容不匹配", 400}
	}
	return imageMetadata(cfg.Width, cfg.Height), nil
}

func (a *App) directExistingMessage(r *request, message M) (any, bool, error) {
	id := str(message["id"])
	if id == "" {
		return nil, false, nil
	}
	to, group := target(message)
	if to == 0 {
		return nil, false, nil
	}
	existing, err := r.one("SELECT * FROM "+a.t("message")+" WHERE id=? AND from_user=? AND chat_identify=?", id, r.uid(), chatKey(r.uid(), to, group))
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	serialized, err := a.serialize(r.ctx(), existing)
	return serialized, true, err
}

func (a *App) completeDirectUpload(r *request) (any, error) {
	ticket, err := a.parseDirectUploadTicket(r.s("ticket"))
	if err != nil {
		return nil, err
	}
	if ticket.UID != r.uid() || !safeAsset(ticket.Key) || !strings.HasPrefix(ticket.Key, "storage/"+ticket.Kind+"/") {
		return nil, deny()
	}
	config := a.config(r.ctx(), "fileUpload")
	store, err := a.directAliyunStore(config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(r.ctx(), 15*time.Second)
	defer cancel()
	object, err := statDirectObject(ctx, store, ticket.Key)
	if err != nil {
		return nil, clientError{"OSS 文件尚未上传完成", 409}
	}
	if object.ContentLength < 1 || object.ContentLength > ticket.MaxSize || (ticket.DeclaredSize > 0 && object.ContentLength != ticket.DeclaredSize) {
		return nil, clientError{"OSS 文件大小校验失败", 400}
	}
	metadata := M{}
	if ticket.Kind == "image" {
		metadata, err = verifyDirectImage(ctx, store, ticket)
		if err != nil {
			return nil, err
		}
	}

	a.directUploadMu.Lock()
	defer a.directUploadMu.Unlock()
	rel := ticket.Key
	src := "/" + rel
	p, err := r.one("SELECT * FROM "+a.t("file")+" WHERE src=? AND user_id=? AND status=1 AND COALESCE(delete_time,0)=0 LIMIT 1", src, r.uid())
	if err == sql.ErrNoRows {
		name := strings.TrimSuffix(ticket.FileName, filepath.Ext(ticket.FileName))
		etag := ""
		if object.ETag != nil {
			candidate := strings.Trim(*object.ETag, "\"")
			if !strings.Contains(candidate, "-") && len(candidate) == 32 {
				etag = strings.ToLower(candidate)
			}
		}
		p = M{"cate": ticket.Category, "file_type": ticket.MIME, "name": name, "src": src, "size": object.ContentLength, "ext": ticket.Ext, "md5": etag, "user_id": r.uid(), "create_time": time.Now().Unix(), "status": 1}
		tx, beginErr := a.db.BeginTx(r.ctx(), nil)
		if beginErr != nil {
			return nil, beginErr
		}
		defer tx.Rollback()
		fid, insertErr := insert(r.ctx(), tx, a.t("file"), p)
		if insertErr == nil {
			_, insertErr = insert(r.ctx(), tx, a.t("imgo_object"), M{"file_id": fid, "disk": "aliyun", "object_key": rel})
		}
		if insertErr != nil {
			return nil, insertErr
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		p["file_id"] = fid
	} else if err != nil {
		return nil, err
	}
	p["videoInfo"] = metadata
	p["type"] = 2
	p["url"] = a.mediaPath(src)

	switch ticket.Action {
	case "uploadavatar":
		if err = update(r.ctx(), a.db, a.t("user"), M{"avatar": src}, "user_id=?", r.uid()); err != nil {
			return nil, err
		}
		return p["url"], nil
	case "uploademoji":
		if _, err = r.one("SELECT id FROM "+a.t("emoji")+" WHERE user_id=? AND file_id=? AND status=1 LIMIT 1", r.uid(), p["file_id"]); err == sql.ErrNoRows {
			_, err = insert(r.ctx(), a.db, a.t("emoji"), M{"user_id": r.uid(), "name": p["name"], "src": src, "file_id": p["file_id"], "type": 2, "create_time": time.Now().Unix(), "update_time": time.Now().Unix(), "status": 1})
		}
		return p["url"], err
	case "uploadimage":
		return p["url"], nil
	}

	message := obj(r.p["message"])
	if len(message) == 0 {
		return p, nil
	}
	if existing, found, existingErr := a.directExistingMessage(r, message); found || existingErr != nil {
		return existing, existingErr
	}
	message["file_id"] = p["file_id"]
	message["content"] = src
	message["type"] = ticket.Kind
	if ticket.Kind == "image" {
		message["extends"] = metadata
	}
	if ticket.Kind == "video" {
		supplied := obj(message["extends"])
		videoMetadata, claimed, claimErr := a.claimUploadedVideoCover(r, number(p["file_id"]), str(supplied["poster"]))
		if claimErr != nil {
			return nil, claimErr
		}
		if !claimed {
			videoMetadata = M{"poster": a.mediaPath("/static/common/img/video-placeholder.svg")}
		}
		message["extends"] = mergeVideoMetadata(videoMetadata, supplied)
	}
	return a.sendMessage(r, message)
}
