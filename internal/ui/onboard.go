package ui

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/ef1m03/kordon/internal/app"
	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/procs"
)

// onboarding is the first setup on a fresh install, shown once in the tray
// window: is the current IP the VPN's, which running apps to protect, and
// whether to be counted among Kordon's users.
type onboarding struct {
	mu      sync.Mutex
	pending bool
	sug     []appRef
	sugAt   time.Time
}

type appRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func onboardPath() string { return filepath.Join(app.Dir(), "onboarded") }

// newOnboarding must run before the first state.json is written, like newWhatsNew:
// someone who already used Kordon doesn't go through the first setup.
func newOnboarding() *onboarding {
	o := &onboarding{}
	if _, err := os.Stat(onboardPath()); err == nil {
		return o
	}
	if _, err := os.Stat(app.StatePath()); err == nil {
		o.finish()
		return o
	}
	o.pending = true
	return o
}

func (o *onboarding) isPending() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.pending
}

// changed: the protected apps changed, so the suggestions are recomputed now.
func (o *onboarding) changed() {
	o.mu.Lock()
	o.sugAt = time.Time{}
	o.mu.Unlock()
}

func (o *onboarding) finish() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.pending = false
	_ = os.WriteFile(onboardPath(), []byte(time.Now().Format(time.RFC3339)), 0o644)
}

// suggest lists running apps from the presets that aren't protected yet.
// Browsers are left out: one is always running, and protecting it is a bigger choice.
func (o *onboarding) suggest(cfg *config.Config) []appRef {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.pending {
		return nil
	}
	if time.Since(o.sugAt) < 3*time.Second {
		return o.sug
	}
	o.sugAt = time.Now()
	all, err := procs.New().List()
	if err != nil {
		return o.sug
	}
	var tgts []procs.Target
	for _, p := range procs.Presets {
		if !p.Browser && !slices.Contains(cfg.Apps, p.ID) {
			tgts = append(tgts, procs.Target{ID: p.ID, Name: p.Name, Rule: p.Rule()})
		}
	}
	o.sug = []appRef{}
	seen := map[string]bool{}
	for _, p := range procs.Filter(all, tgts, os.Getpid()) {
		if !seen[p.AppID] {
			seen[p.AppID] = true
			o.sug = append(o.sug, appRef{ID: p.AppID, Name: p.App})
		}
	}
	slices.SortFunc(o.sug, func(a, b appRef) int {
		if a.Name < b.Name {
			return -1
		}
		return 1
	})
	return o.sug
}
