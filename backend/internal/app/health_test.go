package app

import (
	"net/http/httptest"
	"testing"
)

func TestHealthDoesNotNeedDatabase(t *testing.T) {
	r := New(nil, Config{Origin: "http://localhost:8080", MaxBody: 1048576})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 503 {
		t.Fatalf("readiness without database: %d", w.Code)
	}
}
