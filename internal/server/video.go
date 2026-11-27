package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"image"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Executables and paths are passed directly, never interpolated into a shell command.
func videoMetadata(ctx context.Context, input, output string) (M, error) {
	bin := os.Getenv("FFMPEG_BIN")
	probe := os.Getenv("FFPROBE_BIN")
	if bin == "" {
		return M{}, nil
	}
	if probe == "" {
		probe = filepath.Join(filepath.Dir(bin), "ffprobe")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	raw, e := exec.CommandContext(ctx, probe, "-v", "error", "-protocol_whitelist", "file,pipe", "-show_entries", "format=duration", "-of", "json", input).Output()
	if e != nil {
		return nil, clientError{"无法读取视频信息", 400}
	}
	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if e = json.Unmarshal(raw, &data); e != nil {
		return nil, e
	}
	duration, _ := strconv.ParseFloat(data.Format.Duration, 64)
	if duration <= 0 || math.IsInf(duration, 0) || math.IsNaN(duration) {
		return nil, clientError{"视频时长无效", 400}
	}
	if e = exec.CommandContext(ctx, bin, "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-ss", "0", "-i", input, "-frames:v", "1", "-threads", "2", "-vf", "scale='min(1280,iw)':-2", "-y", output).Run(); e != nil {
		return nil, clientError{"视频封面生成失败", 400}
	}
	f, e := os.Open(output)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	cfg, _, e := image.DecodeConfig(f)
	if e != nil {
		return nil, e
	}
	m := imageMetadata(cfg.Width, cfg.Height)
	m["duration"] = int64(math.Ceil(duration))
	return m, nil
}

const maxUploadedVideoCoverBytes int64 = 3 << 20

// saveUploadedVideoCover persists the first frame produced by the browser.
// This keeps video thumbnails available after refresh without requiring
// FFmpeg on every deployment host.
func (a *App) saveUploadedVideoCover(r *request, parentID int64) (M, bool, error) {
	header, err := r.c.FormFile("poster")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if header.Size <= 0 || header.Size > maxUploadedVideoCoverBytes {
		return nil, true, clientError{"视频封面无效或超过 3 MB", http.StatusBadRequest}
	}
	f, err := header.Open()
	if err != nil {
		return nil, true, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxUploadedVideoCoverBytes+1))
	if err != nil {
		return nil, true, err
	}
	if int64(len(b)) > maxUploadedVideoCoverBytes {
		return nil, true, clientError{"视频封面无效或超过 3 MB", http.StatusBadRequest}
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
		return nil, true, clientError{"视频封面格式无效", http.StatusBadRequest}
	}
	ext := "jpg"
	fileType := "image/jpeg"
	if format == "png" {
		ext, fileType = "png", "image/png"
	} else if format != "jpeg" {
		return nil, true, clientError{"视频封面仅支持 JPG 或 PNG", http.StatusBadRequest}
	}
	rel := filepath.ToSlash(filepath.Join("storage", "cover", time.Now().Format("2006-01-02"), randomID()+"."+ext))
	dest := filepath.Join(a.cfg.PublicDir, filepath.FromSlash(rel))
	if err = os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
		return nil, true, err
	}
	if err = os.WriteFile(dest, b, 0640); err != nil {
		return nil, true, err
	}
	_, err = insert(r.ctx(), a.db, a.t("file"), M{"parent_id": parentID, "cate": 2, "src": "/" + rel, "name": "cover", "ext": ext, "file_type": fileType, "size": len(b), "user_id": r.uid(), "create_time": time.Now().Unix(), "status": 1})
	if err != nil {
		_ = os.Remove(dest)
		return nil, true, err
	}
	metadata := imageMetadata(cfg.Width, cfg.Height)
	metadata["poster"] = a.mediaPath("/" + rel)
	return metadata, true, nil
}

// claimUploadedVideoCover links a separately uploaded image to the video.
// Android and the legacy H5 client use this path when their upload API can
// send only one binary file per request.
func (a *App) claimUploadedVideoCover(r *request, parentID int64, poster string) (M, bool, error) {
	poster = strings.TrimSpace(poster)
	if poster == "" {
		return nil, false, nil
	}
	normalized := a.mediaPath(poster)
	u, err := url.Parse(normalized)
	if err != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, false, nil
	}
	path := u.Path
	if !strings.HasPrefix(path, "/storage/image/") && !strings.HasPrefix(path, "/storage/cover/") {
		return nil, false, nil
	}
	file, err := one(
		r.ctx(),
		a.db,
		"SELECT file_id,src,cate,user_id,parent_id FROM "+a.t("file")+" WHERE src=? AND user_id=? AND cate=2 AND status=1 AND COALESCE(delete_time,0)=0 AND parent_id IN (0,?) LIMIT 1",
		path,
		r.uid(),
		parentID,
	)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if number(file["parent_id"]) == 0 {
		if err = update(
			r.ctx(),
			a.db,
			a.t("file"),
			M{"parent_id": parentID},
			"file_id=? AND user_id=? AND parent_id=0",
			number(file["file_id"]),
			r.uid(),
		); err != nil {
			return nil, false, err
		}
	}
	return M{"poster": a.mediaPath(str(file["src"]))}, true, nil
}

func mergeVideoMetadata(metadata, supplied M) M {
	if metadata == nil {
		metadata = M{}
	}
	for _, key := range []string{"duration", "width", "height"} {
		if number(metadata[key]) <= 0 && number(supplied[key]) > 0 {
			metadata[key] = number(supplied[key])
		}
	}
	return metadata
}

func (a *App) makeVideoCover(r *request, path string, parentID int64) (M, error) {
	tmp, e := os.CreateTemp("", "imgo-cover-*.jpg")
	if e != nil {
		return nil, e
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	metadata, e := videoMetadata(r.ctx(), path, name)
	if e != nil {
		return nil, e
	}
	if len(metadata) == 0 {
		return M{"poster": a.mediaPath("/static/common/img/video-placeholder.svg")}, nil
	}
	rel := filepath.ToSlash(filepath.Join("storage", "cover", time.Now().Format("2006-01-02"), randomID()+".jpg"))
	dest := filepath.Join(a.cfg.PublicDir, filepath.FromSlash(rel))
	if e = os.MkdirAll(filepath.Dir(dest), 0750); e != nil {
		return nil, e
	}
	b, e := os.ReadFile(name)
	if e != nil {
		return nil, e
	}
	if e = os.WriteFile(dest, b, 0640); e != nil {
		return nil, e
	}
	_, e = insert(r.ctx(), a.db, a.t("file"), M{"parent_id": parentID, "cate": 2, "src": "/" + rel, "name": "cover", "ext": "jpg", "file_type": "image/jpeg", "size": len(b), "user_id": r.uid(), "create_time": time.Now().Unix(), "status": 1})
	if e != nil {
		os.Remove(dest)
		return nil, e
	}
	metadata["poster"] = a.mediaPath("/" + rel)
	return metadata, nil
}

// Old installations shipped a portrait as the default poster. Normalize only
// that known placeholder; retain actual video thumbnails and their URLs.
func (a *App) videoPoster(poster string) string {
	u, err := url.Parse(poster)
	if poster == "" || (err == nil && (u.Path == "/static/common/img/video.png" || u.Path == "static/common/img/video.png")) {
		return a.mediaPath("/static/common/img/video-placeholder.svg")
	}
	return a.mediaPath(poster)
}
