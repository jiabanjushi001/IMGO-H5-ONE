package server

import (
	"context"
	"database/sql"
	"fmt"
	"mime"
	"net/url"
	"path"
	"strings"
)

// Legacy voice and avatar uploads could have no file row. Register their existing
// database references without downloading or executing anything from the old server.
func (a *App) migrateLegacyMedia(ctx context.Context) error {
	ensure := func(tx *sql.Tx, src string, uid int64, size any) (int64, error) {
		src = strings.TrimPrefix(src, a.cfg.BaseURL)
		u, e := url.Parse(src)
		if e != nil || u.IsAbs() || u.Host != "" {
			return 0, nil
		}
		rel := strings.TrimLeft(u.Path, "/")
		if !isMediaPath(rel) || !safeAsset(rel) {
			return 0, nil
		}
		f, e := one(ctx, tx, "SELECT file_id FROM "+a.t("file")+" WHERE src IN (?,?) ORDER BY file_id LIMIT 1", "/"+rel, rel)
		if e == nil {
			return number(f["file_id"]), nil
		}
		if e != sql.ErrNoRows {
			return 0, e
		}
		ext := strings.TrimPrefix(strings.ToLower(path.Ext(rel)), ".")
		cate, _ := fileCategory(ext, false)
		name := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		return insert(ctx, tx, a.t("file"), M{"src": "/" + rel, "name": name, "ext": ext, "cate": cate, "file_type": mime.TypeByExtension("." + ext), "size": number(size), "user_id": uid, "status": 1})
	}
	cursor := int64(0)
	for {
		list, e := rows(ctx, a.db, "SELECT msg_id,content,from_user,file_size FROM "+a.t("message")+" WHERE msg_id>? AND COALESCE(file_id,0)=0 AND type IN ('voice','image','video','file') ORDER BY msg_id LIMIT 256", cursor)
		if e != nil {
			return e
		}
		if len(list) == 0 {
			break
		}
		for _, m := range list {
			cursor = number(m["msg_id"])
			src, e := decryptContent(a.cfg.ChatKey, str(m["content"]))
			if e != nil {
				return fmt.Errorf("历史附件消息 %d 解密失败，请检查 CHAT_KEY: %w", cursor, e)
			}
			tx, e := a.db.BeginTx(ctx, nil)
			if e != nil {
				return e
			}
			fid, e := ensure(tx, src, number(m["from_user"]), m["file_size"])
			if e == nil && fid > 0 {
				e = update(ctx, tx, a.t("message"), M{"file_id": fid}, "msg_id=? AND COALESCE(file_id,0)=0", cursor)
			}
			if e != nil {
				tx.Rollback()
				return e
			}
			if e = tx.Commit(); e != nil {
				return e
			}
		}
	}
	cursor = 0
	for {
		list, e := rows(ctx, a.db, "SELECT user_id,avatar FROM "+a.t("user")+" WHERE user_id>? AND avatar IS NOT NULL AND avatar<>'' ORDER BY user_id LIMIT 256", cursor)
		if e != nil {
			return e
		}
		if len(list) == 0 {
			break
		}
		for _, u := range list {
			cursor = number(u["user_id"])
			tx, e := a.db.BeginTx(ctx, nil)
			if e != nil {
				return e
			}
			_, e = ensure(tx, str(u["avatar"]), cursor, 0)
			if e != nil {
				tx.Rollback()
				return e
			}
			if e = tx.Commit(); e != nil {
				return e
			}
		}
	}
	return nil
}
