package app

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"
	"net/http"
	"strings"
	"sync"
	"time"
)

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func hashPassword(password string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return "$argon2id$v=19$m=65536,t=3,p=2$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
}
func verifyPassword(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=65536,t=3,p=2" {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil || len(salt) != 16 {
		return false
	}
	key, e := base64.RawStdEncoding.DecodeString(parts[5])
	if e != nil || len(key) != 32 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(got, key) == 1
}
func digest(token string) []byte { v := sha256.Sum256([]byte(token)); return v[:] }
func (a *App) origin(c *gin.Context) {
	if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" && c.GetHeader("Origin") != a.cfg.Origin {
		fail(c, 403, "请求来源校验失败")
		return
	}
	c.Next()
}
func (a *App) auth(c *gin.Context) {
	token, e := c.Cookie("hooklab_session")
	if e != nil || len(token) != 64 {
		fail(c, 401, "请先登录")
		return
	}
	var uid uint64
	var csrf string
	e = a.db.QueryRowContext(c.Request.Context(), "SELECT user_id,csrf FROM sessions WHERE token_hash=? AND expires_at>UTC_TIMESTAMP(6)", digest(token)).Scan(&uid, &csrf)
	if e != nil {
		if !noRows(e) {
			serverError(c, e)
			return
		}
		fail(c, 401, "会话已过期，请重新登录")
		return
	}
	if c.Request.Method != "GET" && c.Request.Method != "HEAD" && subtle.ConstantTimeCompare([]byte(c.GetHeader("X-CSRF-Token")), []byte(csrf)) != 1 {
		fail(c, 403, "CSRF 校验失败")
		return
	}
	c.Set("uid", uid)
	c.Set("csrf", csrf)
	c.Next()
}
func (a *App) cookie(c *gin.Context, token string, age int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: "hooklab_session", Value: token, Path: "/", MaxAge: age, HttpOnly: true, Secure: a.cfg.Secure, SameSite: http.SameSiteLaxMode})
}

// Bounded fixed-window rate limiter; contains IP addresses only and is local to this process.
type bucket struct {
	count int
	until time.Time
}
type limiter struct {
	mu    sync.Mutex
	items map[string]bucket
}

func (l *limiter) allow(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.items == nil {
		l.items = map[string]bucket{}
	}
	if len(l.items) >= 10000 {
		for k, v := range l.items {
			if now.After(v.until) {
				delete(l.items, k)
			}
		}
		if len(l.items) >= 10000 {
			return false
		}
	}
	v := l.items[key]
	if now.After(v.until) {
		v = bucket{until: now.Add(window)}
	}
	v.count++
	l.items[key] = v
	return v.count <= max
}
func (a *App) authLimit(c *gin.Context) {
	if !a.loginLimit.allow(c.ClientIP(), 20, time.Minute) {
		c.Header("Retry-After", "60")
		fail(c, 429, "操作过于频繁，请稍后重试")
		return
	}
	select {
	case a.authSlots <- struct{}{}:
		defer func() { <-a.authSlots }()
		c.Next()
	default:
		fail(c, 429, "认证服务繁忙，请稍后重试")
	}
}
func redact(h http.Header) http.Header {
	out := h.Clone()
	for k := range out {
		if sensitive(k) {
			out[k] = []string{"[REDACTED]"}
		}
	}
	return out
}
func sensitive(k string) bool {
	k = strings.ToLower(k)
	for _, part := range []string{"authorization", "cookie", "token", "secret", "api-key", "apikey", "signature", "credential", "password", "auth"} {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}
func idParam(c *gin.Context, name string) (uint64, bool) {
	var id uint64
	_, e := fmt.Sscanf(c.Param(name), "%d", &id)
	if e != nil || id == 0 || fmt.Sprint(id) != c.Param(name) {
		fail(c, 400, "无效的资源 ID")
		return 0, false
	}
	return id, true
}
