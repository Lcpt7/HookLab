package app

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPatchEndpointPreservesOmittedActive(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080"})
	a := register(t, r, db)
	w := call(r, a, "POST", "/api/endpoints", `{"name":"original"}`, true)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	id := fmt.Sprint(object(t, w)["id"])
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"rename active endpoint", `{"name":"renamed"}`, true},
		{"explicit false", `{"name":"paused","active":false}`, false},
		{"rename paused endpoint", `{"name":"still paused"}`, false},
		{"explicit true", `{"name":"resumed","active":true}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := call(r, a, "PATCH", "/api/endpoints/"+id, tc.body, true)
			if w.Code != 200 {
				t.Fatalf("patch: %d %s", w.Code, w.Body.String())
			}
			if got := object(t, w)["active"]; got != tc.want {
				t.Errorf("response active=%v, want %v", got, tc.want)
			}
			var got bool
			if err := db.QueryRow("SELECT active FROM endpoints WHERE id=? AND user_id=?", id, a.id).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("persisted active=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestCapturePersistsRequestHost(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080"})
	a := register(t, r, db)
	w := call(r, a, "POST", "/api/endpoints", `{"name":"host capture"}`, true)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	token := object(t, w)["token"].(string)
	req := httptest.NewRequest("POST", "/h/"+token+"/events", strings.NewReader(`{"ok":true}`))
	req.Host = "hooks.example.test:8443"
	req.Header.Set("Host", "not-the-request-host.example")
	req.Header.Set("Authorization", "Bearer do-not-store")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 202 {
		t.Fatalf("capture: %d %s", w.Code, w.Body.String())
	}
	id := fmt.Sprint(object(t, w)["id"])
	w = call(r, a, "GET", "/api/events/"+id, "", false)
	if w.Code != 200 {
		t.Fatalf("detail: %d %s", w.Code, w.Body.String())
	}
	headers := object(t, w)["headers"].(map[string]any)
	values, ok := headers["Host"].([]any)
	if !ok || len(values) != 1 || values[0] != "hooks.example.test:8443" {
		t.Fatalf("captured Host=%v, want actual Request.Host including port", headers["Host"])
	}
	if strings.Contains(w.Body.String(), "do-not-store") {
		t.Fatal("sensitive header was not redacted")
	}
	var persisted string
	if err := db.QueryRow("SELECT JSON_UNQUOTE(JSON_EXTRACT(headers, '$.Host[0]')) FROM events WHERE id=?", id).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != "hooks.example.test:8443" {
		t.Fatalf("persisted Host=%q", persisted)
	}
}

func TestMethodFilterIsCaseSensitive(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080"})
	a := register(t, r, db)
	w := call(r, a, "POST", "/api/endpoints", `{"name":"method case"}`, true)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	ep := object(t, w)
	id, token := fmt.Sprint(ep["id"]), ep["token"].(string)
	for _, method := range []string{"BREW", "brew"} {
		w := call(r, actor{}, method, "/h/"+token+"/events", "payload", false)
		if w.Code != 202 {
			t.Fatalf("capture %s: %d %s", method, w.Code, w.Body.String())
		}
	}
	for _, method := range []string{"BREW", "brew"} {
		t.Run(method, func(t *testing.T) {
			w := call(r, a, "GET", "/api/events?endpoint_id="+id+"&method="+method, "", false)
			if w.Code != 200 {
				t.Fatalf("filter: %d %s", w.Code, w.Body.String())
			}
			events := object(t, w)["events"].([]any)
			if len(events) != 1 {
				t.Fatalf("method=%s matched %d events, want exactly one", method, len(events))
			}
			if got := events[0].(map[string]any)["method"]; got != method {
				t.Fatalf("method=%s matched %v", method, got)
			}
		})
	}
}
