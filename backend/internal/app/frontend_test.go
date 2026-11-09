package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFrontendProxyDoesNotSwallowAPIRoutes(t *testing.T) {
	cookies := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies <- r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>frontend</html>"))
	}))
	defer upstream.Close()
	r := New(nil, Config{Origin: "http://localhost:8080", FrontendURL: upstream.URL})
	server := httptest.NewServer(r)
	defer server.Close()
	req, _ := http.NewRequest("GET", server.URL+"/assets/index.js", nil)
	req.Header.Set("Cookie", "hooklab_session=private")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	cookie := <-cookies
	if res.StatusCode != 200 || !strings.Contains(string(body), "frontend") || cookie != "" {
		t.Fatalf("frontend %d cookie=%s", res.StatusCode, cookie)
	}
	for _, path := range []string{"/api/missing", "/h"} {
		res, err = server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 404 {
			t.Fatalf("API fell through frontend %s %d", path, res.StatusCode)
		}
	}
}
func TestAllHTTPMethodsIncludingExtensions(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080"})
	a := register(t, r, db)
	token := randomToken()
	res, e := db.Exec("INSERT INTO endpoints(user_id,name,token) VALUES(?,?,?)", a.id, "all methods", token)
	if e != nil {
		t.Fatal(e)
	}
	id, _ := res.LastInsertId()
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE", "CONNECT", "BREW", "custom", "M!X"}
	for _, method := range methods {
		w := call(r, actor{}, method, "/h/"+token+"/events", "payload", false)
		if w.Code != 202 {
			t.Errorf("%s: %d", method, w.Code)
		}
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM events WHERE endpoint_id=?", id).Scan(&count)
	if count != len(methods) {
		t.Fatalf("persisted %d methods", count)
	}
}
