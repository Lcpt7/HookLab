package app

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashAndVerification(t *testing.T) {
	one := hashPassword("correct-password-123")
	two := hashPassword("correct-password-123")
	if one == two || !verifyPassword(one, "correct-password-123") || verifyPassword(one, "incorrect-password") || strings.Contains(one, "correct-password") {
		t.Fatal("password hashing failed")
	}
	for _, v := range []string{"", "$argon2id$v=19$m=1,t=1,p=1$bad$bad", "$argon2id$v=19$m=65536,t=3,p=2$bad$bad"} {
		if verifyPassword(v, "anything") {
			t.Fatal("invalid hash accepted")
		}
	}
}
func TestSessionExpiryCookieAndStorage(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080", Secure: true})
	a := register(t, r, db)
	if !a.cookie.Secure {
		t.Fatal("secure flag missing")
	}
	sum := sha256.Sum256([]byte(a.cookie.Value))
	var stored []byte
	var hash string
	if e := db.QueryRow("SELECT token_hash FROM sessions WHERE user_id=?", a.id).Scan(&stored); e != nil || string(stored) != string(sum[:]) {
		t.Fatal("session credential was not hashed")
	}
	db.QueryRow("SELECT password_hash FROM users WHERE id=?", a.id).Scan(&hash)
	if !verifyPassword(hash, "correct-horse-12345") {
		t.Fatal("password not Argon2id")
	}
	db.Exec("UPDATE sessions SET expires_at=? WHERE user_id=?", time.Now().UTC().Add(-time.Hour), a.id)
	w := call(r, a, "GET", "/api/auth/me", "", false)
	if w.Code != 401 {
		t.Fatal("expired session accepted")
	}
}
func TestTrustedProxyCaptureIP(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080", TrustedProxies: []string{"10.20.0.2"}})
	a := register(t, r, db)
	res, _ := db.Exec("INSERT INTO endpoints(user_id,name,token) VALUES(?,?,?)", a.id, "proxy", randomToken())
	id, _ := res.LastInsertId()
	var token string
	db.QueryRow("SELECT token FROM endpoints WHERE id=?", id).Scan(&token)
	for _, v := range []struct{ remote, forwarded, want string }{{"10.20.0.2:1234", "203.0.113.55", "203.0.113.55"}, {"203.0.113.11:1234", "1.1.1.1", "203.0.113.11"}} {
		req := httptest.NewRequest("POST", "/h/"+token, strings.NewReader("hello"))
		req.RemoteAddr = v.remote
		req.Header.Set("X-Forwarded-For", v.forwarded)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 202 {
			t.Fatal(w.Code)
		}
		var ip string
		db.QueryRow("SELECT source_ip FROM events WHERE endpoint_id=? ORDER BY id DESC LIMIT 1", id).Scan(&ip)
		if ip != v.want {
			t.Fatalf("source %s want %s", ip, v.want)
		}
	}
}
func TestCSRFAndLoginRotation(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080"})
	a := register(t, r, db)
	var email string
	db.QueryRow("SELECT email FROM users WHERE id=?", a.id).Scan(&email)
	w := call(r, a, "POST", "/api/endpoints", `{"name":"x"}`, true)
	if w.Code != 201 {
		t.Fatal(w.Code)
	}
	bad := a
	bad.csrf = "wrong"
	w = call(r, bad, "POST", "/api/endpoints", `{"name":"x"}`, true)
	if w.Code != 403 {
		t.Fatal("invalid CSRF accepted")
	}
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.example")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatal("cross-origin login")
	}
	w = call(r, a, "POST", "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"correct-horse-12345"}`, email), true)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value == a.cookie.Value {
		t.Fatal("session not rotated")
	}
	w = call(r, a, "GET", "/api/auth/me", "", false)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("old session still valid")
	}
}
