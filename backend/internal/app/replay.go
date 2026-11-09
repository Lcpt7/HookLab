package app

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("3fff::/20"),
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.Zone() != "" {
		return false
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, p := range blockedPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

type lookupFunc func(context.Context, string) ([]netip.Addr, error)

func resolveTarget(ctx context.Context, raw string, lookup lookupFunc) (*url.URL, []netip.Addr, error) {
	invalid := errors.New("仅允许公网 HTTP/HTTPS 地址，端口限 80/443，不允许凭证或片段")
	if len(raw) > 2048 || strings.TrimSpace(raw) != raw {
		return nil, nil, invalid
	}
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.Opaque != "" || u.User != nil || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, nil, invalid
	}
	if strings.Contains(u.Hostname(), "%") || strings.HasSuffix(u.Host, ":") {
		return nil, nil, invalid
	}
	port := u.Port()
	if port != "" && !((u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443")) {
		return nil, nil, invalid
	}
	var ips []netip.Addr
	if ip, e := netip.ParseAddr(u.Hostname()); e == nil {
		ips = []netip.Addr{ip}
	} else {
		dnsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		ips, e = lookup(dnsCtx, u.Hostname())
		if e != nil {
			return nil, nil, errors.New("目标域名解析失败")
		}
	}
	if len(ips) == 0 || len(ips) > 32 {
		return nil, nil, invalid
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return nil, nil, invalid
		}
	}
	return u, ips, nil
}

type dialFunc func(context.Context, string, string) (net.Conn, error)

func replayClient(u *url.URL, ips []netip.Addr, dial dialFunc) *http.Client {
	if dial == nil {
		dial = (&net.Dialer{Timeout: 3 * time.Second}).DialContext
	}
	port := "80"
	if u.Scheme == "https" {
		port = "443"
	}
	expected := net.JoinHostPort(u.Hostname(), port)
	tr := &http.Transport{Proxy: nil, DisableKeepAlives: true, DisableCompression: true, MaxResponseHeaderBytes: 32768, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 5 * time.Second, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr != expected {
			return nil, errors.New("target changed")
		}
		var err error
		for _, ip := range ips {
			var conn net.Conn
			conn, err = dial(ctx, "tcp", net.JoinHostPort(ip.Unmap().String(), port))
			if err == nil {
				return conn, nil
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, err
	}}
	return &http.Client{Transport: tr, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func replayHeaders(in http.Header) http.Header {
	blocked := map[string]bool{"host": true, "connection": true, "proxy-connection": true, "keep-alive": true, "upgrade": true, "transfer-encoding": true, "trailer": true, "te": true, "content-length": true, "accept-encoding": true, "forwarded": true, "x-real-ip": true}
	for _, v := range in.Values("Connection") {
		for _, key := range strings.Split(v, ",") {
			blocked[strings.ToLower(strings.TrimSpace(key))] = true
		}
	}
	out := http.Header{}
	for k, values := range in {
		lower := strings.ToLower(k)
		if blocked[lower] || sensitive(k) || strings.HasPrefix(lower, "x-forwarded-") || strings.HasPrefix(lower, "proxy-") {
			continue
		}
		for _, v := range values {
			if v != "[REDACTED]" {
				out.Add(k, v)
			}
		}
	}
	return out
}

type replayRecord struct {
	ID         uint64    `json:"id"`
	Target     string    `json:"target"`
	StatusCode int       `json:"status_code"`
	DurationMS int64     `json:"duration_ms"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
}

func (a *App) replay(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	event, body, ok := a.loadEvent(c, id)
	if !ok {
		return
	}
	if !a.replayLimit.allow(c.ClientIP(), 30, time.Minute) {
		c.Header("Retry-After", "60")
		fail(c, 429, "重放过于频繁，请稍后重试")
		return
	}
	var in struct {
		Target string `json:"target"`
	}
	if !decode(c, &in) {
		return
	}
	lookup := a.cfg.lookup
	if lookup == nil {
		lookup = func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		}
	}
	u, ips, e := resolveTarget(c.Request.Context(), in.Target, lookup)
	if e != nil {
		fail(c, 400, e.Error())
		return
	}
	req, e := http.NewRequestWithContext(c.Request.Context(), event.Method, u.String(), bytes.NewReader(body))
	if e != nil {
		fail(c, 400, "无法构造重放请求")
		return
	}
	var headers http.Header
	if e = json.Unmarshal(event.Headers, &headers); e != nil {
		serverError(c, e)
		return
	}
	req.Header = replayHeaders(headers)
	start := time.Now()
	client := replayClient(u, ips, a.cfg.dial)
	defer client.CloseIdleConnections()
	response, sendErr := client.Do(req)
	record := replayRecord{Target: u.String(), CreatedAt: time.Now().UTC()}
	if sendErr != nil {
		record.Error = "网络连接失败、TLS 验证失败或请求超时"
	} else {
		record.StatusCode = response.StatusCode
		_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 65536))
		response.Body.Close()
		if readErr != nil {
			record.Error = "读取响应时超时或连接中断"
		}
	}
	record.DurationMS = time.Since(start).Milliseconds()
	// Save the outcome even if the browser disconnects after the external request.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, e := a.db.ExecContext(ctx, "INSERT INTO replays(event_id,target,status_code,duration_ms,error) VALUES(?,?,?,?,?)", id, record.Target, record.StatusCode, record.DurationMS, record.Error)
	if e != nil {
		serverError(c, e)
		return
	}
	rid, _ := res.LastInsertId()
	record.ID = uint64(rid)
	c.JSON(201, record)
}
func (a *App) replays(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	if _, _, ok = a.loadEvent(c, id); !ok {
		return
	}
	rows, e := a.db.QueryContext(c.Request.Context(), "SELECT id,target,status_code,duration_ms,error,created_at FROM replays WHERE event_id=? ORDER BY id DESC LIMIT 50", id)
	if e != nil {
		serverError(c, e)
		return
	}
	defer rows.Close()
	out := []replayRecord{}
	for rows.Next() {
		var v replayRecord
		if e = rows.Scan(&v.ID, &v.Target, &v.StatusCode, &v.DurationMS, &v.Error, &v.CreatedAt); e != nil {
			serverError(c, e)
			return
		}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		serverError(c, e)
		return
	}
	c.JSON(200, gin.H{"replays": out})
}
