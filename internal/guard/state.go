package guard

import (
	"fmt"
	"strings"
	"time"

	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/procs"
)

type EventKind int

const (
	EvAutoAllowed EventKind = iota // first run: current IP became the allowed one
	EvNetChanged                   // local network changed
	EvIPChanged                    // public IP changed
	EvOffline                      // internet is gone
	EvOnline                       // internet is back
	EvFrozen                       // apps frozen (Reason says why)
	EvResumed                      // allowed IP is back within grace, apps continue
	EvBlocked                      // grace ran out, apps closed
	EvKilledNew                    // an app was started while blocked
	EvUnblocked                    // allowed IP is back after a block
	EvPaused
	EvUnpaused
	EvAllowedIP
	EvAllowedCountry
	EvDNSLeak // DNS resolver is elsewhere than the VPN (dns_check: warn)
	EvTrusted // on a trusted network (or it was just added)
)

type Event struct {
	At              time.Time
	Kind            EventKind
	Reason          Reason
	IP, Country     string
	IP6, Country6   string
	Others          []string
	DNS, DNSCountry string
	Network         string
	Apps            []string
	Dur             time.Duration
	Notify          bool // worth a system notification, not just a line in the history
}

// GroupState is one set of apps sharing rules, for the panel.
type GroupState struct {
	Key       string // "" for the shared rules
	Apps      []string
	Status    Status
	Reason    Reason
	GraceLeft time.Duration
	Running   []string
	Frozen    int
}

type Snapshot struct {
	Status          Status // the worst of the groups
	Reason          Reason
	IP, Country     string
	IP6, Country6   string
	Others          []string // addresses other IP services reported
	DNS, DNSCountry string
	DNSLeak         bool
	NetID           string
	Network         string // name of the trusted network we're on, if any
	Offline         bool
	Checking        bool
	AllowedNow      bool          // the shared rules allow the current network
	GraceLeft       time.Duration // shortest time until apps are closed
	PausedUntil     time.Time     // zero if not paused
	Running         []procs.Proc  // protected apps' processes right now
	Frozen          int
	Groups          []GroupState
	Counters        Counters
	Config          *config.Config
	History         []Event
}

func (g *Guard) snapshot(now time.Time) Snapshot {
	s := Snapshot{
		Status: OK,
		IP:     g.ip, Country: g.country, IP6: g.ip6, Country6: g.cc6, Others: append([]string(nil), g.others...),
		DNS: g.dns, DNSCountry: g.dnsCC, NetID: g.netID,
		Offline: g.offline, Checking: g.probing,
		Counters: g.counters,
		Config:   g.cfg.Clone(),
		History:  append([]Event(nil), g.history...),
	}
	if n, ok := g.cfg.Trusted(g.netID); ok {
		s.Network = n.Name
	}
	s.DNSLeak = g.cfg.DNSCheck != "off" && s.Network == "" && g.dnsLeak(g.cfg.Allow)
	if g.paused(now) {
		s.PausedUntil = g.pausedUntil
	}
	first := true
	for _, gr := range g.groups {
		gs := GroupState{Key: gr.key, Apps: gr.names(), Status: gr.status, Reason: gr.reason, Frozen: len(gr.frozen)}
		if gr.status == Suspect {
			gs.GraceLeft = max(0, g.cfg.Grace.D()-now.Sub(gr.since))
		}
		gs.Running = procs.AppNames(gr.running)
		s.Running = append(s.Running, gr.running...)
		s.Frozen += len(gr.frozen)
		if gr.key == "" {
			s.AllowedNow = g.ip != "" && !g.offline && g.verdict(gr) == NoReason
		}
		if first || worse(gr.status, s.Status) || (gr.status == Suspect && s.Status == Suspect && gs.GraceLeft < s.GraceLeft) {
			s.Status, s.Reason, s.GraceLeft = gr.status, gr.reason, gs.GraceLeft
			first = false
		}
		s.Groups = append(s.Groups, gs)
	}
	return s
}

// Paused reports whether protection is paused.
func (s Snapshot) Paused() bool { return !s.PausedUntil.IsZero() }

// Allowed reports whether the shared rules allow the current network.
func (s Snapshot) Allowed() bool { return s.AllowedNow }

// key changes whenever something visible in the menu changes (the grace countdown by the second).
func (s Snapshot) key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d|%d|%s|%s|%s|%s|%v|%s|%s|%s|%t|%d|%d|%v|%d|", s.Status, s.Reason, s.IP, s.Country, s.IP6, s.Country6,
		s.Others, s.DNS, s.DNSCountry, s.Network, s.Offline, int(s.GraceLeft.Seconds()), s.Frozen, s.PausedUntil.Unix(), len(s.Running))
	for _, gr := range s.Groups {
		fmt.Fprintf(&b, "%d%d%d%v;", gr.Status, gr.Reason, int(gr.GraceLeft.Seconds()), gr.Running)
	}
	fmt.Fprintf(&b, "|%v|%v|%v|%v|%v|%s", s.Config.Allow, s.Config.Apps, len(s.Config.CustomApps), s.Config.AppRules, s.Config.TrustedNetworks, s.Config.MenuBar)
	return b.String()
}
