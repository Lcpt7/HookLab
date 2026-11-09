package app

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type event struct {
	ID           uint64          `json:"id"`
	EndpointID   uint64          `json:"endpoint_id"`
	EndpointName string          `json:"endpoint_name"`
	Method       string          `json:"method"`
	Path         string          `json:"path"`
	RawQuery     string          `json:"raw_query"`
	Headers      json.RawMessage `json:"headers,omitempty"`
	BodyBase64   string          `json:"body_base64,omitempty"`
	Size         int             `json:"size"`
	SourceIP     string          `json:"source_ip"`
	ReceivedAt   time.Time       `json:"received_at"`
}

func (a *App) capture(c *gin.Context) {
	// Intercept here rather than Gin.Any, which only registers a fixed method list.
	if !strings.HasPrefix(c.Request.URL.Path, "/h/") {
		c.Next()
		return
	}
	c.Abort()
	parts := strings.SplitN(strings.TrimPrefix(c.Request.URL.Path, "/h/"), "/", 2)
	token := parts[0]
	if len(token) != 64 || len(c.Request.Method) > 32 {
		fail(c, 404, "Endpoint 不存在")
		return
	}
	if !a.captureLimit.allow(c.ClientIP(), 600, time.Minute) {
		c.Header("Retry-After", "60")
		fail(c, 429, "请求过于频繁")
		return
	}
	var eid uint64
	e := a.db.QueryRowContext(c.Request.Context(), "SELECT id FROM endpoints WHERE token=? AND active=TRUE", token).Scan(&eid)
	if noRows(e) {
		fail(c, 404, "Endpoint 不存在")
		return
	}
	if e != nil {
		serverError(c, e)
		return
	}
	path := "/"
	if len(parts) == 2 {
		path += parts[1]
	}
	if len(path) > 4096 || len(c.Request.URL.RawQuery) > 8192 {
		fail(c, 414, "URL 过长")
		return
	}
	body, e := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, a.cfg.MaxBody))
	if e != nil {
		if _, ok := e.(*http.MaxBytesError); ok {
			fail(c, 413, "请求体超出大小限制")
		} else {
			fail(c, 400, "无法读取请求体")
		}
		return
	}
	if body == nil {
		body = []byte{}
	}
	headersToStore := redact(c.Request.Header)
	headersToStore.Set("Host", c.Request.Host)
	headers, _ := json.Marshal(headersToStore)
	ip := c.ClientIP()
	res, e := a.db.ExecContext(c.Request.Context(), "INSERT INTO events(endpoint_id,method,path,raw_query,headers,body,source_ip) VALUES(?,?,?,?,?,?,?)", eid, c.Request.Method, path, c.Request.URL.RawQuery, headers, body, ip)
	if e != nil {
		serverError(c, e)
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(202, gin.H{"id": id, "status": "captured"})
}
func (a *App) events(c *gin.Context) {
	q := c.Query("q")
	if len(q) > 200 {
		fail(c, 400, "搜索最多 200 字节")
		return
	}
	method := c.Query("method")
	if len(method) > 32 {
		fail(c, 400, "无效的方法筛选")
		return
	}
	where := " WHERE e.user_id=?"
	args := []any{c.GetUint64("uid")}
	if s := c.Query("endpoint_id"); s != "" {
		id, e := strconv.ParseUint(s, 10, 64)
		if e != nil || id == 0 {
			fail(c, 400, "无效的 Endpoint ID")
			return
		}
		if !a.ownsEndpoint(c, id) {
			return
		}
		where += " AND v.endpoint_id=?"
		args = append(args, id)
	}
	if method != "" {
		where += " AND v.method=CAST(? AS BINARY)"
		args = append(args, method)
	}
	if q != "" {
		where += " AND (LOCATE(?,v.path)>0 OR LOCATE(?,CONVERT(v.body USING utf8mb4))>0)"
		args = append(args, q, q)
	}
	if s := c.Query("before"); s != "" {
		before, e := strconv.ParseUint(s, 10, 64)
		if e != nil || before == 0 {
			fail(c, 400, "无效的分页游标")
			return
		}
		where += " AND v.id<?"
		args = append(args, before)
	}
	rows, e := a.db.QueryContext(c.Request.Context(), "SELECT v.id,v.endpoint_id,e.name,v.method,v.path,v.raw_query,OCTET_LENGTH(v.body),v.source_ip,v.received_at FROM events v JOIN endpoints e ON e.id=v.endpoint_id"+where+" ORDER BY v.id DESC LIMIT 51", args...)
	if e != nil {
		serverError(c, e)
		return
	}
	defer rows.Close()
	out := []event{}
	for rows.Next() {
		var v event
		if e = rows.Scan(&v.ID, &v.EndpointID, &v.EndpointName, &v.Method, &v.Path, &v.RawQuery, &v.Size, &v.SourceIP, &v.ReceivedAt); e != nil {
			serverError(c, e)
			return
		}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		serverError(c, e)
		return
	}
	var next uint64
	if len(out) > 50 {
		out = out[:50]
		next = out[49].ID
	}
	c.JSON(200, gin.H{"events": out, "next_cursor": next})
}
func (a *App) loadEvent(c *gin.Context, id uint64) (event, []byte, bool) {
	var v event
	var body, headers []byte
	e := a.db.QueryRowContext(c.Request.Context(), "SELECT v.id,v.endpoint_id,e.name,v.method,v.path,v.raw_query,v.headers,v.body,v.source_ip,v.received_at FROM events v JOIN endpoints e ON e.id=v.endpoint_id WHERE v.id=? AND e.user_id=?", id, c.GetUint64("uid")).Scan(&v.ID, &v.EndpointID, &v.EndpointName, &v.Method, &v.Path, &v.RawQuery, &headers, &body, &v.SourceIP, &v.ReceivedAt)
	if noRows(e) {
		fail(c, 404, "事件不存在")
		return v, nil, false
	}
	if e != nil {
		serverError(c, e)
		return v, nil, false
	}
	v.Headers = headers
	v.Size = len(body)
	v.BodyBase64 = base64.StdEncoding.EncodeToString(body)
	return v, body, true
}
func (a *App) event(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	v, _, ok := a.loadEvent(c, id)
	if ok {
		c.JSON(200, v)
	}
}
