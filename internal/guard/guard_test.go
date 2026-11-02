package guard

import (
	"errors"
	"testing"
	"time"

	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/procs"
)

// fakeProcs records what the guard did to each process.
type fakeProcs struct {
	procs  []procs.Proc
	frozen map[int]bool
	closed map[int]bool
	killed map[int]bool
}

func newFakeProcs(ps ...procs.Proc) *fakeProcs {
	return &fakeProcs{procs: ps, frozen: map[int]bool{}, closed: map[int]bool{}, killed: map[int]bool{}}
}

func (f *fakeProcs) List() ([]procs.Proc, error) { return f.procs, nil }
func (f *fakeProcs) Freeze(p procs.Proc) error   { f.frozen[p.PID] = true; return nil }
func (f *fakeProcs) Resume(p procs.Proc) error   { f.frozen[p.PID] = false; return nil }
func (f *fakeProcs) Close(p procs.Proc) error {
	f.closed[p.PID] = true
	f.frozen[p.PID] = false
	return nil
}
func (f *fakeProcs) Kill(p procs.Proc) error { f.killed[p.PID] = true; return nil }
func (f *fakeProcs) remove(pid int) {
	for i, p := range f.procs {
		if p.PID == pid {
			f.procs = append(f.procs[:i], f.procs[i+1:]...)
			return
		}
	}
}

var claude = procs.Proc{PID: 100, Name: "Claude", Path: "/Applications/Claude.app/Contents/MacOS/Claude"}
var other = procs.Proc{PID: 200, Name: "Safari", Path: "/Applications/Safari.app/Contents/MacOS/Safari"}

type harness struct {
	t      *testing.T
	g      *Guard
	p      *fakeProcs
	now    time.Time
	fp     string
	probes int
	saved  *config.Config
	events []Event
}

func newHarness(t *testing.T, allow ...string) *harness {
	h := &harness{t: t, p: newFakeProcs(claude, other), now: time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC), fp: "en0=192.168.0.10,utun6=172.19.0.1"}
	cfg := config.Default()
	cfg.Allow.IPs = allow
	h.g = New(cfg, Deps{
		Fingerprint: func() string { return h.fp },
		StartProbe:  func(*config.Config) { h.probes++ },
		Procs:       h.p,
		SaveConfig:  func(c *config.Config) error { h.saved = c; return nil },
	})
	h.g.OnChange(func(_ Snapshot, evs []Event) { h.events = append(h.events, evs...) })
	return h
}

func (h *harness) step(d time.Duration) { h.now = h.now.Add(d); h.g.Step(h.now) }
func (h *harness) probe(ip, cc string) {
	h.g.HandleProbe(h.now, ProbeResult{IP: ip, Country: cc, Votes: 2})
}
func (h *harness) probeFull(r ProbeResult) { h.g.HandleProbe(h.now, r) }
func (h *harness) offline()                { h.g.HandleProbe(h.now, ProbeResult{Err: errors.New("offline")}) }
func (h *harness) status() Status          { return h.g.Snapshot(h.now).Status }

func (h *harness) want(s Status) {
	h.t.Helper()
	if got := h.status(); got != s {
		h.t.Fatalf("status = %v, want %v", got, s)
	}
}

func (h *harness) had(k EventKind) bool {
	for _, e := range h.events {
		if e.Kind == k {
			return true
		}
	}
	return false
}

func TestFirstRunAllowsCurrentIP(t *testing.T) {
	h := newHarness(t)
	h.step(0)
	if !h.p.frozen[claude.PID] {
		t.Fatal("apps must stay frozen until the first check")
	}
	if h.p.frozen[other.PID] {
		t.Fatal("unrelated app frozen")
	}
	h.probe("64.137.54.87", "NL")
	h.want(OK)
	if h.p.frozen[claude.PID] {
		t.Fatal("claude still frozen after OK")
	}
	if h.saved == nil || len(h.saved.Allow.IPs) != 1 || h.saved.Allow.IPs[0] != "64.137.54.87" {
		t.Fatalf("first IP not saved to config: %+v", h.saved)
	}
	if !h.had(EvAutoAllowed) {
		t.Fatal("no EvAutoAllowed")
	}
}

func TestVPNDropAndReturnWithinGrace(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("64.137.54.87", "NL")
	h.want(OK)

	// VPN off: the tunnel interface disappears.
	h.fp = "en0=192.168.0.10"
	h.step(500 * time.Millisecond)
	h.want(Suspect)
	if !h.p.frozen[claude.PID] {
		t.Fatal("claude not frozen on network change")
	}
	h.probe("95.24.1.1", "RU")
	h.want(Suspect)
	if !h.had(EvFrozen) {
		t.Fatal("no EvFrozen for a disallowed IP")
	}

	// VPN back 10 s later: same IP, apps continue.
	h.fp = "en0=192.168.0.10,utun6=172.19.0.1"
	h.step(10 * time.Second)
	h.probe("64.137.54.87", "NL")
	h.want(OK)
	if h.p.frozen[claude.PID] || h.p.closed[claude.PID] {
		t.Fatal("claude must be resumed, not closed")
	}
	if !h.had(EvResumed) {
		t.Fatal("no EvResumed")
	}
}

func TestNetworkBlipWithSameIPIsSilent(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("64.137.54.87", "NL")
	h.events = nil
	h.fp = "en0=192.168.0.11,utun6=172.19.0.1"
	h.step(500 * time.Millisecond)
	h.probe("64.137.54.87", "NL")
	h.want(OK)
	for _, e := range h.events {
		if e.Notify {
			t.Fatalf("routine re-check must not notify, got %+v", e)
		}
	}
}

func TestGraceRunsOutThenBlockedThenAutoUnblock(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("64.137.54.87", "NL")
	h.step(time.Second)
	h.probe("95.24.1.1", "RU")
	h.want(Suspect)

	h.step(21 * time.Second)
	h.want(Blocked)
	if !h.p.closed[claude.PID] {
		t.Fatal("claude not asked to close after grace")
	}
	// Still running after close_timeout: force.
	h.step(6 * time.Second)
	if !h.p.killed[claude.PID] {
		t.Fatal("claude not killed after close_timeout")
	}
	h.p.remove(claude.PID)
	h.step(500 * time.Millisecond)

	// Started again while blocked: killed at once.
	relaunched := procs.Proc{PID: 101, Name: "Claude", Path: claude.Path}
	h.p.procs = append(h.p.procs, relaunched)
	h.step(500 * time.Millisecond)
	if !h.p.killed[101] {
		t.Fatal("relaunch while blocked not killed")
	}
	if !h.had(EvKilledNew) {
		t.Fatal("no EvKilledNew")
	}
	h.p.remove(101)

	h.step(2 * time.Second)
	h.probe("64.137.54.87", "NL")
	h.want(OK)
	if !h.had(EvUnblocked) {
		t.Fatal("no EvUnblocked")
	}
}

func TestCountryAllowList(t *testing.T) {
	h := newHarness(t)
	h.g.cfg.Allow.Countries = []string{"NL"}
	h.step(0)
	h.probe("64.137.54.87", "NL")
	h.want(OK)
	// The VPN rotated to another NL address: still fine.
	h.step(time.Second)
	h.probe("185.10.10.10", "NL")
	h.want(OK)
	h.step(time.Second)
	h.probe("185.10.10.11", "DE")
	h.want(Suspect)
	// Unknown country (lookup failed) is not allowed.
	h.step(time.Second)
	h.probe("185.10.10.12", "")
	h.want(Suspect)
}

func TestOffline(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("64.137.54.87", "NL")
	h.step(time.Second)
	h.offline()
	h.want(Suspect)
	if s := h.g.Snapshot(h.now); s.Reason != Offline || !s.Offline {
		t.Fatalf("snapshot %+v", s)
	}
	h.step(time.Second)
	h.probe("64.137.54.87", "NL")
	h.want(OK)
}

func TestOfflineIgnoredWhenDisabled(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.g.cfg.BlockWhenOffline = false
	h.step(0)
	h.offline()
	h.want(OK)
}

func TestAllowCurrentIPFromMenu(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("5.5.5.5", "DE")
	h.want(Suspect)
	if !h.g.AllowCurrentIP(h.now) {
		t.Fatal("AllowCurrentIP returned false")
	}
	h.want(OK)
	if h.p.frozen[claude.PID] {
		t.Fatal("still frozen")
	}
	if len(h.saved.Allow.IPs) != 2 {
		t.Fatalf("not saved: %v", h.saved.Allow.IPs)
	}
}

func TestAllowCountryFromMenu(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("5.5.5.5", "DE")
	h.g.AllowCurrentCountry(h.now)
	h.want(OK)
	if h.saved.Allow.Countries[0] != "DE" {
		t.Fatal("country not saved")
	}
}

func TestPause(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("5.5.5.5", "DE")
	h.want(Suspect)
	h.g.Pause(h.now, 15*time.Minute)
	h.want(OK)
	if h.p.frozen[claude.PID] {
		t.Fatal("frozen during pause")
	}
	// Network changes during the pause do nothing.
	h.fp = "en0=10.0.0.2"
	h.step(time.Second)
	h.probe("5.5.5.5", "DE")
	h.want(OK)
	// Pause ends: protection is back.
	h.step(15 * time.Minute)
	h.probe("5.5.5.5", "DE")
	h.want(Suspect)
}

func TestShutdownResumes(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("5.5.5.5", "DE")
	h.g.Shutdown()
	if h.p.frozen[claude.PID] {
		t.Fatal("left frozen on shutdown")
	}
}

func TestToggleApp(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("5.5.5.5", "DE")
	h.g.ToggleApp(h.now, "claude")
	if h.p.frozen[claude.PID] {
		t.Fatal("app removed from protection must be resumed")
	}
	h.g.ToggleApp(h.now, "claude")
	if !h.p.frozen[claude.PID] {
		t.Fatal("app added back must be frozen while suspect")
	}
}

func TestProbeScheduling(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	if h.probes != 1 {
		t.Fatalf("probes = %d", h.probes)
	}
	h.step(time.Second)
	if h.probes != 1 {
		t.Fatal("second probe started while the first is running")
	}
	h.probe("64.137.54.87", "NL")
	h.step(3 * time.Second)
	if h.probes != 1 {
		t.Fatal("probe before check_interval")
	}
	h.step(3 * time.Second)
	if h.probes != 2 {
		t.Fatalf("no probe after check_interval, probes = %d", h.probes)
	}
	h.probe("64.137.54.87", "NL")
	h.fp = "changed"
	h.step(500 * time.Millisecond)
	if h.probes != 3 {
		t.Fatal("network change must trigger an immediate probe")
	}
}

func TestConfigEditResumesWithNotice(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	h.probe("5.5.5.5", "DE")
	c := h.g.cfg.Clone()
	c.Allow.Countries = []string{"DE"}
	h.events = nil
	h.g.SetConfig(h.now, c)
	h.want(OK)
	if h.p.frozen[claude.PID] {
		t.Fatal("still frozen")
	}
	if !h.had(EvResumed) {
		t.Fatal("allowing via config.yaml must say the app is running again")
	}
}

func TestIPv6Leak(t *testing.T) {
	h := newHarness(t)
	h.g.cfg.Allow.Countries = []string{"NL"}
	h.step(0)
	// The VPN tunnels IPv4 only; IPv6 goes out directly from Russia.
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2, IP6: "2a02:2698:1:2::5", Country6: "RU"})
	h.want(Suspect)
	if s := h.g.Snapshot(h.now); s.Reason != IPv6Leak || s.Allowed() {
		t.Fatalf("reason = %v, allowed = %v", s.Reason, s.Allowed())
	}
	if !h.p.frozen[claude.PID] {
		t.Fatal("not frozen on IPv6 leak")
	}
	// IPv6 through the VPN too: fine.
	h.step(time.Second)
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2, IP6: "2a0e:1c80:1::7", Country6: "NL"})
	h.want(OK)
	// IPv6 check off: the leak is ignored.
	h.g.cfg.CheckIPv6 = false
	h.step(time.Second)
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2, IP6: "2a02:2698:1:2::5", Country6: "RU"})
	h.want(OK)
}

func TestFirstRunAllowsIPv6Network(t *testing.T) {
	h := newHarness(t)
	h.step(0)
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2, IP6: "2a0e:1c80:1:2:aaaa:bbbb:cccc:dddd", Country6: "NL"})
	h.want(OK)
	if got := h.saved.Allow.IPs; len(got) != 2 || got[1] != "2a0e:1c80:1:2::/64" {
		t.Fatalf("allowed = %v", got)
	}
	// A new privacy address in the same /64 is still allowed.
	h.step(time.Second)
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2, IP6: "2a0e:1c80:1:2:1111:2222:3333:4444"})
	h.want(OK)
}

func TestUnconfirmedIP(t *testing.T) {
	h := newHarness(t, "64.137.54.87")
	h.step(0)
	// Only one service answered: not enough to unfreeze.
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 1})
	h.want(Suspect)
	if h.g.Snapshot(h.now).Reason != Unconfirmed {
		t.Fatal("want Unconfirmed")
	}
	// Two agree, but a third saw another address: looks like a leak.
	h.step(time.Second)
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 2, Others: []string{"95.24.1.1"}})
	h.want(Suspect)
	h.step(time.Second)
	h.probe("64.137.54.87", "NL")
	h.want(OK)
	// With a single self-hosted service, one answer is enough.
	h.g.cfg.IPServices = []string{"https://ip.example.com"}
	h.step(10 * time.Second)
	h.probeFull(ProbeResult{IP: "64.137.54.87", Country: "NL", Votes: 1})
	h.want(OK)
}
