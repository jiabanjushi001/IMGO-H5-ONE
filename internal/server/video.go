package server

import (
	"context"
	"encoding/json"
	"image"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	seek := "1"
	if duration < 1 {
		seek = "0"
	}
	if e = exec.CommandContext(ctx, bin, "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-ss", seek, "-i", input, "-frames:v", "1", "-threads", "2", "-vf", "scale='min(1280,iw)':-2", "-y", output).Run(); e != nil {
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
