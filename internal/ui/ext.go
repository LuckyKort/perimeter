package ui

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ef1m03/kordon/internal/app"
)

// ExtPort is where the Kordon browser extension asks whether AI tabs must
// pause. A fixed port, because an extension can't read panel.json; it answers
// one read-only question and only to browser extensions.
const ExtPort = "47833"

// StartExt serves GET /ext/v1/state on 127.0.0.1:47833 until ctx ends.
// A busy port only means the extension can't reach this Kordon.
func (s *Server) StartExt(ctx context.Context) {
	ln, err := net.Listen("tcp4", "127.0.0.1:"+ExtPort)
	if err != nil {
		app.Logf("расширение для браузера: порт %s занят: %v", ExtPort, err)
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ext/v1/state", s.extState)
	srv := &http.Server{Handler: extGuard(mux), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	go func() { _ = srv.Serve(ln) }()
}

// extGuard: the Host must be this port (against DNS rebinding), and only
// extension pages get CORS, so a website open in the browser can't read it.
func extGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "127.0.0.1:"+ExtPort && r.Host != "localhost:"+ExtPort {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		o := r.Header.Get("Origin")
		if o != "" && !strings.HasPrefix(o, "chrome-extension://") && !strings.HasPrefix(o, "moz-extension://") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if o != "" {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) extState(w http.ResponseWriter, _ *http.Request) {
	snap := s.G.Snapshot(time.Now())
	// The shared rules decide for the browser: an app with its own rules
	// (Cursor only from NL, say) being frozen says nothing about AI tabs.
	st, reason := panelStatus(snap), snap.Reason
	if !snap.Paused() {
		for _, g := range snap.Groups {
			if g.Key == "" {
				st, reason = [...]string{"starting", "ok", "frozen", "blocked"}[g.Status], g.Reason
			}
		}
	}
	pause := st == "frozen" || st == "blocked" || st == "starting"
	why := ""
	if pause {
		why = reasonText(reason)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"kordon": s.Version,
		"status": st,
		// pause: what the protected apps do right now, the extension does with AI tabs.
		"pause":   pause,
		"reason":  why,
		"ip":      snap.IP,
		"country": countryOf(snap.Country),
		"ip6":     snap.IP6,
	})
}
