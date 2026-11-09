package app

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"net/mail"
	"strings"
	"time"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *App) credentials(c *gin.Context) (credentials, bool) {
	var v credentials
	if !decode(c, &v) {
		return v, false
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	m, e := mail.ParseAddress(v.Email)
	if e != nil || m.Address != v.Email || len(v.Email) > 254 || len(v.Password) < 12 || len(v.Password) > 128 {
		fail(c, 400, "请输入有效邮箱，密码长度应为 12–128 字节")
		return v, false
	}
	return v, true
}
func (a *App) register(c *gin.Context) {
	v, ok := a.credentials(c)
	if !ok {
		return
	}
	hash := hashPassword(v.Password)
	res, e := a.db.ExecContext(c.Request.Context(), "INSERT INTO users(email,password_hash) VALUES(?,?)", v.Email, hash)
	if e != nil {
		var me *mysql.MySQLError
		if errors.As(e, &me) && me.Number == 1062 {
			fail(c, 409, "邮箱已注册")
			return
		}
		serverError(c, e)
		return
	}
	id, _ := res.LastInsertId()
	a.createSession(c, uint64(id), v.Email, 201)
}
func (a *App) login(c *gin.Context) {
	v, ok := a.credentials(c)
	if !ok {
		return
	}
	var id uint64
	var hash string
	e := a.db.QueryRowContext(c.Request.Context(), "SELECT id,password_hash FROM users WHERE email=?", v.Email).Scan(&id, &hash)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		serverError(c, e)
		return
	}
	if errors.Is(e, sql.ErrNoRows) {
		hash = a.dummyHash
	}
	if !verifyPassword(hash, v.Password) || id == 0 {
		fail(c, 401, "邮箱或密码不正确")
		return
	}
	a.createSession(c, id, v.Email, 200)
}
func (a *App) createSession(c *gin.Context, id uint64, email string, status int) {
	token, csrf := randomToken(), randomToken()
	tx, e := a.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		serverError(c, e)
		return
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(c.Request.Context(), "DELETE FROM sessions WHERE user_id=? AND expires_at<=UTC_TIMESTAMP(6)", id); e != nil {
		serverError(c, e)
		return
	}
	if old, err := c.Cookie("hooklab_session"); err == nil {
		if _, e = tx.ExecContext(c.Request.Context(), "DELETE FROM sessions WHERE token_hash=?", digest(old)); e != nil {
			serverError(c, e)
			return
		}
	}
	if _, e = tx.ExecContext(c.Request.Context(), "INSERT INTO sessions(token_hash,user_id,csrf,expires_at) VALUES(?,?,?,?)", digest(token), id, csrf, time.Now().UTC().Add(7*24*time.Hour)); e != nil {
		serverError(c, e)
		return
	}
	if e = tx.Commit(); e != nil {
		serverError(c, e)
		return
	}
	a.cookie(c, token, 604800)
	c.JSON(status, gin.H{"user": gin.H{"id": id, "email": email}, "csrf": csrf})
}
func (a *App) me(c *gin.Context) {
	var email string
	if e := a.db.QueryRowContext(c.Request.Context(), "SELECT email FROM users WHERE id=?", c.GetUint64("uid")).Scan(&email); e != nil {
		serverError(c, e)
		return
	}
	c.JSON(200, gin.H{"user": gin.H{"id": c.GetUint64("uid"), "email": email}, "csrf": c.GetString("csrf")})
}
func (a *App) logout(c *gin.Context) {
	token, _ := c.Cookie("hooklab_session")
	if _, e := a.db.ExecContext(c.Request.Context(), "DELETE FROM sessions WHERE token_hash=?", digest(token)); e != nil {
		serverError(c, e)
		return
	}
	a.cookie(c, "", -1)
	c.Status(204)
}
