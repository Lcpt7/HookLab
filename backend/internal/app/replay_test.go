package app

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestReplayAddressPolicy(t *testing.T) {
	cases := []struct {
		ip    string
		allow bool
	}{{"8.8.8.8", true}, {"1.1.1.1", true}, {"2606:4700:4700::1111", true}, {"127.0.0.1", false}, {"::1", false}, {"10.1.1.1", false}, {"172.16.0.1", false}, {"192.168.1.1", false}, {"169.254.169.254", false}, {"100.100.100.200", false}, {"0.0.0.0", false}, {"192.0.2.1", false}, {"198.19.1.1", false}, {"203.0.113.1", false}, {"224.0.0.1", false}, {"255.255.255.255", false}, {"::ffff:127.0.0.1", false}, {"fc00::1", false}, {"fe80::1", false}, {"2001:db8::1", false}, {"2002:7f00:1::", false}, {"64:ff9b::7f00:1", false}, {"3fff::1", false}}
	for _, v := range cases {
		t.Run(v.ip, func(t *testing.T) {
			if got := publicIP(netip.MustParseAddr(v.ip)); got != v.allow {
				t.Fatalf("publicIP(%s)=%v want %v", v.ip, got, v.allow)
			}
		})
	}
}
func TestReplayURLPolicyAndDNS(t *testing.T) {
	ctx := context.Background()
	lookups := 0
	resolver := func(context.Context, string) ([]netip.Addr, error) {
		lookups++
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	for _, raw := range []string{"file:///etc/passwd", "ftp://example.com", "http://user:pass@example.com", "http://example.com:8080", "https://example.com:80", "http://127.0.0.1", "http://[::ffff:127.0.0.1]", "http://example.com/#secret", "https://example.com:444", "http://0x7f000001:22", "http://2130706433:22"} {
		if _, _, e := resolveTarget(ctx, raw, resolver); e == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	u, ips, e := resolveTarget(ctx, "https://api.example.com/webhook?x=1", resolver)
	if e != nil || u.Hostname() != "api.example.com" || len(ips) != 1 {
		t.Fatalf("valid target %v %v", u, e)
	}
	if lookups == 0 {
		t.Fatal("DNS was not validated")
	}
	mixed := func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")}, nil
	}
	if _, _, e = resolveTarget(ctx, "http://mixed.example", mixed); e == nil {
		t.Fatal("mixed DNS private address accepted")
	}
}
func TestReplayPinnedDialAndRedirect(t *testing.T) {
	hits := 0
	var gotBody string
	var auth, custom string
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		auth = r.Header.Get("Authorization")
		custom = r.Header.Get("X-Event-Type")
		w.Header().Set("Location", "http://127.0.0.1/private")
		w.WriteHeader(302)
	}))
	defer target.Close()
	local, _ := url.Parse(target.URL)
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "8.8.8.8:80" {
			redirected = true
			return nil, fmt.Errorf("unexpected dial %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, local.Host)
	}
	u, _ := url.Parse("http://receiver.example/hooks")
	client := replayClient(u, []netip.Addr{netip.MustParseAddr("8.8.8.8")}, dial)
	headers := http.Header{"Authorization": {"Bearer secret"}, "Content-Type": {"application/json"}, "X-Event-Type": {"payment"}, "Connection": {"X-Event-Type"}, "X-Forwarded-For": {"127.0.0.1"}}
	req, _ := http.NewRequest("PATCH", u.String(), strings.NewReader(`{"ok":true}`))
	req.Header = replayHeaders(headers)
	res, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	if redirected || hits != 1 || res.StatusCode != 302 || auth != "" || custom != "" || gotBody != `{"ok":true}` {
		t.Fatalf("unsafe replay hits=%d redirected=%v auth=%s custom=%s body=%s status=%d", hits, redirected, auth, custom, gotBody, res.StatusCode)
	}
	if client.Timeout > 10*time.Second || client.Timeout == 0 {
		t.Fatal("timeout missing")
	}
}
func TestReplayOwnershipAndPrivateTarget(t *testing.T) {
	db := testDB(t)
	r := New(db, Config{Origin: "http://localhost:8080", MaxBody: 1024})
	a := register(t, r, db)
	b := register(t, r, db)
	res, e := db.Exec("INSERT INTO endpoints(user_id,name,token) VALUES(?,?,?)", a.id, "replay", randomToken())
	if e != nil {
		t.Fatal(e)
	}
	eid, _ := res.LastInsertId()
	res, e = db.Exec("INSERT INTO events(endpoint_id,method,path,raw_query,headers,body,source_ip) VALUES(?,'POST','/','','{}',?,'127.0.0.1')", eid, []byte("hello"))
	if e != nil {
		t.Fatal(e)
	}
	id, _ := res.LastInsertId()
	path := fmt.Sprintf("/api/events/%d", id)
	w := call(r, b, "POST", path+"/replay", `{"target":"http://127.0.0.1"}`, true)
	if w.Code != 404 {
		t.Fatalf("replay ownership %d", w.Code)
	}
	w = call(r, b, "GET", path+"/replays", "", false)
	if w.Code != 404 {
		t.Fatalf("replay history ownership %d", w.Code)
	}
	w = call(r, a, "POST", path+"/replay", `{"target":"http://169.254.169.254/latest/meta-data"}`, true)
	if w.Code != 400 {
		t.Fatalf("private target accepted %d %s", w.Code, w.Body.String())
	}
	w = call(r, a, "GET", path+"/replays", "", false)
	if w.Code != 200 || len(object(t, w)["replays"].([]any)) != 0 {
		t.Fatalf("history %d %s", w.Code, w.Body.String())
	}
}

func TestReplayDeliveryAndPersistence(t *testing.T) {
	db := testDB(t)
	var gotMethod, gotQuery, gotBody, gotAuth string
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(204)
	}))
	defer receiver.Close()
	local, _ := url.Parse(receiver.URL)
	cfg := Config{Origin: "http://localhost:8080", MaxBody: 1024, lookup: func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}, dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "8.8.8.8:80" {
			return nil, fmt.Errorf("unsafe dial %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, local.Host)
	}}
	r := New(db, cfg)
	a := register(t, r, db)
	res, _ := db.Exec("INSERT INTO endpoints(user_id,name,token) VALUES(?,?,?)", a.id, "delivery", randomToken())
	eid, _ := res.LastInsertId()
	res, e := db.Exec("INSERT INTO events(endpoint_id,method,path,raw_query,headers,body,source_ip) VALUES(?,'PATCH','/old','original=private',?,?, '127.0.0.1')", eid, `{"Authorization":["[REDACTED]"],"Content-Type":["application/json"]}`, []byte(`{"amount":42}`))
	if e != nil {
		t.Fatal(e)
	}
	id, _ := res.LastInsertId()
	path := fmt.Sprintf("/api/events/%d", id)
	w := call(r, a, "POST", path+"/replay", `{"target":"http://public.example/receive?new=yes"}`, true)
	if w.Code != 201 {
		t.Fatalf("delivery %d %s", w.Code, w.Body.String())
	}
	if gotMethod != "PATCH" || gotQuery != "new=yes" || gotBody != `{"amount":42}` || gotAuth != "" {
		t.Fatalf("delivery corrupt %s %s %s %s", gotMethod, gotQuery, gotBody, gotAuth)
	}
	w = call(r, a, "GET", path+"/replays", "", false)
	records := object(t, w)["replays"].([]any)
	if len(records) != 1 || records[0].(map[string]any)["status_code"] != float64(204) {
		t.Fatalf("audit not persisted %s", w.Body.String())
	}
}
