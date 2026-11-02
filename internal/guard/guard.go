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
// Apps with their own rules (app_rules) form separate groups, each with its own
// state: Claude can be frozen while Cursor, allowed in more countries, runs on.
//
// Guard does no I/O of its own beyond Deps and is driven by Step and HandleProbe,
// so tests can run it on a fake clock.
package guard

import (
	"slices"
	"strings"
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

// worse orders statuses for the overall state shown in the menu.
func worse(a, b Status) bool {
	rank := map[Status]int{OK: 0, Starting: 1, Suspect: 2, Blocked: 3}
	return rank[a] > rank[b]
}

type Reason int

const (
	NoReason     Reason = iota
	NetChanged          // routine re-check after a network change
	IPNotAllowed        // the public IPv4 is not on the allow-list
	Offline             // no internet
	IPv6Leak            // IPv4 is fine, but there is a public IPv6 that isn't allowed
	Unconfirmed         // services disagree about the IP, or too few answered
	DNSLeak             // DNS lookups go out somewhere else than the VPN (dns_check: block)
)

const (
	// FastCheck is the re-check interval while something is wrong.
	FastCheck = 2 * time.Second
	// probeStuck guards against a check that never reports back.
	probeStuck = 30 * time.Second
	historyLen = 50
)

// Forever pauses until Kordon restarts.
const Forever = time.Duration(1<<63 - 1)

type ProbeResult struct {
	IP, Country     string
	Votes           int      // how many IP services agreed on IP
	Others          []string // other addresses some services reported
	IP6, Country6   string   // public IPv6, empty if there is none
	DNS, DNSCountry string   // public address of the DNS resolver in use
	NetID           string   // identifies the local network (the router's MAC)
	Err             error
}

type Deps struct {
	Fingerprint func() string
	// StartProbe begins an IP check in the background; its result must come back through HandleProbe.
	StartProbe func(c *config.Config)
	Procs      procs.Controller
	SaveConfig func(*config.Config) error
	Self       int
}

// group is a set of apps sharing one allow-list, with its own state.
type group struct {
	key        string // "" for the shared rules, else the app_rules key
	tgts       []procs.Target
	status     Status
	reason     Reason
	since      time.Time
	frozen     map[int]procs.Proc
	closing    map[int]time.Time
	running    []procs.Proc
	notifiedKO bool // "can't start" already said during this block
}

func newGroup(key string, st Status) *group {
	return &group{key: key, status: st, frozen: map[int]procs.Proc{}, closing: map[int]time.Time{}}
}

func (gr *group) names() []string {
	var n []string
	for _, t := range gr.tgts {
		n = append(n, t.Name)
	}
	return n
}

type Guard struct {
	mu     sync.Mutex
	d      Deps
	cfg    *config.Config
	groups []*group

	ip, country  string
	ip6, cc6     string
	votes        int
	others       []string
	dns, dnsCC   string
	netID        string
	offline      bool
	known        bool // at least one check finished
	lastFP       string
	nextCheck    time.Time
	probing      bool
	probeStarted time.Time
	pausedUntil  time.Time
	recovered    bool
	lastStep     time.Time
	dnsWarned    string

	counters Counters
	history  []Event
	listener func(Snapshot, []Event)
	pending  []Event
	lastKey  string
}

// Counters are totals since Kordon started, for the statistics.
type Counters struct {
	Freezes, Saves, Blocks, Killed int
	FrozenSeconds                  float64
}

func New(cfg *config.Config, d Deps) *Guard {
	g := &Guard{d: d}
	g.setConfig(cfg)
	return g
}

// OnChange is called after every change with the new state and the events since the last call.
func (g *Guard) OnChange(f func(Snapshot, []Event)) {
	g.mu.Lock()
	g.listener = f
	g.mu.Unlock()
}

// setConfig applies a config and regroups the apps, keeping each group's state.
// It returns apps it un-froze because their rule went away and their new group is fine.
func (g *Guard) setConfig(c *config.Config) []procs.Proc {
	g.cfg = c.Clone()
	all := procs.Targets(g.cfg)
	byKey := map[string][]procs.Target{}
	order := []string{""}
	for _, t := range all {
		key := ""
		if _, ok := g.cfg.AppRules[t.ID]; ok {
			key = t.ID
		}
		if _, seen := byKey[key]; !seen && key != "" {
			order = append(order, key)
		}
		byKey[key] = append(byKey[key], t)
	}
	old := map[string]*group{}
	for _, gr := range g.groups {
		old[gr.key] = gr
	}
	// A new group starts where the shared group is: reevaluateAll fixes it right
	// after, except while offline, and then the shared group's state is the right one.
	base := old[""]
	var groups []*group
	for _, key := range order {
		gr, ok := old[key]
		if !ok {
			gr = newGroup(key, Starting)
			if base != nil {
				gr.status, gr.reason, gr.since = base.status, base.reason, base.since
			}
		}
		delete(old, key)
		gr.tgts = byKey[key]
		groups = append(groups, gr)
	}
	g.groups = groups
	// Apps of a group that's gone move to the group that owns them now, so the
	// next evaluation resumes them with the usual notice; anything left is resumed.
	var resumed []procs.Proc
	for _, gone := range old {
		for pid, p := range gone.frozen {
			if to := g.ownerOf(p); to != nil && to.status != OK {
				to.frozen[pid] = p
			} else {
				_ = g.d.Procs.Resume(p)
				if gone.reason != NetChanged {
					resumed = append(resumed, p)
				}
			}
		}
	}
	return resumed
}

func (g *Guard) paused(now time.Time) bool { return now.Before(g.pausedUntil) }

func (g *Guard) allowFor(gr *group) config.Allow {
	if gr.key == "" {
		return g.cfg.Allow
	}
	return g.cfg.AppRules[gr.key]
}

func (g *Guard) trusted() bool {
	_, ok := g.cfg.Trusted(g.netID)
	return ok
}

// dnsLeak: the resolver is in another country than the VPN exit, and that country isn't allowed either.
func (g *Guard) dnsLeak(a config.Allow) bool {
	return g.dnsCC != "" && g.country != "" && g.dnsCC != g.country && !slices.Contains(a.Countries, g.dnsCC)
}

func (g *Guard) emit(now time.Time, e Event) {
	e.At = now
	switch e.Kind {
	case EvFrozen:
		g.counters.Freezes++
	case EvResumed:
		g.counters.Saves++
	case EvBlocked:
		g.counters.Blocks++
	case EvKilledNew:
		g.counters.Killed++
	}
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

// Step runs every ~500 ms and on OS network events: notices network changes,
// schedules checks, enforces the state.
func (g *Guard) Step(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.lastStep.IsZero() {
		if dt := now.Sub(g.lastStep).Seconds(); dt > 0 && dt < 5 && g.anyAlarm() {
			g.counters.FrozenSeconds += dt
		}
	}
	g.lastStep = now

	if fp := g.d.Fingerprint(); fp != g.lastFP {
		if g.lastFP != "" {
			g.emit(now, Event{Kind: EvNetChanged})
			if g.cfg.FreezeOnNetworkChange && !g.paused(now) {
				for _, gr := range g.groups {
					if gr.status == OK {
						g.toSuspect(now, gr, NetChanged)
					}
				}
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
		g.d.StartProbe(g.cfg.Clone())
	}

	for _, gr := range g.groups {
		if gr.status == Suspect && now.Sub(gr.since) >= g.cfg.Grace.D() {
			g.toBlocked(now, gr)
		}
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
	g.known = true

	if r.NetID != g.netID {
		g.netID = r.NetID
		if n, ok := g.cfg.Trusted(r.NetID); ok {
			g.emit(now, Event{Kind: EvTrusted, Network: n.Name})
		}
	}

	if r.Err != nil {
		if !g.offline {
			g.emit(now, Event{Kind: EvOffline})
		}
		g.offline, g.ip, g.country, g.ip6, g.cc6, g.votes, g.others = true, "", "", "", "", 0, nil
		g.dns, g.dnsCC = "", ""
		g.nextCheck = now.Add(FastCheck)
		if g.paused(now) {
			return
		}
		for _, gr := range g.groups {
			switch {
			case g.cfg.BlockWhenOffline && (gr.status == OK || gr.status == Starting):
				g.toSuspect(now, gr, Offline)
			case gr.status == Suspect:
				gr.reason = Offline
			case !g.cfg.BlockWhenOffline && gr.status == Starting:
				gr.status = OK
			}
		}
		return
	}

	if g.offline {
		g.emit(now, Event{Kind: EvOnline, IP: r.IP, Country: r.Country})
	} else if g.ip != "" && r.IP != g.ip {
		g.emit(now, Event{Kind: EvIPChanged, IP: r.IP, Country: r.Country})
	}
	g.offline, g.ip, g.country = false, r.IP, r.Country
	g.ip6, g.cc6, g.votes, g.others = r.IP6, r.Country6, r.Votes, r.Others
	g.dns, g.dnsCC = r.DNS, r.DNSCountry

	if g.cfg.Allow.IsEmpty() {
		// First run: whatever network the user installed Kordon on is the right one.
		g.cfg.Allow.AddIP(r.IP)
		if r.IP6 != "" {
			g.cfg.Allow.AddIP(r.IP6)
		}
		g.save()
		g.emit(now, Event{Kind: EvAutoAllowed, IP: r.IP, Country: r.Country, Notify: true})
	}

	if g.cfg.DNSCheck == "warn" && !g.trusted() && g.dnsLeak(g.cfg.Allow) {
		if k := g.netID + "|" + g.dnsCC; k != g.dnsWarned {
			g.dnsWarned = k
			g.emit(now, Event{Kind: EvDNSLeak, IP: g.dns, Country: g.dnsCC, Notify: true})
		}
	}
	g.reevaluateAll(now)
}

// verdict says why the network is not trusted for a group, or NoReason if it is.
func (g *Guard) verdict(gr *group) Reason {
	if g.trusted() {
		return NoReason
	}
	a := g.allowFor(gr)
	if !a.Matches(g.ip, g.country) {
		return IPNotAllowed
	}
	if g.votes < g.cfg.Need() || len(g.others) > 0 {
		return Unconfirmed
	}
	if g.cfg.CheckIPv6 && g.ip6 != "" && !a.Matches(g.ip6, g.cc6) {
		return IPv6Leak
	}
	if g.cfg.DNSCheck == "block" && g.dnsLeak(a) {
		return DNSLeak
	}
	return NoReason
}

func (g *Guard) reevaluateAll(now time.Time) {
	if g.offline || g.ip == "" {
		return
	}
	allOK := true
	for _, gr := range g.groups {
		g.reevaluate(now, gr)
		if gr.status != OK {
			allOK = false
		}
	}
	if allOK {
		g.nextCheck = now.Add(g.cfg.CheckInterval.D())
	} else {
		g.nextCheck = now.Add(FastCheck)
	}
}

func (g *Guard) reevaluate(now time.Time, gr *group) {
	if g.paused(now) {
		gr.status, gr.reason = OK, NoReason
		return
	}
	why := g.verdict(gr)
	if why == NoReason {
		g.toOK(now, gr)
		return
	}
	switch gr.status {
	case OK, Starting:
		g.toSuspect(now, gr, why)
	case Suspect:
		if gr.reason != why {
			gr.reason = why
			g.announceFrozen(now, gr)
		}
	case Blocked:
		gr.reason = why
	}
}

func (g *Guard) event(gr *group, kind EventKind, apps []string, notify bool) Event {
	return Event{Kind: kind, Reason: gr.reason, IP: g.ip, Country: g.country, IP6: g.ip6, Country6: g.cc6,
		Others: g.others, DNS: g.dns, DNSCountry: g.dnsCC, Apps: apps, Dur: g.cfg.Grace.D(), Notify: notify}
}

func (g *Guard) toOK(now time.Time, gr *group) {
	switch gr.status {
	case Suspect:
		if len(gr.frozen) > 0 && gr.reason != NetChanged {
			g.emit(now, g.event(gr, EvResumed, procs.AppNames(mapValues(gr.frozen)), true))
		}
	case Blocked:
		g.emit(now, g.event(gr, EvUnblocked, gr.names(), true))
	}
	gr.status, gr.reason = OK, NoReason
	g.resume(gr)
	gr.closing = map[int]time.Time{}
}

func (g *Guard) toSuspect(now time.Time, gr *group, r Reason) {
	gr.status, gr.reason, gr.since = Suspect, r, now
	g.nextCheck = now
	g.enforce(now) // freeze right now, not on the next tick
	g.announceFrozen(now, gr)
}

// announceFrozen tells about the freeze unless it's a routine network-change check.
func (g *Guard) announceFrozen(now time.Time, gr *group) {
	if gr.reason == NetChanged {
		return
	}
	g.emit(now, g.event(gr, EvFrozen, procs.AppNames(mapValues(gr.frozen)), len(gr.frozen) > 0))
}

func (g *Guard) toBlocked(now time.Time, gr *group) {
	gr.status, gr.since, gr.notifiedKO = Blocked, now, false
	apps := procs.AppNames(mapValues(gr.frozen))
	for pid, p := range gr.frozen {
		_ = g.d.Procs.Close(p)
		gr.closing[pid] = now
	}
	gr.frozen = map[int]procs.Proc{}
	g.emit(now, g.event(gr, EvBlocked, apps, len(apps) > 0))
}

func (g *Guard) resume(gr *group) {
	for _, p := range gr.frozen {
		_ = g.d.Procs.Resume(p)
	}
	gr.frozen = map[int]procs.Proc{}
}

func (g *Guard) resumeAll() {
	for _, gr := range g.groups {
		g.resume(gr)
	}
}

// anyAlarm: apps are frozen because something is wrong (not the short freeze at start).
func (g *Guard) anyAlarm() bool {
	for _, gr := range g.groups {
		if gr.status == Suspect && gr.reason != NetChanged && len(gr.frozen) > 0 {
			return true
		}
	}
	return false
}

func (g *Guard) ownerOf(p procs.Proc) *group {
	for _, gr := range g.groups {
		if len(procs.Filter([]procs.Proc{p}, gr.tgts, g.d.Self)) > 0 {
			return gr
		}
	}
	return nil
}

// releaseUntargeted resumes frozen processes that no group claims any more.
func (g *Guard) releaseUntargeted() {
	for _, gr := range g.groups {
		for pid, p := range gr.frozen {
			if len(procs.Filter([]procs.Proc{p}, gr.tgts, g.d.Self)) == 0 {
				_ = g.d.Procs.Resume(p)
				delete(gr.frozen, pid)
			}
		}
	}
}

func (g *Guard) enforce(now time.Time) {
	all, err := g.d.Procs.List()
	if err != nil {
		return
	}
	var tgts []procs.Target
	groupOf := map[string]*group{}
	for _, gr := range g.groups {
		tgts = append(tgts, gr.tgts...)
		for _, t := range gr.tgts {
			groupOf[t.ID] = gr
		}
	}
	// One pass over all targets, so a process (or a child) belongs to exactly one group.
	list := procs.Filter(all, tgts, g.d.Self)
	for _, gr := range g.groups {
		gr.running = nil
	}
	for _, p := range list {
		if gr := groupOf[p.AppID]; gr != nil {
			gr.running = append(gr.running, p)
		}
	}

	if !g.recovered {
		// If Kordon crashed while apps were frozen, they would stay frozen forever.
		g.recovered = true
		for _, p := range list {
			_ = g.d.Procs.Resume(p)
		}
	}

	for _, gr := range g.groups {
		g.enforceGroup(now, gr)
	}
}

func (g *Guard) enforceGroup(now time.Time, gr *group) {
	if g.paused(now) || gr.status == OK {
		g.resume(gr)
		return
	}
	alive := map[int]bool{}
	var started []procs.Proc
	for _, p := range gr.running {
		alive[p.PID] = true
		switch gr.status {
		case Starting, Suspect:
			if _, ok := gr.frozen[p.PID]; !ok {
				if g.d.Procs.Freeze(p) == nil {
					gr.frozen[p.PID] = p
				}
			}
		case Blocked:
			if t, ok := gr.closing[p.PID]; ok {
				if now.Sub(t) >= g.cfg.CloseTimeout.D() {
					_ = g.d.Procs.Kill(p)
				}
				continue
			}
			// Started while blocked: nothing to save yet, so no polite close.
			_ = g.d.Procs.Kill(p)
			gr.closing[p.PID] = now
			started = append(started, p)
		}
	}
	for pid := range gr.frozen {
		if !alive[pid] {
			delete(gr.frozen, pid)
		}
	}
	for pid := range gr.closing {
		if !alive[pid] {
			delete(gr.closing, pid)
		}
	}
	if len(started) > 0 {
		g.emit(now, g.event(gr, EvKilledNew, procs.AppNames(started), !gr.notifiedKO))
		gr.notifiedKO = true
	}
}

func (g *Guard) save() {
	if g.d.SaveConfig != nil {
		_ = g.d.SaveConfig(g.cfg.Clone())
	}
}

// ---- actions from the menu, the panel and the CLI ----

// apply swaps in an edited config and re-checks everything against it.
func (g *Guard) apply(now time.Time, c *config.Config, save bool) {
	if resumed := g.setConfig(c); len(resumed) > 0 {
		g.emit(now, Event{Kind: EvResumed, IP: g.ip, Country: g.country, Apps: procs.AppNames(resumed), Notify: true})
	}
	g.releaseUntargeted()
	if save {
		g.save()
	}
	g.reevaluateAll(now)
	g.enforce(now)
}

// SetConfig applies an edited config.yaml.
func (g *Guard) SetConfig(now time.Time, c *config.Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	g.apply(now, c, false)
}

// Update changes the config from the menu or the panel and saves it.
func (g *Guard) Update(now time.Time, fn func(*config.Config)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	c := g.cfg.Clone()
	fn(c)
	g.apply(now, c, true)
}

func (g *Guard) AllowCurrentIP(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if g.ip == "" {
		return false
	}
	c := g.cfg.Clone()
	added := c.Allow.AddIP(g.ip)
	if g.ip6 != "" && c.Allow.AddIP(g.ip6) {
		added = true
	}
	if !added {
		return false
	}
	g.emit(now, Event{Kind: EvAllowedIP, IP: g.ip, Country: g.country})
	g.apply(now, c, true)
	return true
}

func (g *Guard) AllowCurrentCountry(now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if g.country == "" {
		return false
	}
	c := g.cfg.Clone()
	added := c.Allow.AddCountry(g.country)
	if g.ip6 != "" && c.Allow.AddCountry(g.cc6) {
		added = true
	}
	if !added {
		return false
	}
	g.emit(now, Event{Kind: EvAllowedCountry, IP: g.ip, Country: g.country})
	g.apply(now, c, true)
	return true
}

// TrustCurrentNetwork marks the local network Kordon is on as trusted.
func (g *Guard) TrustCurrentNetwork(now time.Time, name string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	defer g.publish(now)
	if g.netID == "" || g.trusted() {
		return false
	}
	if name = strings.TrimSpace(name); name == "" {
		name = g.netID
	}
	c := g.cfg.Clone()
	c.TrustedNetworks = append(c.TrustedNetworks, config.TrustedNetwork{Name: name, ID: g.netID})
	g.emit(now, Event{Kind: EvTrusted, Network: name})
	g.apply(now, c, true)
	return true
}

// ToggleApp turns a built-in preset on or off.
func (g *Guard) ToggleApp(now time.Time, id string) {
	g.Update(now, func(c *config.Config) {
		if i := slices.Index(c.Apps, id); i >= 0 {
			c.Apps = slices.Delete(c.Apps, i, i+1)
		} else {
			c.Apps = append(c.Apps, id)
		}
	})
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
	for _, gr := range g.groups {
		gr.status, gr.reason = OK, NoReason
		g.resume(gr)
		gr.closing = map[int]time.Time{}
	}
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
	g.reevaluateAll(now)
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
