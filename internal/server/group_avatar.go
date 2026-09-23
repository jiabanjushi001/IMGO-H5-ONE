package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) groupAvatarURL(gid int64) string {
	return a.mediaPath(fmt.Sprintf("/avatar/group-%d/120/%d", gid, gid))
}
func (a *App) groupAvatar(c *gin.Context, gid int64) {
	uid, e := a.mediaUser(c)
	if e != nil {
		c.Status(401)
		return
	}
	if uid != 1 {
		if _, err := a.member(c.Request.Context(), a.db, gid, uid); err != nil {
			user, _ := c.Get("mediaUser")
			principal, _ := user.(M)
			if number(principal["admin_role_id"]) > 0 {
				scope, err := a.adminScope(c.Request.Context(), principal)
				if err == nil {
					err = a.requireScopedGroup(c.Request.Context(), a.db, scope, gid)
				}
				if err == nil {
					err = a.authorizeManage(c.Request.Context(), principal, "manage.groups")
				}
				if err != nil {
					c.Status(403)
					return
				}
			} else {
				admin, err := one(c.Request.Context(), a.db, "SELECT role FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND COALESCE(delete_time,0)=0", uid)
				if err != nil || number(admin["role"]) <= 0 {
					c.Status(403)
					return
				}
			}
		}
	}
	g, e := one(c.Request.Context(), a.db, "SELECT avatar FROM "+a.t("group")+" WHERE group_id=? AND status=1 AND COALESCE(delete_time,0)=0", gid)
	if e != nil {
		c.Status(404)
		return
	}
	if src := str(g["avatar"]); src != "" {
		f, err := one(c.Request.Context(), a.db, "SELECT * FROM "+a.t("file")+" WHERE src=? AND cate=2 AND status=1 AND COALESCE(delete_time,0)=0", src)
		if err != nil || !strings.HasPrefix(src, "/storage/image/") || !safeAsset(strings.TrimPrefix(src, "/")) {
			c.Status(404)
			return
		}
		if remote, err := a.objectURL(c.Request.Context(), f); err != nil {
			c.Status(502)
			return
		} else if remote != "" {
			c.Redirect(302, remote)
			return
		}
		root, err := filepath.EvalSymlinks(a.cfg.PublicDir)
		if err != nil {
			c.Status(404)
			return
		}
		root, _ = filepath.Abs(root)
		asset, err := filepath.EvalSymlinks(filepath.Join(root, strings.TrimPrefix(src, "/")))
		if err != nil || !strings.HasPrefix(asset, root+string(os.PathSeparator)) {
			c.Status(404)
			return
		}
		c.Header("Cache-Control", "private, no-store")
		c.File(asset)
		return
	}
	users, e := rows(c.Request.Context(), a.db, "SELECT u.user_id,u.realname,u.avatar FROM "+a.t("user")+" u JOIN "+a.t("group_user")+" gu ON gu.user_id=u.user_id WHERE gu.group_id=? AND gu.status=1 ORDER BY gu.role,gu.id LIMIT 9", gid)
	if e != nil || len(users) == 0 {
		c.Status(404)
		return
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 120, 120))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{238, 240, 244, 255}), image.Point{}, draw.Src)
	cols := 2
	if len(users) == 1 {
		cols = 1
	} else if len(users) > 4 {
		cols = 3
	}
	size := (120 - (cols+1)*3) / cols
	for i, u := range users {
		x := 3 + (i%cols)*(size+3)
		y := 3 + (i/cols)*(size+3)
		rect := image.Rect(x, y, x+size, y+size)
		tile := a.avatarTile(u, size)
		xdraw.CatmullRom.Scale(canvas, rect, tile, tile.Bounds(), draw.Over, nil)
	}
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "private, no-store")
	c.Status(200)
	_ = png.Encode(c.Writer, canvas)
}
func (a *App) avatarTile(u M, size int) image.Image {
	avatar := str(u["avatar"])
	if strings.HasPrefix(avatar, "/storage/") && safeAsset(strings.TrimPrefix(avatar, "/")) {
		root, e := filepath.EvalSymlinks(a.cfg.PublicDir)
		if e == nil {
			root, _ = filepath.Abs(root)
			p, e := filepath.EvalSymlinks(filepath.Join(root, strings.TrimPrefix(avatar, "/")))
			if e == nil && strings.HasPrefix(p, root+string(os.PathSeparator)) {
				if f, e := os.Open(p); e == nil {
					cfg, _, err := image.DecodeConfig(f)
					if err == nil && cfg.Width*cfg.Height < 40_000_000 {
						f.Seek(0, 0)
						img, _, err := image.Decode(f)
						f.Close()
						if err == nil {
							return img
						}
					} else {
						f.Close()
					}
				}
			}
		}
	}
	tile := image.NewRGBA(image.Rect(0, 0, size, size))
	n := number(u["user_id"])
	draw.Draw(tile, tile.Bounds(), image.NewUniform(color.RGBA{uint8(50 + n*13%100), uint8(90 + n*23%100), uint8(120 + n*17%100), 255}), image.Point{}, draw.Src)
	face := font.Face(basicfont.Face7x13)
	if a.avatarFont != nil {
		f, e := opentype.NewFace(a.avatarFont, &opentype.FaceOptions{Size: float64(size) * 0.55, DPI: 72})
		if e == nil {
			face = f
			defer f.Close()
		}
	}
	name := []rune(str(u["realname"]))
	text := "?"
	if len(name) > 0 {
		text = string(name[0])
	}
	d := font.Drawer{Dst: tile, Src: image.White, Face: face}
	width := d.MeasureString(text).Ceil()
	d.Dot = fixed.P((size-width)/2, (size+face.Metrics().Ascent.Ceil()-face.Metrics().Descent.Ceil())/2)
	d.DrawString(text)
	return tile
}
