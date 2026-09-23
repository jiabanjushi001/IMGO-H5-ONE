package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/image/font/opentype"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Config struct{ ModerationToken, Addr, DSN, Prefix, JWTKey, ChatKey, PublicDir, BaseURL, QRBaseURL, H5URL, InviteURL, TrustedProxies, SMTPHost, SMTPUser, SMTPPass, SMTPFrom, APIID, APISecret string }

func EnvConfig() (Config, error) {
	c := Config{ModerationToken: os.Getenv("THINKAPI_TOKEN"), Addr: env("IMGO_ADDR", "127.0.0.1:8080"), DSN: os.Getenv("MYSQL_DSN"), Prefix: env("TABLE_PREFIX", "yu_"), JWTKey: os.Getenv("JWT_KEY"), ChatKey: os.Getenv("CHAT_KEY"), PublicDir: env("PUBLIC_DIR", "public"), BaseURL: strings.TrimRight(os.Getenv("BASE_URL"), "/"), QRBaseURL: strings.TrimSpace(os.Getenv("QR_BASE_URL")), H5URL: strings.TrimSpace(os.Getenv("H5_URL")), InviteURL: strings.TrimSpace(os.Getenv("INVITE_URL")), TrustedProxies: env("TRUSTED_PROXIES", "127.0.0.1,::1"), SMTPHost: os.Getenv("SMTP_ADDR"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPass: os.Getenv("SMTP_PASSWORD"), SMTPFrom: os.Getenv("SMTP_FROM"), APIID: os.Getenv("API_APP_ID"), APISecret: os.Getenv("API_SECRET")}
	if len(c.JWTKey) < 32 || strings.HasPrefix(c.JWTKey, "REPLACE_") {
		return c, errors.New("JWT_KEY 至少需要 32 字节随机密钥")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(c.Prefix) {
		return c, errors.New("TABLE_PREFIX 无效")
	}
	if c.DSN == "" {
		return c, errors.New("请设置 MYSQL_DSN")
	}
	return c, nil
}
func env(k, d string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return d
}

type cacheItem struct {
	Value  any
	Expiry time.Time
}
type App struct {
	metricsCancel context.CancelFunc
	maintenanceMu sync.Mutex
	avatarFont    *opentype.Font
	outbound      *http.Client
	ipdb          *ipDatabase
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	db            *sql.DB
	cfg           Config
	hub           *Hub
	mu            sync.Mutex
	cache         map[string]cacheItem
	routes        map[string]endpoint
	log           *slog.Logger
}
type endpoint struct {
	handler    func(*request) (any, error)
	public     bool
	super      bool
	permission string
}
type request struct {
	app    *App
	c      *gin.Context
	p      M
	user   M
	claims claims
	count  int64
	page   int64
}

func (r *request) ctx() context.Context { return r.c.Request.Context() }
func (r *request) uid() int64           { return number(r.user["user_id"]) }
func (r *request) s(k string) string    { return str(r.p[k]) }
func (r *request) n(k string) int64     { return number(r.p[k]) }
func (r *request) pagination() (int64, int64) {
	size := r.n("limit")
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	page := r.n("page")
	if page < 1 {
		page = 1
	}
	r.page = page
	return size, (page - 1) * size
}
func (r *request) list(q string, args ...any) ([]M, error) {
	return rows(r.ctx(), r.app.db, q, args...)
}
func (r *request) one(q string, args ...any) (M, error) { return one(r.ctx(), r.app.db, q, args...) }
func (r *request) exec(q string, args ...any) error {
	_, e := r.app.db.ExecContext(r.ctx(), q, args...)
	return e
}
func (r *request) fail(s string) error { return clientError{s, 400} }

type clientError struct {
	message string
	code    int
}

func (e clientError) Error() string { return e.message }
func deny() error                   { return clientError{"无权操作", 403} }
func New(c Config) (*App, error) {
	dc, e := mysql.ParseDSN(c.DSN)
	if e != nil {
		return nil, errors.New("MYSQL_DSN 格式错误")
	}
	dc.MultiStatements = false
	dc.ParseTime = true
	dc.Timeout = 5 * time.Second
	dc.ReadTimeout = 15 * time.Second
	dc.WriteTimeout = 15 * time.Second
	db, e := sql.Open("mysql", dc.FormatDSN())
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接失败: %w", e)
	}
	a := &App{db: db, cfg: c, cache: map[string]cacheItem{}, routes: map[string]endpoint{}, log: slog.Default()}
	a.ipdb, e = loadIPDatabase(os.Getenv("IP_DATABASE"))
	if e != nil {
		db.Close()
		return nil, e
	}
	if data, err := os.ReadFile(filepath.Join(c.PublicDir, "static/fonts/PingFangHeavy.ttf")); err == nil {
		a.avatarFont, _ = opentype.Parse(data)
	}
	a.hub = newHub(a)
	a.register()
	a.startMaintenance()
	return a, nil
}
func (a *App) Close() error {
	if a.metricsCancel != nil {
		a.metricsCancel()
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
	a.hub.close()
	return a.db.Close()
}
func (a *App) put(k string, v any, d time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, v := range a.cache {
		if now.After(v.Expiry) {
			delete(a.cache, k)
		}
	}
	a.cache[k] = cacheItem{v, now.Add(d)}
}
func (a *App) get(k string, consume bool) any {
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.cache[k]
	if !ok {
		return nil
	}
	if time.Now().After(v.Expiry) {
		delete(a.cache, k)
		return nil
	}
	if consume {
		delete(a.cache, k)
	}
	return v.Value
}
func (a *App) allow(k string, d time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if x, ok := a.cache[k]; ok && now.Before(x.Expiry) {
		return false
	}
	if len(a.cache) > 100000 {
		for k, v := range a.cache {
			if now.After(v.Expiry) {
				delete(a.cache, k)
			}
		}
		if len(a.cache) > 100000 {
			return false
		}
	}
	a.cache[k] = cacheItem{true, now.Add(d)}
	return true
}
func (a *App) config(ctx context.Context, name string) M {
	m, e := one(ctx, a.db, "SELECT value FROM "+a.t("config")+" WHERE name=? AND status=1", name)
	if e == nil {
		return obj(m["value"])
	}
	return defaultConfig(name)
}

func (a *App) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(gin.Recovery())
	trustedProxies := make([]string, 0, 2)
	for _, proxy := range strings.Split(a.cfg.TrustedProxies, ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			trustedProxies = append(trustedProxies, proxy)
		}
	}
	if len(trustedProxies) == 0 {
		trustedProxies = []string{"127.0.0.1", "::1"}
	}
	if e := g.SetTrustedProxies(trustedProxies); e != nil {
		panic("invalid TRUSTED_PROXIES")
	}
	g.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Content-Length, X-CSRF-Token, Token, session, sessionId, clientId, cid, X-Im-AppId, X-Im-TimeStamp, X-Im-Sign, X-Requested-With, Range")
		c.Header("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "600")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition, Content-Length, Content-Range, Accept-Ranges")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 60<<20)
		c.Next()
	})
	g.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if a.db.PingContext(ctx) != nil {
			c.JSON(503, M{"status": "unavailable"})
			return
		}
		c.JSON(200, M{"status": "ok"})
	})
	g.GET("/captcha", a.captcha)
	g.GET("/captcha/:config", a.captcha)
	g.GET("/wss", a.hub.serve)
	g.GET("/ws", a.hub.serve)
	g.NoRoute(a.dispatch)
	return g
}
func normalizedPath(s string) string { s = strings.TrimSuffix(s, "/"); return strings.ToLower(s) }
func parseParams(c *gin.Context) (M, error) {
	m := decodeForm(c.Request.URL.Query())
	ct := c.ContentType()
	if ct == "application/json" {
		dec := json.NewDecoder(c.Request.Body)
		dec.UseNumber()
		var body M
		if e := dec.Decode(&body); e != nil && !errors.Is(e, io.EOF) {
			return nil, clientError{"JSON 格式错误", 400}
		}
		for k, v := range body {
			m[k] = v
		}
	} else if ct == "application/x-www-form-urlencoded" || ct == "multipart/form-data" {
		var e error
		if ct == "multipart/form-data" {
			e = c.Request.ParseMultipartForm(8 << 20)
		} else {
			e = c.Request.ParseForm()
		}
		if e != nil {
			return nil, clientError{"请求体格式或大小不正确", 400}
		}
		for k, v := range decodeForm(c.Request.PostForm) {
			m[k] = v
		}
	}
	return m, nil
}
func (a *App) dispatch(c *gin.Context) {
	p := normalizedPath(c.Request.URL.Path)
	if p == "/index.php" && c.Query("s") != "" {
		p = normalizedPath(c.Query("s"))
	}
	if strings.HasPrefix(p, "/downloadapp/") {
		c.Request.URL.RawQuery = "platform=" + strings.TrimPrefix(p, "/downloadapp/")
		p = "/index/index/downloadapp"
	}
	ep, ok := a.routes[p]
	if !ok {
		a.asset(c)
		return
	}
	params, e := parseParams(c)
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	r := &request{app: a, c: c, p: params, page: 1}
	if e == nil && !ep.public {
		r.user, r.claims, e = a.authenticate(c.Request.Context(), c.GetHeader("Authorization"))
		if e == nil {
			maxAge := int(r.claims.Exp - time.Now().Unix())
			if maxAge > 0 {
				a.setMediaCookie(c, c.GetHeader("Authorization"), maxAge)
			}
		}
		if e == nil && ep.super && r.uid() != 1 {
			e = deny()
		}
		if e == nil && strings.HasPrefix(p, "/manage/") && !ep.super {
			e = a.authorizeManage(c.Request.Context(), r.user, ep.permission)
		}
	}
	var data any
	if e == nil {
		data, e = ep.handler(r)
	}
	if e != nil {
		code := 500
		msg := "服务处理失败"
		var ce clientError
		if errors.As(e, &ce) {
			code = ce.code
			msg = ce.message
		} else if errors.Is(e, sql.ErrNoRows) {
			code = 404
			msg = "记录不存在"
		} else {
			a.log.Error("request failed", "path", p, "error", e)
		}
		c.JSON(200, M{"code": code, "msg": msg, "data": []any{}, "count": 0, "page": r.page})
		return
	}
	if c.Writer.Written() {
		return
	}
	if data == nil {
		data = ""
	}
	c.JSON(200, M{"code": 0, "msg": "", "data": data, "count": r.count, "page": r.page})
}

// PHP CheckAuth uses HTTP 200 with business code -1 for invalid login sessions.
func (a *App) authenticate(ctx context.Context, token string) (M, claims, error) {
	token = strings.TrimSpace(token)
	if len(token) > 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	cl, e := parseToken(a.cfg.JWTKey, token)
	if e != nil {
		return nil, cl, clientError{"请重新登录", -1}
	}
	session, e := one(ctx, a.db, "SELECT user_id FROM "+a.t("imgo_session")+" WHERE sid=? AND user_id=? AND expires_at>?", cl.SID, cl.UID, time.Now().Unix())
	if e != nil || session == nil {
		return nil, cl, clientError{"请重新登录", -1}
	}
	u, e := one(ctx, a.db, "SELECT * FROM "+a.t("user")+" WHERE user_id=? AND status=1 AND delete_time=0", cl.UID)
	if e != nil {
		return nil, cl, clientError{"账号不可用，请重新登录", -1}
	}
	return u, cl, nil
}
func (a *App) asset(c *gin.Context) {
	p := strings.TrimPrefix(c.Request.URL.Path, "/")
	if strings.HasPrefix(p, "avatar/") && !strings.Contains(p, ".") {
		a.avatar(c)
		return
	}
	if strings.HasPrefix(p, "filedown/") {
		a.download(c)
		return
	}
	if strings.HasPrefix(p, "scan/") {
		a.scan(c)
		return
	}
	if p == "" {
		p = "index.html"
	}
	if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
		c.JSON(404, M{"code": 404, "msg": "接口不存在"})
		return
	}
	if !a.authorizeStorage(c, p) {
		return
	}
	if !safeAsset(p) {
		c.Status(404)
		return
	}
	root, e := filepath.EvalSymlinks(a.cfg.PublicDir)
	if e == nil {
		root, e = filepath.Abs(root)
	}
	if e != nil {
		c.Status(404)
		return
	}
	resolved, e := filepath.EvalSymlinks(filepath.Join(root, p))
	if e != nil || !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		c.Status(404)
		return
	}
	info, e := os.Stat(resolved)
	if e != nil || !info.Mode().IsRegular() {
		c.Status(404)
		return
	}
	c.File(resolved)
}
