package server

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// callVideo exposes a mentor's or the super administrator's own uploaded
// videos as the selectable source library for a labelled recorded-video call.
// The file remains protected by the normal media authorization checks.
func (a *App) callVideo(r *request) (any, error) {
	enabled, err := a.preparedVideoEnabled(r.ctx(), a.db, r.uid())
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, deny()
	}
	if action(r) != "index" {
		return nil, r.fail("未知操作")
	}

	limit, offset := r.pagination()
	where := "user_id=? AND status=1 AND COALESCE(delete_time,0)=0 AND cate=4"
	count, err := r.one("SELECT COUNT(*) n FROM "+a.t("file")+" WHERE "+where, r.uid())
	if err != nil {
		return nil, err
	}
	r.count = number(count["n"])
	items, err := r.list("SELECT file_id,name,ext,src,size,create_time FROM "+a.t("file")+" WHERE "+where+" ORDER BY file_id DESC LIMIT ? OFFSET ?", r.uid(), limit, offset)
	if err != nil {
		return nil, err
	}
	available := make([]M, 0, len(items))
	for _, item := range items {
		ok, checkErr := a.callVideoAvailable(r.ctx(), item)
		if checkErr != nil {
			return nil, checkErr
		}
		if !ok {
			if r.count > 0 {
				r.count--
			}
			continue
		}
		name := strings.TrimSpace(str(item["name"]))
		ext := strings.TrimSpace(str(item["ext"]))
		if ext != "" && !strings.HasSuffix(strings.ToLower(name), "."+strings.ToLower(ext)) {
			name += "." + ext
		}
		item["name"] = name
		item["url"] = a.mediaPath(str(item["src"]))
		delete(item, "src")
		available = append(available, item)
	}
	return available, nil
}

func (a *App) preparedVideoEnabled(ctx context.Context, db DB, userID int64) (bool, error) {
	if userID == 1 {
		return true, nil
	}
	return a.mentorEnabled(ctx, db, userID)
}

func (a *App) validateRecordedCallVideo(r *request) (M, error) {
	if r.n("type") != 1 {
		return nil, r.fail("预录视频仅支持视频通话")
	}
	enabled, err := a.preparedVideoEnabled(r.ctx(), a.db, r.uid())
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, deny()
	}
	fileID := r.n("video_file_id")
	if fileID < 1 {
		return nil, r.fail("请选择预录视频")
	}
	file, err := r.one("SELECT file_id,user_id,cate,status,delete_time,src FROM "+a.t("file")+" WHERE file_id=? AND user_id=? AND cate=4 AND status=1 AND COALESCE(delete_time,0)=0", fileID, r.uid())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.fail("预录视频不存在或不属于当前账号")
	}
	if err != nil {
		return nil, err
	}
	available, err := a.callVideoAvailable(r.ctx(), file)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, r.fail("预设视频文件已丢失，请重新上传")
	}
	return file, nil
}

func (a *App) callVideoAvailable(ctx context.Context, file M) (bool, error) {
	rel := strings.TrimLeft(str(file["src"]), "/")
	if !strings.HasPrefix(rel, "storage/video/") || !safeAsset(rel) {
		return false, nil
	}
	root, err := filepath.Abs(a.cfg.PublicDir)
	if err != nil {
		return false, err
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return false, err
	}
	if !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return false, nil
	}
	info, err := os.Stat(full)
	if err == nil {
		return !info.IsDir(), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	_, err = one(ctx, a.db, "SELECT file_id FROM "+a.t("imgo_object")+" WHERE file_id=? AND disk<>'' AND object_key<>'' LIMIT 1", number(file["file_id"]))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}
