package ui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ef1m03/kordon/internal/app"
	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/guard"
	"github.com/ef1m03/kordon/internal/procs"
)

//go:embed panel
var panelFiles embed.FS

// Server is the local API behind the Kordon panel. It listens on 127.0.0.1 only,
// and every request needs the random token from panel.json (readable by the user
// only), so neither another site in a browser nor another user can reach it.
type Server struct {
	G       *guard.Guard
	Stats   *app.Stats
	Version string
	// Optional hooks the tray provides.
	UpdateAvailable func() string
	InstallUpdate   func()

	token string
	addr  string
}

const cookieName = "kordon"

// PanelInfo is what `kordon panel` reads to find the server.
type PanelInfo struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func panelInfoPath() string { return filepath.Join(app.Dir(), "panel.json") }

// ReadPanelInfo returns the address of the running Kordon's panel.
func ReadPanelInfo() (PanelInfo, error) {
	var p PanelInfo
	b, err := os.ReadFile(panelInfoPath())
	if err != nil {
		return p, err
	}
	return p, json.Unmarshal(b, &p)
}

// Start listens on a random local port and serves until ctx ends.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	s.token = hex.EncodeToString(b)
	s.addr = ln.Addr().String()
	info, _ := json.Marshal(PanelInfo{URL: "http://" + s.addr + "/", Token: s.token})
	if err := os.WriteFile(panelInfoPath(), info, 0o600); err != nil {
		ln.Close()
		return err
	}

	mux := http.NewServeMux()
	static, _ := fs.Sub(panelFiles, "panel")
	mux.Handle("GET /", http.FileServer(http.FS(static)))
	mux.HandleFunc("GET /api/state", s.state)
	mux.HandleFunc("GET /api/countries", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, countryList()) })
	mux.HandleFunc("POST /api/do", s.do)

	srv := &http.Server{Handler: s.guardRequest(mux), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = os.Remove(panelInfoPath())
		_ = srv.Close()
	}()
	go func() { _ = srv.Serve(ln) }()
	return nil
}

// guardRequest checks the Host (against DNS rebinding), the token, and for
// POSTs the Origin (against cross-site requests).
func (s *Server) guardRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.addr {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if t := r.URL.Query().Get("t"); t != "" && s.tokenOK(t) {
			// First open: trade the token in the URL for a cookie and drop it from the address.
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: s.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		c, err := r.Cookie(cookieName)
		if err != nil || !s.tokenOK(c.Value) {
			http.Error(w, "Откройте панель из меню Kordon. Open the panel from the Kordon menu.", http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("Origin") != "http://"+s.addr {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) tokenOK(t string) bool {
	return subtle.ConstantTimeCompare([]byte(t), []byte(s.token)) == 1
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

// ---- state ----

type countryJSON struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Flag string `json:"flag"`
}

func countryOf(cc string) *countryJSON {
	if cc == "" {
		return nil
	}
	return &countryJSON{Code: cc, Name: CountryName(cc), Flag: Flag(cc)}
}

type ruleJSON struct {
	IPs       []string      `json:"ips"`
	Countries []countryJSON `json:"countries"`
}

func ruleOf(a config.Allow) ruleJSON {
	r := ruleJSON{IPs: append([]string{}, a.IPs...), Countries: []countryJSON{}}
	for _, cc := range a.Countries {
		r.Countries = append(r.Countries, *countryOf(cc))
	}
	return r
}

type appJSON struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Custom  bool      `json:"custom"`
	Browser bool      `json:"browser"`
	Match   []string  `json:"match,omitempty"`
	Enabled bool      `json:"enabled"`
	Running bool      `json:"running"`
	Status  string    `json:"status"`
	Rule    *ruleJSON `json:"rule"`
}

func panelStatus(s guard.Snapshot) string {
	if s.Paused() {
		return "paused"
	}
	return [...]string{"starting", "ok", "frozen", "blocked"}[s.Status]
}

func (s *Server) state(w http.ResponseWriter, _ *http.Request) {
	now := time.Now()
	snap := s.G.Snapshot(now)
	cfg := snap.Config
	running := map[string]bool{}
	for _, p := range snap.Running {
		running[p.AppID] = true
	}
	groupStatus := map[string]string{}
	for _, gr := range snap.Groups {
		st := [...]string{"starting", "ok", "frozen", "blocked"}[gr.Status]
		for _, a := range gr.Apps {
			groupStatus[a] = st
		}
	}
	var apps []appJSON
	for _, p := range procs.Presets {
		a := appJSON{ID: p.ID, Name: p.Name, Browser: p.Browser, Enabled: slices.Contains(cfg.Apps, p.ID), Running: running[p.ID], Status: groupStatus[p.Name]}
		if r, ok := cfg.AppRules[p.ID]; ok {
			rj := ruleOf(r)
			a.Rule = &rj
		}
		apps = append(apps, a)
	}
	for _, c := range cfg.CustomApps {
		id := strings.ToLower(c.Name)
		a := appJSON{ID: id, Name: c.Name, Custom: true, Match: c.Match, Enabled: true, Running: running[id], Status: groupStatus[c.Name]}
		if r, ok := cfg.AppRules[id]; ok {
			rj := ruleOf(r)
			a.Rule = &rj
		}
		apps = append(apps, a)
	}
	var hist []map[string]string
	for i := len(snap.History) - 1; i >= 0; i-- {
		e := snap.History[i]
		line := EventLine(e)
		hist = append(hist, map[string]string{"time": e.At.Local().Format("15:04:05"), "date": e.At.Local().Format("02.01"),
			"text": strings.TrimSpace(line[strings.Index(line, " ")+1:]), "kind": eventKind(e.Kind)})
	}
	var trusted []map[string]any
	for _, n := range cfg.TrustedNetworks {
		trusted = append(trusted, map[string]any{"name": n.Name, "id": n.ID, "current": n.ID == snap.NetID})
	}
	update := ""
	if s.UpdateAvailable != nil {
		update = s.UpdateAvailable()
	}
	pausedUntil := ""
	if snap.Paused() {
		if snap.PausedUntil.Sub(now) > 24*time.Hour {
			pausedUntil = "forever"
		} else {
			pausedUntil = snap.PausedUntil.Format("15:04")
		}
	}
	writeJSON(w, map[string]any{
		"lang": Lang, "goos": runtime.GOOS, "version": s.Version,
		"status": panelStatus(snap), "statusText": StatusLine(snap), "reason": reasonKey(snap.Reason), "reasonText": reasonText(snap.Reason),
		"graceLeft": int(snap.GraceLeft.Seconds()), "paused": snap.Paused(), "pausedUntil": pausedUntil,
		"ip": snap.IP, "country": countryOf(snap.Country), "ip6": snap.IP6, "country6": countryOf(snap.Country6),
		"others": snap.Others, "offline": snap.Offline, "allowedNow": snap.AllowedNow,
		"dns": snap.DNS, "dnsCountry": countryOf(snap.DNSCountry), "dnsLeak": snap.DNSLeak,
		"netId": snap.NetID, "network": snap.Network, "trusted": trusted,
		"apps": apps, "allow": ruleOf(cfg.Allow),
		"settings": map[string]any{
			"grace": int(cfg.Grace.D().Seconds()), "close_timeout": int(cfg.CloseTimeout.D().Seconds()),
			"check_interval": int(cfg.CheckInterval.D().Seconds()), "min_agree": cfg.MinAgree,
			"freeze_on_network_change": cfg.FreezeOnNetworkChange, "block_when_offline": cfg.BlockWhenOffline,
			"check_ipv6": cfg.CheckIPv6, "dns_check": cfg.DNSCheck, "notifications": cfg.Notifications,
			"menubar": cfg.MenuBar, "update_check": cfg.UpdateCheck, "language": cfg.Language,
		},
		"autostart": app.AutostartEnabled(),
		"stats":     map[string]any{"days": s.Stats.Last(14, now), "total": s.Stats.Total(), "session": snap.Counters},
		"history":   hist,
		"update":    update,
		"paths":     map[string]string{"config": app.ConfigPath(), "log": app.LogPath()},
	})
}

func reasonKey(r guard.Reason) string {
	return [...]string{"", "net", "ip", "offline", "v6", "unconf", "dns"}[r]
}

func eventKind(k guard.EventKind) string {
	switch k {
	case guard.EvFrozen, guard.EvDNSLeak:
		return "warn"
	case guard.EvBlocked, guard.EvKilledNew, guard.EvOffline:
		return "bad"
	case guard.EvResumed, guard.EvUnblocked, guard.EvAutoAllowed, guard.EvAllowedIP, guard.EvAllowedCountry, guard.EvTrusted, guard.EvOnline:
		return "good"
	}
	return "info"
}

// ---- actions ----

type action struct {
	Op        string          `json:"op"`
	Value     string          `json:"value"`
	App       string          `json:"app"`
	Minutes   int             `json:"minutes"`
	IPs       []string        `json:"ips"`
	Countries []string        `json:"countries"`
	Settings  json.RawMessage `json:"settings"`
}

func (s *Server) do(w http.ResponseWriter, r *http.Request) {
	var a action
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.apply(a); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) apply(a action) error {
	now := time.Now()
	g := s.G
	switch a.Op {
	case "allow_current_ip":
		g.AllowCurrentIP(now)
	case "allow_current_country":
		g.AllowCurrentCountry(now)
	case "add_ip", "remove_ip", "add_country", "remove_country":
		return s.editAllow(now, a)
	case "toggle_app":
		g.ToggleApp(now, a.App)
	case "set_rule":
		return s.setRule(now, a)
	case "remove_rule":
		g.Update(now, func(c *config.Config) { delete(c.AppRules, a.App) })
	case "remove_custom":
		g.Update(now, func(c *config.Config) {
			c.CustomApps = slices.DeleteFunc(c.CustomApps, func(x config.CustomApp) bool { return strings.ToLower(x.Name) == a.App })
			delete(c.AppRules, a.App)
		})
	case "add_custom":
		name := strings.TrimSpace(a.Value)
		var match []string
		for _, m := range a.IPs { // reused field: the match patterns
			if m = strings.TrimSpace(m); m != "" {
				match = append(match, m)
			}
		}
		if name == "" || len(match) == 0 {
			return errors.New(T("panel.err.custom"))
		}
		g.Update(now, func(c *config.Config) {
			c.CustomApps = append(c.CustomApps, config.CustomApp{Name: name, Match: match})
		})
	case "trust_network":
		if !g.TrustCurrentNetwork(now, a.Value) {
			return errors.New(T("panel.err.trust"))
		}
	case "untrust_network":
		g.Update(now, func(c *config.Config) {
			c.TrustedNetworks = slices.DeleteFunc(c.TrustedNetworks, func(n config.TrustedNetwork) bool { return n.ID == a.Value })
		})
	case "rename_network":
		g.Update(now, func(c *config.Config) {
			for i := range c.TrustedNetworks {
				if c.TrustedNetworks[i].ID == a.App && strings.TrimSpace(a.Value) != "" {
					c.TrustedNetworks[i].Name = strings.TrimSpace(a.Value)
				}
			}
		})
	case "pause":
		d := time.Duration(a.Minutes) * time.Minute
		if a.Minutes <= 0 {
			d = guard.Forever
		}
		g.Pause(now, d)
	case "unpause":
		g.Unpause(now)
	case "check_now":
		g.CheckNow(now)
	case "settings":
		return s.settings(now, a.Settings)
	case "autostart":
		return app.SetAutostart(a.Value == "on")
	case "open_config":
		return app.OpenFile(app.ConfigPath())
	case "open_log":
		return app.OpenFile(app.LogPath())
	case "report":
		return app.OpenURL(app.ReportURL(s.Version))
	case "open_url":
		if a.Value == app.RepoURL || a.Value == app.StudioURL || a.Value == app.TelegramURL || strings.HasPrefix(a.Value, app.RepoURL+"/") {
			return app.OpenURL(a.Value)
		}
		return errors.New("unknown link")
	case "install_update":
		if s.InstallUpdate == nil {
			return errors.New("no update")
		}
		go s.InstallUpdate()
	default:
		return errors.New("unknown op " + a.Op)
	}
	return nil
}

func (s *Server) editAllow(now time.Time, a action) error {
	v := strings.TrimSpace(a.Value)
	switch a.Op {
	case "add_ip":
		if _, err := config.ParseIPOrNet(v); err != nil {
			return err
		}
	case "add_country":
		v = strings.ToUpper(v)
		if len(v) != 2 {
			return errors.New(T("panel.err.country"))
		}
	}
	target := func(c *config.Config) *config.Allow {
		if a.App == "" {
			return &c.Allow
		}
		r := c.AppRules[a.App]
		return &r
	}
	s.G.Update(now, func(c *config.Config) {
		if a.App != "" && c.AppRules == nil {
			c.AppRules = map[string]config.Allow{}
		}
		al := target(c)
		switch a.Op {
		case "add_ip":
			al.AddIP(v)
		case "remove_ip":
			al.RemoveIP(v)
		case "add_country":
			al.AddCountry(v)
		case "remove_country":
			al.RemoveCountry(v)
		}
		if a.App != "" {
			c.AppRules[a.App] = *al
		}
	})
	return nil
}

func (s *Server) setRule(now time.Time, a action) error {
	rule := config.Allow{}
	for _, ip := range a.IPs {
		if _, err := config.ParseIPOrNet(ip); err != nil {
			return err
		}
		rule.AddIP(ip)
	}
	for _, cc := range a.Countries {
		rule.AddCountry(cc)
	}
	if rule.IsEmpty() {
		// A rule that allows nothing would keep the app frozen forever; start from the shared list.
		rule = s.G.Snapshot(now).Config.Allow
	}
	s.G.Update(now, func(c *config.Config) {
		if c.AppRules == nil {
			c.AppRules = map[string]config.Allow{}
		}
		c.AppRules[a.App] = rule
	})
	return nil
}

func (s *Server) settings(now time.Time, raw json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	num := func(k string, min int) (int, bool) {
		f, ok := m[k].(float64)
		if !ok {
			return 0, false
		}
		return max(int(f), min), true
	}
	var err error
	s.G.Update(now, func(c *config.Config) {
		if n, ok := num("grace", 0); ok {
			c.Grace = config.Duration(time.Duration(n) * time.Second)
		}
		if n, ok := num("close_timeout", 0); ok {
			c.CloseTimeout = config.Duration(time.Duration(n) * time.Second)
		}
		if n, ok := num("check_interval", 3); ok {
			c.CheckInterval = config.Duration(time.Duration(n) * time.Second)
		}
		if n, ok := num("min_agree", 1); ok {
			c.MinAgree = n
		}
		for k, dst := range map[string]*bool{
			"freeze_on_network_change": &c.FreezeOnNetworkChange, "block_when_offline": &c.BlockWhenOffline,
			"check_ipv6": &c.CheckIPv6, "notifications": &c.Notifications, "update_check": &c.UpdateCheck,
		} {
			if v, ok := m[k].(bool); ok {
				*dst = v
			}
		}
		for k, allowed := range map[string][]string{
			"dns_check": {"warn", "block", "off"}, "menubar": {"ip", "country", "icon"}, "language": {"auto", "ru", "en"},
		} {
			if v, ok := m[k].(string); ok {
				if !slices.Contains(allowed, v) {
					err = errors.New("bad " + k + ": " + strconv.Quote(v))
					continue
				}
				switch k {
				case "dns_check":
					c.DNSCheck = v
				case "menubar":
					c.MenuBar = v
				case "language":
					c.Language = v
					Lang = app.Language(v)
					config.CommentLang = Lang
				}
			}
		}
	})
	return err
}
