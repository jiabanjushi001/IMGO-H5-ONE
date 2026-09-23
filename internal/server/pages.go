package server

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) index(r *request) (any, error) {
	switch action(r) {
	case "index":
		r.c.Redirect(302, a.url("/index.html"))
		return nil, nil
	case "avatar":
		r.c.Request.URL.Path = "/avatar/" + url.PathEscape(r.s("str")) + "/" + r.s("s") + "/" + r.s("uid")
		a.avatar(r.c)
		r.c.Writer.WriteHeaderNow()
		return nil, nil
	case "download":
		r.c.Request.URL.Path = "/filedown/" + r.s("file_id")
		a.download(r.c)
		r.c.Writer.WriteHeaderNow()
		return nil, nil
	case "view":
		src := r.s("src")
		u, e := url.Parse(src)
		if e != nil {
			return nil, r.fail("文件地址无效")
		}
		base, _ := url.Parse(a.cfg.BaseURL)
		if u.Host != "" && u.Host != base.Host && u.Host != r.c.Request.Host {
			return nil, r.fail("只允许预览本站文件")
		}
		if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
			return nil, r.fail("地址协议无效")
		}
		if !strings.HasPrefix(u.Path, "/storage/") || !a.authorizeStorage(r.c, strings.TrimPrefix(u.Path, "/")) {
			if !r.c.Writer.Written() {
				return nil, r.fail("只允许预览已授权的文件")
			}
			return nil, nil
		}
		r.c.Header("Content-Type", "text/html; charset=utf-8")
		r.c.Header("Content-Security-Policy", "default-src 'self'; frame-src 'self'; style-src 'unsafe-inline'")
		r.c.String(http.StatusOK, `<!doctype html><meta charset="utf-8"><title>文件预览</title><p><a href="%s" download>下载文件</a></p><iframe sandbox style="width:100%%;height:90vh;border:0" src="%s"></iframe>`, html.EscapeString(src), html.EscapeString(src))
		return nil, nil
	case "downapp":
		r.c.Header("Content-Type", "text/html; charset=utf-8")
		r.c.String(200, `<!doctype html><meta charset="utf-8"><title>客户端下载</title><h1>客户端下载</h1><p><a href="/downloadApp/windows">Windows</a> · <a href="/downloadApp/mac">macOS</a> · <a href="/downloadApp/andriod">Android</a></p>`)
		return nil, nil
	case "downloadapp":
		platform := r.s("platform")
		if platform == "" {
			platform = "windows"
		}
		key := map[string]string{"windows": "APP_WINDOWS_FILE", "mac": "APP_MAC_FILE", "andriod": "APP_ANDROID_FILE", "android": "APP_ANDROID_FILE", "ios": "APP_IOS_FILE"}[platform]
		if key == "" {
			return nil, r.fail("平台无效")
		}
		p := os.Getenv(key)
		if p == "" {
			return nil, clientError{"管理员尚未配置此平台的安装包", 404}
		}
		st, e := os.Stat(p)
		if e != nil || !st.Mode().IsRegular() {
			return nil, clientError{"安装包不存在", 404}
		}
		r.c.FileAttachment(p, filepath.Base(p))
		return nil, nil
	case "scanqr":
		return a.scanInfo(r)
	}
	return nil, r.fail("未知页面")
}
func (a *App) scanInfo(r *request) (any, error) {
	kind := r.s("action")
	token := r.s("token")
	if kind == "u" {
		uid := a.decodeID(token)
		u, e := r.one("SELECT user_id,realname,avatar,name_py FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", uid)
		if e != nil {
			return nil, e
		}
		out := a.safeUser(u)
		out["action"] = "userInfo"
		return out, nil
	}
	if kind == "g" {
		if r.s("realToken") != "" {
			token = r.s("realToken")
		}
		v := strings.Split(token, ".")
		if !a.validGroupToken(token) {
			return nil, r.fail("邀请无效或已过期")
		}
		g, e := r.one("SELECT group_id,name,avatar,owner_id FROM "+a.t("group")+" WHERE group_id=? AND status=1", number(v[0]))
		if e != nil {
			return nil, e
		}
		g["id"] = "group-" + fmt.Sprint(number(v[0]))
		g["avatar"] = a.mediaPath(str(g["avatar"]))
		g["action"] = "groupInfo"
		g["token"] = token
		return g, nil
	}
	return nil, r.fail("二维码类型错误")
}
