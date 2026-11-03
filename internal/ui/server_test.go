package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ef1m03/kordon/internal/app"
	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/guard"
	"github.com/ef1m03/kordon/internal/procs"
)

type noProcs struct{}

func (noProcs) List() ([]procs.Proc, error) { return nil, nil }
func (noProcs) Freeze(procs.Proc) error     { return nil }
func (noProcs) Resume(procs.Proc) error     { return nil }
func (noProcs) Close(procs.Proc) error      { return nil }
func (noProcs) Kill(procs.Proc) error       { return nil }

func startServer(t *testing.T) (*Server, PanelInfo, *guard.Guard) {
	t.Setenv("KORDON_DIR", t.TempDir())
	cfg := config.Default()
	cfg.Allow.IPs = []string{"64.137.54.87"}
	g := guard.New(cfg, guard.Deps{Fingerprint: func() string { return "x" }, StartProbe: func(*config.Config) {},
		Procs: noProcs{}, SaveConfig: func(*config.Config) error { return nil }})
	g.Step(time.Now())
	g.HandleProbe(time.Now(), guard.ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s := &Server{G: g, Stats: app.LoadStats(), Version: "1.2.0"}
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	info, err := ReadPanelInfo()
	if err != nil {
		t.Fatal(err)
	}
	return s, info, g
}

func TestServerAuth(t *testing.T) {
	_, info, g := startServer(t)
	plain := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	// No token: refused.
	if r, _ := plain.Get(info.URL + "api/state"); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: %d", r.StatusCode)
	}
	// Wrong token: refused.
	if r, _ := plain.Get(info.URL + "?t=nope"); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token: %d", r.StatusCode)
	}
	// DNS rebinding: a foreign Host header is refused even with the token.
	req, _ := http.NewRequest("GET", info.URL+"?t="+info.Token, nil)
	req.Host = "evil.example:80"
	if r, _ := plain.Do(req); r.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign host: %d", r.StatusCode)
	}

	// The real flow: token in the URL -> cookie -> API works.
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	r, err := c.Get(info.URL + "?t=" + info.Token)
	if err != nil || r.StatusCode != 200 {
		t.Fatalf("open panel: %v %d", err, r.StatusCode)
	}
	r, _ = c.Get(info.URL + "api/state")
	var st map[string]any
	_ = json.NewDecoder(r.Body).Decode(&st)
	if st["status"] != "ok" || st["ip"] != "64.137.54.87" {
		t.Fatalf("state: %v", st)
	}

	// A POST from another origin (a site the user visits) is refused even with the cookie.
	post := func(origin, body string) int {
		req, _ := http.NewRequest("POST", info.URL+"api/do", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		r, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return r.StatusCode
	}
	if code := post("https://evil.example", `{"op":"pause","minutes":0}`); code != http.StatusForbidden {
		t.Fatalf("cross-site post: %d", code)
	}
	if g.Snapshot(time.Now()).Paused() {
		t.Fatal("cross-site post paused protection")
	}
	// From the panel itself it works.
	if code := post(strings.TrimSuffix(info.URL, "/"), `{"op":"add_country","value":"de"}`); code != 200 {
		t.Fatalf("panel post: %d", code)
	}
	if got := g.Snapshot(time.Now()).Config.Allow.Countries; len(got) != 1 || got[0] != "DE" {
		t.Fatalf("countries %v", got)
	}
	if code := post(strings.TrimSuffix(info.URL, "/"), `{"op":"add_ip","value":"not an ip"}`); code != http.StatusBadRequest {
		t.Fatalf("bad ip accepted: %d", code)
	}
}

func TestExtGuard(t *testing.T) {
	h := extGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }))
	try := func(host, origin string) (int, string) {
		r := httptest.NewRequest("GET", "/ext/v1/state", nil)
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code, w.Header().Get("Access-Control-Allow-Origin")
	}
	if c, cors := try("127.0.0.1:"+ExtPort, "chrome-extension://abc"); c != 200 || cors != "chrome-extension://abc" {
		t.Fatalf("extension: %d %q", c, cors)
	}
	if c, _ := try("127.0.0.1:"+ExtPort, "https://evil.example"); c != 403 {
		t.Fatalf("website origin: %d", c)
	}
	if c, _ := try("evil.example:"+ExtPort, ""); c != 403 {
		t.Fatalf("rebinding host: %d", c)
	}
}
