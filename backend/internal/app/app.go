package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Config struct {
	FrontendURL    string
	TrustedProxies []string
	lookup         lookupFunc
	dial           dialFunc
	Origin         string
	Secure         bool
	MaxBody        int64
}
type App struct {
	db           *sql.DB
	cfg          Config
	loginLimit   limiter
	captureLimit limiter
	replayLimit  limiter
	authSlots    chan struct{}
	dummyHash    string
}

func New(db *sql.DB, cfg Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if cfg.MaxBody == 0 {
		cfg.MaxBody = 1048576
	}
	a := &App{db: db, cfg: cfg, authSlots: make(chan struct{}, 4), dummyHash: hashPassword(randomToken())}
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		panic("invalid trusted proxy configuration")
	}
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store")
		c.Next()
		slog.Info("http request", "method", c.Request.Method, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
	})
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if db == nil || db.PingContext(ctx) != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	r.Use(a.capture)
	api := r.Group("/api", a.origin)
	api.POST("/auth/register", a.authLimit, a.register)
	api.POST("/auth/login", a.authLimit, a.login)
	private := api.Group("", a.auth)
	private.GET("/auth/me", a.me)
	private.POST("/auth/logout", a.logout)
	private.GET("/endpoints", a.endpoints)
	private.POST("/endpoints", a.createEndpoint)
	private.PATCH("/endpoints/:id", a.updateEndpoint)
	private.DELETE("/endpoints/:id", a.deleteEndpoint)
	private.GET("/events", a.events)
	private.GET("/events/:id", a.event)
	private.POST("/events/:id/replay", a.replay)
	private.GET("/events/:id/replays", a.replays)
	r.NoRoute(frontendFallback(cfg.FrontendURL))
	return r
}
func fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}
func serverError(c *gin.Context, e error) {
	slog.Error("request failed", "error", e)
	fail(c, 500, "服务暂时不可用，请稍后重试")
}
func noRows(e error) bool { return errors.Is(e, sql.ErrNoRows) }
func decode(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if e := dec.Decode(v); e != nil {
		fail(c, 400, "无效的 JSON 请求")
		return false
	}
	if e := dec.Decode(&struct{}{}); e != io.EOF {
		fail(c, 400, "请求必须是单个 JSON 对象")
		return false
	}
	return true
}
