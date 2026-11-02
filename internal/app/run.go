package app

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/guard"
	"github.com/ef1m03/kordon/internal/netwatch"
	"github.com/ef1m03/kordon/internal/procs"
)

const (
	tick        = 500 * time.Millisecond
	cfgPoll     = 2 * time.Second
	httpTimeout = 4 * time.Second
)

// Runner owns the guard and feeds it time, IP checks and config edits.
type Runner struct {
	G             *guard.Guard
	results       chan guard.ProbeResult
	geo           *netwatch.Geo
	ctx           context.Context
	modMu         sync.Mutex // save runs on the menu's goroutine too
	cfgMod        time.Time
	OnConfigError func(error)
}

func NewRunner(ctx context.Context, cfg *config.Config) *Runner {
	r := &Runner{results: make(chan guard.ProbeResult, 4), geo: netwatch.NewGeo(), ctx: ctx}
	if st, err := os.Stat(ConfigPath()); err == nil {
		r.cfgMod = st.ModTime()
	}
	r.G = guard.New(cfg, guard.Deps{
		Fingerprint: netwatch.Fingerprint,
		StartProbe:  r.probe,
		Procs:       procs.New(),
		SaveConfig:  r.save,
		Self:        os.Getpid(),
	})
	return r
}

func (r *Runner) probe(services []string) {
	go func() {
		ip, err := netwatch.PublicIP(r.ctx, services, httpTimeout)
		cc := ""
		if err == nil {
			cc = r.geo.Country(r.ctx, ip, httpTimeout)
		}
		select {
		case r.results <- guard.ProbeResult{IP: ip, Country: cc, Err: err}:
		case <-r.ctx.Done():
		}
	}()
}

func (r *Runner) save(c *config.Config) error {
	r.modMu.Lock()
	defer r.modMu.Unlock()
	err := config.Save(ConfigPath(), c)
	if st, e := os.Stat(ConfigPath()); e == nil {
		r.cfgMod = st.ModTime() // our own write is not an "edit" to reload
	}
	return err
}

func (r *Runner) reloadIfEdited(now time.Time) {
	r.modMu.Lock()
	st, err := os.Stat(ConfigPath())
	if err != nil || st.ModTime().Equal(r.cfgMod) {
		r.modMu.Unlock()
		return
	}
	r.cfgMod = st.ModTime()
	r.modMu.Unlock()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return
	}
	c, err := config.Parse(data)
	if err != nil {
		Logf("config.yaml: %v", err)
		if r.OnConfigError != nil {
			r.OnConfigError(err)
		}
		return
	}
	Logf("config.yaml перечитан")
	r.G.SetConfig(now, c)
}

// Run blocks until ctx is cancelled, then un-freezes everything.
func (r *Runner) Run() {
	t := time.NewTicker(tick)
	defer t.Stop()
	cp := time.NewTicker(cfgPoll)
	defer cp.Stop()
	defer r.G.Shutdown()
	r.G.Step(time.Now())
	for {
		select {
		case <-r.ctx.Done():
			return
		case now := <-t.C:
			r.G.Step(now)
		case res := <-r.results:
			r.G.HandleProbe(time.Now(), res)
		case now := <-cp.C:
			r.reloadIfEdited(now)
		}
	}
}

// State is what `kordon status` reads from state.json.
type State struct {
	Updated   time.Time `json:"updated"`
	Status    string    `json:"status"`
	Line      string    `json:"line"`
	IP        string    `json:"ip"`
	Country   string    `json:"country"`
	Allowed   bool      `json:"allowed"`
	Apps      []string  `json:"running_apps"`
	Frozen    int       `json:"frozen"`
	PID       int       `json:"pid"`
	Paused    bool      `json:"paused"`
	ConfigLoc string    `json:"config"`
}

func WriteState(s State) {
	s.Updated = time.Now()
	s.PID = os.Getpid()
	s.ConfigLoc = ConfigPath()
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := StatePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, StatePath())
	}
}

func ReadState() (State, error) {
	var s State
	b, err := os.ReadFile(StatePath())
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(b, &s)
}
