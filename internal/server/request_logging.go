package server

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	requestIDContextKey     = "imgo_request_id"
	requestStageContextKey  = "imgo_request_stage"
	requestDetailContextKey = "imgo_request_error_detail"
	requestStartContextKey  = "imgo_request_started_at"
	requestUserContextKey   = "imgo_request_user_id"
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

func requestID(c *gin.Context) string {
	if value, exists := c.Get(requestIDContextKey); exists {
		if id := strings.TrimSpace(fmt.Sprint(value)); id != "" {
			return id
		}
	}
	return ""
}

func requestStage(c *gin.Context, fallback string) string {
	if value, exists := c.Get(requestStageContextKey); exists {
		if stage := strings.TrimSpace(fmt.Sprint(value)); stage != "" {
			return stage
		}
	}
	return fallback
}

func requestStarted(c *gin.Context) time.Time {
	if value, exists := c.Get(requestStartContextKey); exists {
		if started, ok := value.(time.Time); ok {
			return started
		}
	}
	return time.Now()
}

func cleanLogFilename(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	if chars := []rune(value); len(chars) > 160 {
		value = string(chars[:160])
	}
	return value
}

func uploadedFileHeader(c *gin.Context) *multipart.FileHeader {
	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return nil
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		return nil
	}
	return files[0]
}

func (a *App) requestLogAttrs(c *gin.Context, path string, userID int64, started time.Time) []any {
	if userID == 0 {
		if value, exists := c.Get(requestUserContextKey); exists {
			if contextualUserID, ok := value.(int64); ok {
				userID = contextualUserID
			}
		}
	}
	userAgent := strings.TrimSpace(c.GetHeader("User-Agent"))
	if len(userAgent) > 256 {
		userAgent = userAgent[:256]
	}
	attrs := []any{
		"request_id", requestID(c),
		"method", c.Request.Method,
		"path", path,
		"stage", requestStage(c, "dispatch"),
		"user_id", userID,
		"ip", a.clientIP(c),
		"content_type", c.ContentType(),
		"content_length", c.Request.ContentLength,
		"user_agent", userAgent,
		"duration_ms", time.Since(started).Milliseconds(),
	}
	if detail, exists := c.Get(requestDetailContextKey); exists && strings.TrimSpace(fmt.Sprint(detail)) != "" {
		attrs = append(attrs, "detail", fmt.Sprint(detail))
	}
	if file := uploadedFileHeader(c); file != nil {
		attrs = append(attrs,
			"file_name", cleanLogFilename(file.Filename),
			"file_size", file.Size,
			"file_content_type", file.Header.Get("Content-Type"),
		)
	}
	return attrs
}

func (a *App) requestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Set(requestStartContextKey, started)
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !validRequestID.MatchString(id) {
			id = randomID()[:20]
		}
		c.Set(requestIDContextKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
		if status := c.Writer.Status(); status >= 400 {
			attrs := a.requestLogAttrs(c, effectivePath(c), 0, started)
			attrs = append(attrs, "http_status", status)
			if len(c.Errors) > 0 {
				attrs = append(attrs, "gin_errors", c.Errors.String())
			}
			level := slog.LevelWarn
			if status >= 500 {
				level = slog.LevelError
			}
			a.log.Log(c.Request.Context(), level, "http request failed", attrs...)
		}
	}
}

func (a *App) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		attrs := a.requestLogAttrs(c, effectivePath(c), 0, requestStarted(c))
		attrs = append(attrs, "panic", fmt.Sprint(recovered), "stack", string(debug.Stack()))
		a.log.Error("request panic", attrs...)
		if !c.Writer.Written() {
			c.AbortWithStatusJSON(200, M{
				"code":       500,
				"msg":        "服务处理异常，请联系管理员",
				"request_id": requestID(c),
				"data":       []any{},
				"count":      0,
				"page":       1,
			})
			return
		}
		c.Abort()
	})
}

func (a *App) logRequestFailure(r *request, path string, started time.Time, code int, publicMessage string, err error) {
	attrs := a.requestLogAttrs(r.c, path, r.uid(), started)
	attrs = append(attrs, "business_code", code, "message", publicMessage, "error", err)
	level := slog.LevelWarn
	if code < 0 {
		// An expired browser session is normal lifecycle noise. Keep it available
		// to debug-level handlers without flooding production error logs.
		level = slog.LevelDebug
	} else if code >= 500 {
		level = slog.LevelError
	}
	a.log.Log(r.ctx(), level, "request failed", attrs...)
}
