package app

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Go remains the public HTTP entry point so extension methods reach capture
// unchanged. Nginx is an internal static asset server only.
func frontendFallback(raw string) gin.HandlerFunc {
	if raw == "" {
		return func(c *gin.Context) { fail(c, 404, "资源不存在") }
	}
	target, err := url.Parse(raw)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		panic("invalid frontend URL")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) { director(req); req.Header.Del("Cookie"); req.Header.Del("Authorization") }
	proxy.Transport = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 32768}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("frontend unavailable", "error", err)
		http.Error(w, "Frontend temporarily unavailable", http.StatusServiceUnavailable)
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || path == "/h" || strings.HasPrefix(path, "/h/") || (c.Request.Method != "GET" && c.Request.Method != "HEAD") {
			fail(c, 404, "资源不存在")
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
