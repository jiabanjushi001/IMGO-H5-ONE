package server

import (
	"crypto/hmac"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"time"
)

// Legacy captcha route. Login clients that submit captcha also get validation.
func (a *App) captcha(c *gin.Context) {
	code := strings.ToUpper(randomID()[:6])
	key := randomID()
	a.put("captcha:"+key, code, 5*time.Minute)
	c.SetCookie("imgo_captcha", key, 300, "/", "", strings.HasPrefix(a.cfg.BaseURL, "https://"), true)
	img := image.NewRGBA(image.Rect(0, 0, 150, 48))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{238, 242, 250, 255}}, image.Point{}, draw.Src)
	d := font.Drawer{Dst: img, Src: image.NewUniform(color.RGBA{32, 55, 90, 255}), Face: basicfont.Face7x13, Dot: fixed.P(20, 30)}
	d.DrawString(strings.Join(strings.Split(code, ""), " "))
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-store")
	_ = png.Encode(c.Writer, img)
}
func (a *App) validCaptcha(c *gin.Context, code string) bool {
	key, e := c.Cookie("imgo_captcha")
	if e != nil {
		return false
	}
	expected := str(a.get("captcha:"+key, true))
	return expected != "" && hmac.Equal([]byte(expected), []byte(strings.ToUpper(code)))
}
