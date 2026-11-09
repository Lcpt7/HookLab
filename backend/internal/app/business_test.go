package app

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type actor struct {
	cookie *http.Cookie
	csrf   string
	id     float64
}

func call(r http.Handler, a actor, method, path, body string, origin bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if origin {
		req.Header.Set("Origin", "http://localhost:8080")
	}
	if a.cookie != nil {
		req.AddCookie(a.cookie)
	}
	req.Header.Set("X-CSRF-Token", a.csrf)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func object(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid json %d %s", w.Code, w.Body.String())
	}
	return m
}
func register(t *testing.T, r http.Handler, db *sql.DB) actor {
	t.Helper()
	email := fmt.Sprintf("t%d@example.com", time.Now().UnixNano())
	w := call(r, actor{}, "POST", "/api/auth/register", fmt.Sprintf(`{"email":%q,"password":"correct-horse-12345"}`, email), true)
	if w.Code != 201 {
		t.Fatalf("register %d: %s", w.Code, w.Body.String())
	}
	m := object(t, w)
	id := m["user"].(map[string]any)["id"].(float64)
	t.Cleanup(func() { db.Exec("DELETE FROM users WHERE id=?", id) })
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("unsafe cookie")
	}
	return actor{cookies[0], m["csrf"].(string), id}
}
func TestBusinessIsolationAndCapture(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080", MaxBody: 1024})
	a := register(t, r, db)
	b := register(t, r, db)
	w := call(r, a, "POST", "/api/endpoints", `{"name":"payments"}`, true)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	ep := object(t, w)
	id := fmt.Sprint(ep["id"])
	token := ep["token"].(string)
	if len(token) != 64 {
		t.Fatal("token must have 256 bits")
	}
	for _, method := range []string{"PATCH", "DELETE"} {
		w = call(r, b, method, "/api/endpoints/"+id, `{"name":"stolen","active":false}`, true)
		if w.Code != 404 {
			t.Fatalf("ownership %s: %d", method, w.Code)
		}
	}
	w = call(r, actor{cookie: a.cookie}, "DELETE", "/api/endpoints/"+id, "", true)
	if w.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	w = call(r, a, "DELETE", "/api/endpoints/"+id, "", false)
	if w.Code != 403 {
		t.Fatal("missing origin accepted")
	}
	raw := []byte{'{', 0xff, 0, 10, '}'}
	req := httptest.NewRequest("PURGE", "/h/"+token+"/orders?x=1&x=2", bytes.NewReader(raw))
	req.RemoteAddr = "203.0.113.8:1234"
	req.Header.Set("Authorization", "Bearer TOP_SECRET")
	req.Header.Set("X-Api-Key", "TOP_SECRET")
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	req.Header.Set("Content-Type", "application/octet-stream")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 202 {
		t.Fatalf("capture %d %s", w.Code, w.Body.String())
	}
	w = call(r, a, "GET", "/api/events?endpoint_id="+id, "", false)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	list := object(t, w)["events"].([]any)
	if len(list) != 1 {
		t.Fatalf("events %v", list)
	}
	eid := fmt.Sprint(list[0].(map[string]any)["id"])
	w = call(r, a, "GET", "/api/events/"+eid, "", false)
	detail := object(t, w)
	if detail["body_base64"] != base64.StdEncoding.EncodeToString(raw) || detail["raw_query"] != "x=1&x=2" || detail["source_ip"] != "203.0.113.8" {
		t.Fatalf("capture corrupt %v", detail)
	}
	if strings.Contains(w.Body.String(), "TOP_SECRET") {
		t.Fatal("secret persisted")
	}
	for _, path := range []string{"/api/events/" + eid, "/api/events?endpoint_id=" + id} {
		w = call(r, b, "GET", path, "", false)
		if w.Code != 404 {
			t.Fatalf("events ownership %s: %d", path, w.Code)
		}
	}
	w = call(r, a, "GET", "/api/events?q=%27%20OR%201%3D1--", "", false)
	if len(object(t, w)["events"].([]any)) != 0 {
		t.Fatal("SQL search injection")
	}
	w = call(r, a, "GET", "/api/events?method=POST", "", false)
	if len(object(t, w)["events"].([]any)) != 0 {
		t.Fatal("method filter")
	}
	w = call(r, actor{}, "POST", "/h/"+token, strings.Repeat("x", 1025), false)
	if w.Code != 413 {
		t.Fatalf("body limit %d", w.Code)
	}
	w = call(r, a, "PATCH", "/api/endpoints/"+id, `{"name":"paused","active":false}`, true)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = call(r, actor{}, "POST", "/h/"+token, "hello", false)
	if w.Code != 404 {
		t.Fatal("paused endpoint accepted")
	}
	w = call(r, a, "DELETE", "/api/endpoints/"+id, "", true)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", eid).Scan(&n)
	if n != 0 {
		t.Fatal("cascade failed")
	}
	w = call(r, a, "POST", "/api/auth/logout", "", true)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	w = call(r, a, "GET", "/api/auth/me", "", false)
	if w.Code != 401 {
		t.Fatal("logout session still valid")
	}
}
func TestAuthValidation(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080", Secure: true, MaxBody: 1024})
	w := call(r, actor{}, "POST", "/api/auth/register", `{"email":"no","password":"short"}`, true)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	w = call(r, actor{}, "POST", "/api/auth/login", `{"email":"nobody@example.com","password":"wrong-password"}`, true)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	w = call(r, actor{}, "POST", "/api/auth/login", `{}`, false)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
