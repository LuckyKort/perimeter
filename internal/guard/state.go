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
)

type Event struct {
	At            time.Time
	Kind          EventKind
	Reason        Reason
	IP, Country   string
	IP6, Country6 string
	Others        []string
	Apps          []string
	Dur           time.Duration
	Notify        bool // worth a system notification, not just a line in the history
}

type Snapshot struct {
	Status        Status
	Reason        Reason
	IP, Country   string
	IP6, Country6 string
	Others        []string // addresses other IP services reported
	Offline       bool
	Checking      bool
	GraceLeft     time.Duration // in Suspect: time until the apps are closed
	PausedUntil   time.Time     // zero if not paused
	Running       []procs.Proc  // protected apps' processes right now
	Frozen        int
	Config        *config.Config
	History       []Event
}

func (g *Guard) snapshot(now time.Time) Snapshot {
	s := Snapshot{
		Status: g.status, Reason: g.reason,
		IP: g.ip, Country: g.country, IP6: g.ip6, Country6: g.cc6, Others: append([]string(nil), g.others...),
		Offline: g.offline, Checking: g.probing,
		Running: append([]procs.Proc(nil), g.running...),
		Frozen:  len(g.frozen),
		Config:  g.cfg.Clone(),
		History: append([]Event(nil), g.history...),
	}
	if g.paused(now) {
		s.PausedUntil = g.pausedUntil
	}
	if g.status == Suspect {
		s.GraceLeft = g.cfg.Grace.D() - now.Sub(g.since)
		if s.GraceLeft < 0 {
			s.GraceLeft = 0
		}
	}
	return s
}

// Paused reports whether protection is paused.
func (s Snapshot) Paused() bool { return !s.PausedUntil.IsZero() }

// Allowed reports whether the current IPv4 (and IPv6, if any) is on the allow-list.
func (s Snapshot) Allowed() bool {
	if s.IP == "" || !s.Config.Allow.Matches(s.IP, s.Country) {
		return false
	}
	return !s.Config.CheckIPv6 || s.IP6 == "" || s.Config.Allow.Matches(s.IP6, s.Country6)
}

// key changes whenever something visible in the menu changes (the grace countdown by the second).
func (s Snapshot) key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d|%d|%s|%s|%s|%s|%v|%t|%d|%d|%v|%d|", s.Status, s.Reason, s.IP, s.Country, s.IP6, s.Country6, s.Others, s.Offline,
		int(s.GraceLeft.Seconds()), s.Frozen, s.PausedUntil.Unix(), len(s.Running))
	for _, a := range procs.AppNames(s.Running) {
		b.WriteString(a)
	}
	fmt.Fprintf(&b, "|%v|%v|%v", s.Config.Allow, s.Config.Apps, len(s.Config.CustomApps))
	return b.String()
}
