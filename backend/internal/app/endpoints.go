package app

import (
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

type endpoint struct {
	ID         uint64    `json:"id"`
	Name       string    `json:"name"`
	Token      string    `json:"token"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	EventCount uint64    `json:"event_count"`
}

func (a *App) endpoints(c *gin.Context) {
	rows, e := a.db.QueryContext(c.Request.Context(), "SELECT e.id,e.name,e.token,e.active,e.created_at,(SELECT COUNT(*) FROM events v WHERE v.endpoint_id=e.id) FROM endpoints e WHERE e.user_id=? ORDER BY e.id DESC", c.GetUint64("uid"))
	if e != nil {
		serverError(c, e)
		return
	}
	defer rows.Close()
	out := []endpoint{}
	for rows.Next() {
		var v endpoint
		if e = rows.Scan(&v.ID, &v.Name, &v.Token, &v.Active, &v.CreatedAt, &v.EventCount); e != nil {
			serverError(c, e)
			return
		}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		serverError(c, e)
		return
	}
	c.JSON(200, gin.H{"endpoints": out})
}
func (a *App) createEndpoint(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(c, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 80 {
		fail(c, 400, "名称长度应为 1–80 字节")
		return
	}
	token := randomToken()
	res, e := a.db.ExecContext(c.Request.Context(), "INSERT INTO endpoints(user_id,name,token) SELECT ?,?,? WHERE (SELECT COUNT(*) FROM endpoints WHERE user_id=?)<100", c.GetUint64("uid"), in.Name, token, c.GetUint64("uid"))
	if e != nil {
		serverError(c, e)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(c, 409, "最多创建 100 个 Endpoint")
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(201, endpoint{ID: uint64(id), Name: in.Name, Token: token, Active: true, CreatedAt: time.Now().UTC()})
}
func (a *App) updateEndpoint(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var in struct {
		Name   string `json:"name"`
		Active *bool  `json:"active"`
	}
	if !decode(c, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 80 {
		fail(c, 400, "名称长度应为 1–80 字节")
		return
	}
	if !a.ownsEndpoint(c, id) {
		return
	}
	_, e := a.db.ExecContext(c.Request.Context(), "UPDATE endpoints SET name=?,active=COALESCE(?,active) WHERE id=? AND user_id=?", in.Name, in.Active, id, c.GetUint64("uid"))
	if e != nil {
		serverError(c, e)
		return
	}
	var active bool
	if e = a.db.QueryRowContext(c.Request.Context(), "SELECT active FROM endpoints WHERE id=? AND user_id=?", id, c.GetUint64("uid")).Scan(&active); e != nil {
		if noRows(e) {
			fail(c, 404, "Endpoint 不存在")
		} else {
			serverError(c, e)
		}
		return
	}
	c.JSON(200, gin.H{"id": id, "name": in.Name, "active": active})
}
func (a *App) deleteEndpoint(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	res, e := a.db.ExecContext(c.Request.Context(), "DELETE FROM endpoints WHERE id=? AND user_id=?", id, c.GetUint64("uid"))
	if e != nil {
		serverError(c, e)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(c, 404, "Endpoint 不存在")
		return
	}
	c.Status(204)
}
func (a *App) ownsEndpoint(c *gin.Context, id uint64) bool {
	var n int
	e := a.db.QueryRowContext(c.Request.Context(), "SELECT 1 FROM endpoints WHERE id=? AND user_id=?", id, c.GetUint64("uid")).Scan(&n)
	if noRows(e) {
		fail(c, 404, "Endpoint 不存在")
		return false
	}
	if e != nil {
		serverError(c, e)
		return false
	}
	return true
}
