package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

func (a *App) setMediaCookie(c *gin.Context, token string, maxAge int) {
	token = strings.TrimSpace(token)
	if len(token) > 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	secure := c.Request.TLS != nil
	forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	if strings.EqualFold(forwarded, "https") {
		secure = true
	}
	if !secure {
		base, err := url.Parse(a.cfg.BaseURL)
		secure = err == nil && strings.EqualFold(base.Scheme, "https") && strings.EqualFold(base.Host, c.Request.Host)
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("imgo_media", token, maxAge, "/", "", secure, true)
}

// Browser media requests cannot set Authorization. A HttpOnly cookie authenticates GET media only.
func (a *App) mediaUser(c *gin.Context) (int64, error) {
	token := c.GetHeader("Authorization")
	if token == "" {
		token, _ = c.Cookie("imgo_media")
	}
	u, _, e := a.authenticate(c.Request.Context(), token)
	if e != nil {
		return 0, e
	}
	c.Set("mediaUser", u)
	return number(u["user_id"]), nil
}

// Media keeps the existing chat authorization and adds the current agent's uploader scope.
func (a *App) scopedMediaAccess(c *gin.Context, f M) bool {
	value, ok := c.Get("mediaUser")
	if !ok {
		c.AbortWithStatus(403)
		return false
	}
	user, ok := value.(M)
	if !ok {
		c.AbortWithStatus(403)
		return false
	}
	if number(user["user_id"]) == 1 || number(user["admin_role_id"]) == 0 {
		return true
	}
	scope, err := a.adminScope(c.Request.Context(), user)
	if err == nil && !scope.Global {
		err = a.requireScopedUser(c.Request.Context(), a.db, scope, number(f["user_id"]))
	}
	if err != nil {
		c.AbortWithStatus(403)
		return false
	}
	return true
}

func (a *App) canReadFile(ctx context.Context, uid int64, f M) bool {
	if number(f["parent_id"]) > 0 {
		parent, e := one(ctx, a.db, "SELECT * FROM "+a.t("file")+" WHERE file_id=? AND parent_id=0 AND status=1", f["parent_id"])
		if e != nil {
			return false
		}
		return a.canReadFile(ctx, uid, parent)
	}
	if number(f["user_id"]) == uid {
		return true
	}
	if _, err := one(ctx, a.db, "SELECT id FROM "+a.t("emoji")+" WHERE file_id=? AND user_id=? AND status=1 LIMIT 1", f["file_id"], uid); err == nil {
		return true
	}
	_, e := one(ctx, a.db, "SELECT m.msg_id FROM "+a.t("message")+" m WHERE m.file_id=? AND m.status=1 AND m.is_undo=0 AND NOT FIND_IN_SET(?,COALESCE(m.del_user,'')) AND (m.from_user=? OR (m.is_group=0 AND m.to_user=?) OR (m.is_group=1 AND EXISTS (SELECT 1 FROM "+a.t("group_user")+" gu JOIN "+a.t("group")+" g ON g.group_id=gu.group_id WHERE gu.user_id=? AND gu.group_id=m.to_user AND gu.status=1 AND g.status=1 AND (JSON_EXTRACT(IF(JSON_VALID(g.setting),g.setting,'{}'),'$.history') IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(IF(JSON_VALID(g.setting),g.setting,'{}'),'$.history')) IN ('1','true') OR m.create_time>=gu.create_time)))) LIMIT 1", f["file_id"], uid, uid, uid, uid)
	return e == nil
}
func (a *App) authorizeStorage(c *gin.Context, p string) bool {
	if !isMediaPath(p) {
		return true
	}
	f, e := one(c.Request.Context(), a.db, "SELECT * FROM "+a.t("file")+" WHERE (src=? OR src=?) AND status=1 AND COALESCE(delete_time,0)=0", "/"+p, p)
	if e != nil {
		c.Status(404)
		return false
	}

	// A configured site logo must render on the public login page.
	logo := str(a.config(c.Request.Context(), "sysInfo")["logo"])
	if number(f["cate"]) == 2 && (a.mediaPath(logo) == a.mediaPath(str(f["src"]))) {
		if remote, err := a.objectURL(c.Request.Context(), f); err != nil {
			c.Status(502)
			return false
		} else if remote != "" {
			c.Redirect(302, remote)
			return false
		}
		c.Header("Cache-Control", "public, max-age=300")
		return true
	}
	uid, e := a.mediaUser(c)
	if e != nil {
		c.Status(401)
		return false
	}
	if !a.scopedMediaAccess(c, f) {
		return false
	}
	// The system super administrator moderates all messages. Other accounts
	// still need chat/file access, with the existing avatar exception below.
	if uid != 1 && !a.canReadFile(c.Request.Context(), uid, f) { // Profile avatars are visible to authenticated users.
		_, err := one(c.Request.Context(), a.db, "SELECT user_id FROM "+a.t("user")+" WHERE avatar=? AND status=1 AND delete_time=0 LIMIT 1", "/"+p)
		if err != nil {
			c.Status(403)
			return false
		}
	}
	if remote, err := a.objectURL(c.Request.Context(), f); err != nil {
		c.Status(502)
		return false
	} else if remote != "" {
		c.Redirect(302, remote)
		return false
	}
	c.Header("Cache-Control", "private, no-store")
	return true
}

func isMediaPath(p string) bool {
	for _, prefix := range []string{"storage/", "image/", "file/", "video/", "voice/", "emoji/", "cover/", "avatar/"} {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
