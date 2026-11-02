// Package guard is Kordon's core: it decides when the protected apps may run.
//
//	Starting ──first check──▶ OK ◀───────── allowed IP ──────────┐
//	                           │                                 │
//	         network changed / IP not allowed / offline          │
//	                           ▼                                 │
//	                        Suspect (apps frozen) ── grace ──▶ Blocked (apps closed)
//
// Suspect freezes the apps instead of killing them: if the right IP comes back
// within the grace period they continue as if nothing happened.
//
// Guard does no I/O of its own beyond Deps and is driven by Step and HandleProbe,
// so tests can run it on a fake clock.
package guard

import (
	"slices"
	"sync"
	"time"

	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/procs"
)

type Status int

const (
	Starting Status = iota // the first IP check hasn't finished
	OK                     // apps may run
	Suspect                // apps frozen, waiting for an allowed IP
	Blocked                // apps closed until an allowed IP is back
)

type Reason int

const (
	NoReason Reason = iota
	NetChanged
	IPNotAllowed
	Offline
)

const (
	// FastCheck is the re-check interval while something is wrong.
	FastCheck = 2 * time.Second
	// probeStuck guards against a check that never reports back.
	probeStuck = 30 * time.Second
	historyLen = 30
)

// Forever pauses until Kordon restarts.
const Forever = time.Duration(1<<63 - 1)

type ProbeResult struct {
	IP, Country string
	Err         error
}

type Deps struct {
	Fingerprint func() string
	// StartProbe begins an IP check in the background; its result must come back through HandleProbe.
	StartProbe func(services []string)
	Procs      procs.Controller
	SaveConfig func(*config.Config) error
	Self       int
}

type Guard struct {
	mu   sync.Mutex
	d    Deps
	cfg  *config.Config
	tgts []procs.Target

	status       Status
	reason       Reason
	since        time.Time // when Suspect/Blocked began
	ip, country  string
	offline      bool
	lastFP       string
	nextCheck    time.Time
	probing      bool
	probeStarted time.Time
	pausedUntil  time.Time
	recovered    bool

	frozen     map[int]procs.Proc
	closing    map[int]time.Time
	running    []procs.Proc
	notifiedKO bool // "can't start" already said during this block

	history  []Event
	listener func(Snapshot, []Event)
	pending  []Event
	lastKey  string
}

func New(cfg *config.Config, d Deps) *Guard {
	g := &Guard{d: d, frozen: map[int]procs.Proc{}, closing: map[int]time.Time{}}
	g.setConfig(cfg)
	return g
}

// OnChange is called after every change with the new state and the events since the last call.
func (g *Guard) OnChange(f func(Snapshot, []Event)) {
	g.mu.Lock()
	g.listener = f
	g.mu.Unlock()
}

func (g *Guard) setConfig(c *config.Config) {
	g.cfg = c.Clone()
	g.tgts = procs.Targets(g.cfg)
}

func (g *Guard) paused(now time.Time) bool { return now.Before(g.pausedUntil) }

func (g *Guard) emit(now time.Time, e Event) {
	e.At = now
	g.history = append(g.history, e)
	if len(g.history) > historyLen {
		g.history = g.history[len(g.history)-historyLen:]
	}
	g.pending = append(g.pending, e)
}

// publish must be called with g.mu held; it runs the listener without the lock.
func (g *Guard) publish(now time.Time) {
	s := g.snapshot(now)
	key := s.key()
	if key == g.lastKey && len(g.pending) == 0 {
		return
	}
	g.lastKey = key
	evs := g.pending
	g.pending = nil
	if f := g.listener; f != nil {
		g.mu.Unlock()
		f(s, evs)
		g.mu.Lock()
	}
}

// Step runs every ~500 ms: notices network changes, schedules checks, enforces the state.
func (g *Guard) Step(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if fp := g.d.Fingerprint(); fp != g.lastFP {
		if g.lastFP != "" {
			g.emit(now, Event{Kind: EvNetChanged})
			if g.cfg.FreezeOnNetworkChange && g.status == OK && !g.paused(now) {
				g.toSuspect(now, NetChanged)
			}
		}
		g.lastFP = fp
		g.nextCheck = now // check the IP right away
	}

	if !g.pausedUntil.IsZero() && !g.paused(now) {
		g.pausedUntil = time.Time{}
		g.emit(now, Event{Kind: EvUnpaused})
		g.nextCheck = now
	}

	if g.probing && now.Sub(g.probeStarted) > probeStuck {
		g.probing = false
	}
	if !g.probing && !now.Before(g.nextCheck) {
		g.probing = true
		g.probeStarted = now
		g.d.StartProbe(slices.Clone(g.cfg.IPServices))
	}

	if g.status == Suspect && now.Sub(g.since) >= g.cfg.Grace.D() {
		g.toBlocked(now)
	}
	g.enforce(now)
	g.publish(now)
}

// HandleProbe takes the result of an IP check started by StartProbe.
func (g *Guard) HandleProbe(now time.Time, r ProbeResult) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	g.probing = false

	if r.Err != nil {
		if !g.offline {
			g.emit(now, Event{Kind: EvOffline})
		}
		g.offline, g.ip, g.country = true, "", ""
		g.nextCheck = now.Add(FastCheck)
		if g.paused(now) {
			return
		}
		switch {
		case g.cfg.BlockWhenOffline && (g.status == OK || g.status == Starting):
			g.toSuspect(now, Offline)
		case g.status == Suspect:
			g.reason = Offline
		case !g.cfg.BlockWhenOffline && g.status == Starting:
			g.status = OK
		}
		return
	}

	if g.offline {
		g.emit(now, Event{Kind: EvOnline, IP: r.IP, Country: r.Country})
	} else if g.ip != "" && r.IP != g.ip {
		g.emit(now, Event{Kind: EvIPChanged, IP: r.IP, Country: r.Country})
	}
	g.offline, g.ip, g.country = false, r.IP, r.Country

	if g.cfg.Allow.IsEmpty() {
		// First run: whatever network the user installed Kordon on is the right one.
		g.cfg.Allow.AddIP(r.IP)
		g.save()
		g.emit(now, Event{Kind: EvAutoAllowed, IP: r.IP, Country: r.Country, Notify: true})
	}
	g.reevaluate(now)
}

// reevaluate applies the allow-list to the current IP.
func (g *Guard) reevaluate(now time.Time) {
	if g.offline || g.ip == "" {
		return
	}
	if g.paused(now) {
		g.status, g.reason = OK, NoReason
		g.nextCheck = now.Add(g.cfg.CheckInterval.D())
		return
	}
	if g.cfg.Allow.Matches(g.ip, g.country) {
		g.toOK(now)
		g.nextCheck = now.Add(g.cfg.CheckInterval.D())
		return
	}
	switch g.status {
	case OK, Starting:
		g.toSuspect(now, IPNotAllowed)
	case Suspect:
		if g.reason != IPNotAllowed {
			g.reason = IPNotAllowed
			g.announceFrozen(now)
		}
	}
	g.nextCheck = now.Add(FastCheck)
}

func (g *Guard) toOK(now time.Time) {
	switch g.status {
	case Suspect:
		if len(g.frozen) > 0 && g.reason != NetChanged {
			g.emit(now, Event{Kind: EvResumed, IP: g.ip, Country: g.country, Apps: procs.AppNames(mapValues(g.frozen)), Notify: true})
		}
	case Blocked:
		g.emit(now, Event{Kind: EvUnblocked, IP: g.ip, Country: g.country, Notify: true})
	}
	g.status, g.reason = OK, NoReason
	g.resumeAll()
	g.closing = map[int]time.Time{}
}

func (g *Guard) toSuspect(now time.Time, r Reason) {
	g.status, g.reason, g.since = Suspect, r, now
	g.nextCheck = now
	g.enforce(now) // freeze right now, not on the next tick
	g.announceFrozen(now)
}

// announceFrozen tells about the freeze unless it's a routine network-change check.
func (g *Guard) announceFrozen(now time.Time) {
	if g.reason == NetChanged {
		return
	}
	g.emit(now, Event{Kind: EvFrozen, Reason: g.reason, IP: g.ip, Country: g.country,
		Apps: procs.AppNames(mapValues(g.frozen)), Dur: g.cfg.Grace.D(), Notify: len(g.frozen) > 0})
}

func (g *Guard) toBlocked(now time.Time) {
	g.status, g.since, g.notifiedKO = Blocked, now, false
	apps := procs.AppNames(mapValues(g.frozen))
	for pid, p := range g.frozen {
		_ = g.d.Procs.Close(p)
		g.closing[pid] = now
	}
	g.frozen = map[int]procs.Proc{}
	g.emit(now, Event{Kind: EvBlocked, Reason: g.reason, IP: g.ip, Country: g.country, Apps: apps, Notify: len(apps) > 0})
}

// releaseUntargeted resumes frozen processes of apps that are no longer protected.
func (g *Guard) releaseUntargeted() {
	for pid, p := range g.frozen {
		if len(procs.Filter([]procs.Proc{p}, g.tgts, g.d.Self)) == 0 {
			_ = g.d.Procs.Resume(p)
			delete(g.frozen, pid)
		}
	}
}

func (g *Guard) resumeAll() {
	for _, p := range g.frozen {
		_ = g.d.Procs.Resume(p)
	}
	g.frozen = map[int]procs.Proc{}
}

func (g *Guard) enforce(now time.Time) {
	all, err := g.d.Procs.List()
	if err != nil {
		return
	}
	list := procs.Filter(all, g.tgts, g.d.Self)
	g.running = list

	if !g.recovered {
		// If Kordon crashed while apps were frozen, they would stay frozen forever.
		g.recovered = true
		for _, p := range list {
			_ = g.d.Procs.Resume(p)
		}
	}

	if g.paused(now) || g.status == OK {
		g.resumeAll()
		return
	}

	alive := map[int]bool{}
	var started []procs.Proc
	for _, p := range list {
		alive[p.PID] = true
		switch g.status {
		case Starting, Suspect:
			if _, ok := g.frozen[p.PID]; !ok {
				if g.d.Procs.Freeze(p) == nil {
					g.frozen[p.PID] = p
				}
			}
		case Blocked:
			if t, ok := g.closing[p.PID]; ok {
				if now.Sub(t) >= g.cfg.CloseTimeout.D() {
					_ = g.d.Procs.Kill(p)
				}
				continue
			}
			// Started while blocked: nothing to save yet, so no polite close.
			_ = g.d.Procs.Kill(p)
			g.closing[p.PID] = now
			started = append(started, p)
		}
	}
	for pid := range g.frozen {
		if !alive[pid] {
			delete(g.frozen, pid)
		}
	}
	for pid := range g.closing {
		if !alive[pid] {
			delete(g.closing, pid)
		}
	}
	if len(started) > 0 {
		g.emit(now, Event{Kind: EvKilledNew, IP: g.ip, Country: g.country, Apps: procs.AppNames(started), Notify: !g.notifiedKO})
		g.notifiedKO = true
	}
}

func (g *Guard) save() {
	if g.d.SaveConfig != nil {
		_ = g.d.SaveConfig(g.cfg.Clone())
	}
}

// ---- actions from the menu / CLI ----

// SetConfig applies an edited config.yaml.
func (g *Guard) SetConfig(now time.Time, c *config.Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	g.setConfig(c)
	g.releaseUntargeted()
	g.reevaluate(now)
	g.enforce(now)
}

func (g *Guard) AllowCurrentIP(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if g.ip == "" || !g.cfg.Allow.AddIP(g.ip) {
		return false
	}
	g.save()
	g.emit(now, Event{Kind: EvAllowedIP, IP: g.ip, Country: g.country})
	g.reevaluate(now)
	return true
}

func (g *Guard) AllowCurrentCountry(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if g.country == "" || !g.cfg.Allow.AddCountry(g.country) {
		return false
	}
	g.save()
	g.emit(now, Event{Kind: EvAllowedCountry, IP: g.ip, Country: g.country})
	g.reevaluate(now)
	return true
}

// ToggleApp turns a built-in preset on or off.
func (g *Guard) ToggleApp(now time.Time, id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	c := g.cfg.Clone()
	if i := slices.Index(c.Apps, id); i >= 0 {
		c.Apps = slices.Delete(c.Apps, i, i+1)
	} else {
		c.Apps = append(c.Apps, id)
	}
	g.setConfig(c)
	g.releaseUntargeted()
	g.save()
	g.enforce(now)
}

// Update changes the config from the menu and saves it.
func (g *Guard) Update(now time.Time, fn func(*config.Config)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	c := g.cfg.Clone()
	fn(c)
	g.setConfig(c)
	g.releaseUntargeted()
	g.save()
	g.reevaluate(now)
	g.enforce(now)
}

// Pause stops protecting for d (Forever = until restart).
func (g *Guard) Pause(now time.Time, d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if d == Forever {
		g.pausedUntil = now.AddDate(100, 0, 0)
	} else {
		g.pausedUntil = now.Add(d)
	}
	g.status, g.reason = OK, NoReason
	g.resumeAll()
	g.closing = map[int]time.Time{}
	g.emit(now, Event{Kind: EvPaused, Dur: d})
}

func (g *Guard) Unpause(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if g.pausedUntil.IsZero() {
		return
	}
	g.pausedUntil = time.Time{}
	g.emit(now, Event{Kind: EvUnpaused})
	g.nextCheck = now
	g.reevaluate(now)
}

// CheckNow re-checks the IP on the next step.
func (g *Guard) CheckNow(now time.Time) {
	g.mu.Lock()
	g.nextCheck = now
	g.mu.Unlock()
}

// Shutdown un-freezes everything; called on exit.
func (g *Guard) Shutdown() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resumeAll()
}

func (g *Guard) Snapshot(now time.Time) Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.snapshot(now)
}

func mapValues(m map[int]procs.Proc) []procs.Proc {
	out := make([]procs.Proc, 0, len(m))
	for _, p := range m {
		out = append(out, p)
	}
	return out
}
