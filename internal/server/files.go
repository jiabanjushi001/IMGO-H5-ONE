package server

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) upload(r *request) (any, error) {
	config := a.config(r.ctx(), "fileUpload")
	disk := str(config["disk"])
	file, e := r.c.FormFile("file")
	if e != nil {
		return nil, r.fail("缺少上传文件")
	}
	max := number(config["size"])
	if max < 1 || max > 50 {
		max = 50
	}
	if file.Size > max<<20 {
		return nil, r.fail("文件超过大小限制")
	}
	message := obj(r.p["message"])
	if filepath.Ext(file.Filename) == "" {
		if name := str(message["fileName"]); name != "" {
			file.Filename = name
		}
		if filepath.Ext(file.Filename) == "" && r.s("ext") != "" {
			file.Filename += "." + r.s("ext")
		}
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Filename), "."))
	if !safeAsset(file.Filename) || strings.ContainsAny(file.Filename, "/\\") {
		return nil, r.fail("文件名或扩展名不安全")
	}
	allowed := false
	switch extensions := config["fileExt"].(type) {
	case []any:
		for _, s := range extensions {
			if strings.EqualFold(str(s), ext) {
				allowed = true
			}
		}
	case []string:
		for _, s := range extensions {
			if strings.EqualFold(s, ext) {
				allowed = true
			}
		}
	}
	if strings.Contains("|html|htm|svg|js|mjs|css|exe|dll|com|bat|cmd|ps1|jar|", "|"+ext+"|") {
		allowed = false
	}
	if !allowed {
		return nil, r.fail("不允许上传此文件类型")
	}
	cate, kind := fileCategory(ext, str(message["type"]) == "voice")
	metadata := M{}
	act := action(r)
	if act == "uploadavatar" || act == "uploademoji" || act == "uploadimage" {
		if kind != "image" {
			return nil, r.fail("仅支持图片文件")
		}
	}
	if act == "uploademoji" && file.Size > 2<<20 {
		return nil, r.fail("表情不能超过 2 MB")
	}
	in, e := file.Open()
	if e != nil {
		return nil, e
	}
	defer in.Close()
	if kind == "image" {
		cfg, format, err := image.DecodeConfig(in)
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > 40_000_000 {
			return nil, r.fail("图片格式无效或尺寸过大")
		}
		if (format == "jpeg" && ext != "jpg" && ext != "jpeg") || (format == "png" && ext != "png") || (format == "gif" && ext != "gif") {
			return nil, r.fail("图片扩展名与内容不匹配")
		}
		metadata = imageMetadata(cfg.Width, cfg.Height)
		if _, e = in.Seek(0, 0); e != nil {
			return nil, e
		}
	}
	rel := filepath.ToSlash(filepath.Join("storage", kind, time.Now().Format("2006-01-02"), fmt.Sprint(r.uid()), randomID()+"."+ext))
	full := filepath.Join(a.cfg.PublicDir, filepath.FromSlash(rel))
	if e = os.MkdirAll(filepath.Dir(full), 0750); e != nil {
		return nil, e
	}
	out, e := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if e != nil {
		return nil, e
	}
	defer out.Close()
	hash := md5.New()
	written, e := io.Copy(io.MultiWriter(out, hash), io.LimitReader(in, (max<<20)+1))
	if e != nil || written > max<<20 {
		_ = os.Remove(full)
		return nil, r.fail("文件写入失败或超出限制")
	}
	if e = out.Close(); e != nil {
		_ = os.Remove(full)
		return nil, e
	}
	name := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	p := M{"cate": cate, "file_type": mime.TypeByExtension("." + ext), "name": name, "src": "/" + rel, "size": written, "ext": ext, "md5": fmt.Sprintf("%x", hash.Sum(nil)), "user_id": r.uid(), "create_time": time.Now().Unix(), "status": 1}
	if disk != "" && disk != "local" {
		if e = a.uploadCloud(r.ctx(), config, rel, full, str(p["file_type"])); e != nil {
			_ = os.Remove(full)
			return nil, e
		}
	}
	tx, e := a.db.BeginTx(r.ctx(), nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	fid, e := insert(r.ctx(), tx, a.t("file"), p)
	if e != nil {
		_ = os.Remove(full)
		return nil, e
	}
	if disk != "" && disk != "local" {
		if _, e = insert(r.ctx(), tx, a.t("imgo_object"), M{"file_id": fid, "disk": disk, "object_key": rel}); e != nil {
			return nil, e
		}
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	if disk != "" && disk != "local" {
		defer os.Remove(full)
	}
	p["file_id"] = fid
	p["videoInfo"] = metadata
	p["type"] = 2
	p["url"] = a.mediaPath("/" + rel)
	if act == "uploadavatar" {
		if e = update(r.ctx(), a.db, a.t("user"), M{"avatar": "/" + rel}, "user_id=?", r.uid()); e != nil {
			return nil, e
		}
		return p["url"], nil
	}
	if act == "uploademoji" {
		_, e = insert(r.ctx(), a.db, a.t("emoji"), M{"user_id": r.uid(), "name": name, "src": "/" + rel, "file_id": fid, "type": 2, "create_time": time.Now().Unix(), "update_time": time.Now().Unix(), "status": 1})
		return p["url"], e
	}
	if act == "uploadimage" {
		return p["url"], nil
	}
	if message := obj(r.p["message"]); len(message) > 0 {
		message["file_id"] = fid
		message["content"] = "/" + rel
		message["type"] = kind
		if kind == "video" {
			metadata, e = a.makeVideoCover(r, full, fid)
			if e != nil {
				return nil, e
			}
			message["extends"] = metadata
		}
		if kind == "image" {
			message["extends"] = metadata
		}
		return a.sendMessage(r, message)
	}
	return p, nil
}
func (a *App) emoji(r *request) (any, error) {
	switch action(r) {
	case "index":
		list, e := r.list("SELECT id,name,src,file_id FROM "+a.t("emoji")+" WHERE user_id=? AND type=2 AND status=1 ORDER BY update_time DESC,id DESC", r.uid())
		if e != nil {
			return nil, e
		}
		for _, m := range list {
			m["title"] = m["name"]
			m["src"] = a.mediaPath(str(m["src"]))
		}
		r.count = int64(len(list))
		return list, nil
	case "add":
		f, e := r.one("SELECT * FROM "+a.t("file")+" WHERE file_id=? AND status=1 AND cate=2", r.n("file_id"))
		if e != nil || !a.canReadFile(r.ctx(), r.uid(), f) {
			return nil, deny()
		}
		tx, e := a.db.BeginTx(r.ctx(), nil)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback()
		if _, e = one(r.ctx(), tx, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? FOR UPDATE", r.uid()); e != nil {
			return nil, e
		}
		if old, err := one(r.ctx(), tx, "SELECT id FROM "+a.t("emoji")+" WHERE user_id=? AND file_id=? LIMIT 1", r.uid(), f["file_id"]); err == nil {
			if e = update(r.ctx(), tx, a.t("emoji"), M{"status": 1, "update_time": time.Now().Unix()}, "id=?", old["id"]); e != nil {
				return nil, e
			}
			return M{"id": old["id"]}, tx.Commit()
		} else if err != sql.ErrNoRows {
			return nil, err
		}
		id, e := insert(r.ctx(), tx, a.t("emoji"), M{"user_id": r.uid(), "file_id": f["file_id"], "src": a.mediaPath(str(f["src"])), "name": f["name"], "type": 2, "status": 1, "create_time": time.Now().Unix(), "update_time": time.Now().Unix()})
		if e != nil {
			return nil, e
		}
		return M{"id": id}, tx.Commit()
	case "del", "move":
		ns := ids(r.p["ids"])
		if len(ns) == 0 {
			return nil, r.fail("请选择表情")
		}
		args := append([]any{r.uid()}, values(ns)...)
		q := "UPDATE " + a.t("emoji") + " SET status=0 WHERE user_id=? AND id IN (" + marks(len(ns)) + ")"
		if action(r) == "move" {
			q = "UPDATE " + a.t("emoji") + " SET update_time=? WHERE user_id=? AND id IN (" + marks(len(ns)) + ")"
			args = append([]any{time.Now().Unix()}, args...)
		}
		return nil, r.exec(q, args...)
	}
	return nil, r.fail("未知操作")
}
func (a *App) files(r *request) (any, error) {
	manageAll := r.n("is_all") == 1 || strings.HasPrefix(normalizedPath(r.c.Request.URL.Path), "/manage/files/")
	if r.n("is_all") == 1 {
		if err := a.authorizeManage(r.ctx(), r.user, "manage.files"); err != nil {
			return nil, err
		}
	}
	where := "f.status=1 AND COALESCE(f.delete_time,0)=0"
	args := []any{}
	if r.uid() != 1 && !manageAll {
		where += " AND (f.user_id=? OR EXISTS (SELECT 1 FROM " + a.t("message") + " m WHERE m.file_id=f.file_id AND m.status=1 AND (m.from_user=? OR (m.is_group=0 AND m.to_user=?) OR (m.is_group=1 AND m.to_user IN (SELECT group_id FROM " + a.t("group_user") + " WHERE user_id=? AND status=1)))))"
		args = append(args, r.uid(), r.uid(), r.uid(), r.uid())
	}
	if r.n("role") == 1 {
		where += " AND f.user_id=?"
		args = append(args, r.uid())
	}
	if r.n("role") == 2 {
		where += " AND f.user_id<>?"
		args = append(args, r.uid())
	}
	if r.n("cate") > 0 {
		where += " AND f.cate=?"
		args = append(args, r.n("cate"))
	}
	if r.s("keywords") != "" {
		where += " AND f.name LIKE ?"
		args = append(args, "%"+r.s("keywords")+"%")
	}
	n, e := r.one("SELECT COUNT(*) n FROM "+a.t("file")+" f WHERE "+where, args...)
	if e != nil {
		return nil, e
	}
	r.count = number(n["n"])
	limit, offset := r.pagination()
	list, e := r.list("SELECT f.* FROM "+a.t("file")+" f WHERE "+where+" ORDER BY f.file_id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if e != nil {
		return nil, e
	}
	for _, f := range list {
		ext := strings.ToLower(str(f["ext"]))
		f["name"] = str(f["name"]) + "." + ext
		f["src"] = a.mediaPath(str(f["src"]))
		f["preview"] = f["src"]
		f["download"] = a.mediaPath("/filedown/" + a.hashID(number(f["file_id"])))
		_, f["msg_type"] = fileCategory(ext, false)
		if str(f["msg_type"]) == "image" {
			f["extUrl"] = f["src"]
		} else {
			extURL := "/static/img/ext/" + strings.ToUpper(ext) + ".png"
			if _, err := os.Stat(filepath.Join(a.cfg.PublicDir, filepath.FromSlash(strings.TrimPrefix(extURL, "/")))); err != nil {
				extURL = "/static/img/ext/folder.png"
			}
			f["extUrl"] = extURL
		}
	}
	return list, nil
}
func (a *App) download(c *gin.Context) {
	id := a.decodeID(strings.TrimPrefix(c.Request.URL.Path, "/filedown/"))
	if id <= 0 {
		c.Status(404)
		return
	}
	f, e := one(c.Request.Context(), a.db, "SELECT * FROM "+a.t("file")+" WHERE file_id=? AND status=1 AND COALESCE(delete_time,0)=0", id)
	if e != nil {
		c.Status(404)
		return
	}
	uid, authErr := a.mediaUser(c)
	if authErr != nil {
		c.Status(401)
		return
	}
	if uid != 1 && !a.canReadFile(c.Request.Context(), uid, f) {
		c.Status(403)
		return
	}
	if remote, e := a.objectURL(c.Request.Context(), f); e != nil {
		c.Status(502)
		return
	} else if remote != "" {
		c.Redirect(302, remote)
		return
	}
	p := strings.TrimLeft(str(f["src"]), "/")
	if !strings.HasPrefix(p, "storage/") || !safeAsset(p) {
		c.Status(404)
		return
	}
	root, e := filepath.EvalSymlinks(a.cfg.PublicDir)
	if e == nil {
		root, e = filepath.Abs(root)
	}
	if e != nil {
		c.Status(500)
		return
	}
	resolved, e := filepath.EvalSymlinks(filepath.Join(root, p))
	if e != nil || !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		c.Status(404)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.FileAttachment(resolved, str(f["name"])+"."+str(f["ext"]))
}
func (a *App) avatar(c *gin.Context) {
	parts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	if len(parts) > 1 && strings.HasPrefix(parts[1], "group-") {
		a.groupAvatar(c, number(strings.TrimPrefix(parts[1], "group-")))
		return
	}
	name := "?"
	if len(parts) > 1 && parts[1] != "" {
		rs := []rune(parts[1])
		name = string(rs[0])
	}
	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "public, max-age=86400")
	c.String(http.StatusOK, `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="120" viewBox="0 0 120 120"><rect width="120" height="120" rx="24" fill="#5378d8"/><text x="60" y="78" text-anchor="middle" fill="white" font-family="sans-serif" font-size="60">%s</text></svg>`, html.EscapeString(name))
}
func (a *App) scan(c *gin.Context) {
	parts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	if len(parts) != 3 {
		c.Status(404)
		return
	}
	if c.Request.Method == "POST" {
		p, e := parseParams(c)
		if e != nil {
			c.JSON(200, M{"code": 400, "msg": "参数错误", "data": []any{}})
			return
		}
		p["action"] = parts[1]
		p["token"] = parts[2]
		r := &request{app: a, c: c, p: p}
		data, e := a.scanInfo(r)
		if e != nil {
			c.JSON(200, M{"code": 400, "msg": e.Error(), "data": []any{}})
			return
		}
		c.JSON(200, M{"code": 0, "msg": "", "data": data, "count": 0, "page": 1})
		return
	}
	switch parts[1] {
	case "u":
		uid := a.decodeID(parts[2])
		if uid == 0 {
			c.Status(404)
			return
		}
		c.Redirect(http.StatusFound, a.url("/#/index?user_id="+fmt.Sprint(uid)))
	case "g":
		if !a.validGroupToken(parts[2]) {
			c.String(400, "群邀请已失效")
			return
		}
		groupID := "group-" + strings.SplitN(parts[2], ".", 2)[0]
		c.Redirect(http.StatusFound, a.groupInviteURL(groupID, parts[2]))
	default:
		c.Status(404)
	}
}
