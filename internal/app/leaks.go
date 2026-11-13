package app

import (
	"context"
	"net/url"
	"time"

	"perimeter/internal/config"
	"perimeter/internal/netwatch"
)

// LeakReport is a one-off deep check from the panel: every IP service on its
// own, IPv6, the DNS resolver, and - when a system proxy is set - the address
// of programs that don't use it. The panel turns it into verdicts.
type LeakReport struct {
	At     time.Time `json:"at"`
	Proxy  string    `json:"proxy"` // host:port of the system proxy, "" if none
	V4     []LeakIP  `json:"v4"`
	Direct []LeakIP  `json:"direct"` // without the proxy; empty when there is none
	V6     []LeakIP  `json:"v6"`
	V6Off  bool      `json:"v6off"` // IPv6 checking is switched off in the settings
	DNS    *LeakIP   `json:"dns"`
}

// LeakIP is one address with its country and whether the shared rules allow it.
type LeakIP struct {
	Service string `json:"service,omitempty"`
	IP      string `json:"ip"`
	Country string `json:"country"`
	Allowed bool   `json:"allowed"`
	Error   bool   `json:"error,omitempty"`
}

// CheckLeaks runs the whole check; it takes a few seconds.
func CheckLeaks(ctx context.Context, cfg *config.Config) LeakReport {
	geo := netwatch.NewGeo()
	const timeout = 8 * time.Second
	r := LeakReport{At: time.Now()}
	if p := netwatch.SystemProxy(); p != nil {
		r.Proxy = hostOf(p)
	}
	mark := func(as []netwatch.Answer) []LeakIP {
		out := make([]LeakIP, 0, len(as))
		for _, a := range as {
			l := LeakIP{Service: a.Service, IP: a.IP, Error: a.Error}
			if !a.Error {
				l.Country = geo.Country(ctx, a.IP, timeout)
				l.Allowed = cfg.Allow.Matches(a.IP, l.Country)
			}
			out = append(out, l)
		}
		return out
	}
	r.V4 = mark(netwatch.EachService(ctx, cfg.IPServices, timeout, false, false))
	if r.Proxy != "" {
		// Two services are enough to see where the proxy-less traffic goes.
		r.Direct = mark(netwatch.EachService(ctx, cfg.IPServices[:min(2, len(cfg.IPServices))], timeout, false, true))
	}
	if cfg.CheckIPv6 {
		r.V6 = mark(netwatch.EachService(ctx, cfg.IPv6Services, timeout, true, false))
	} else {
		r.V6Off = true
	}
	if ip := netwatch.DNSResolver(ctx); ip != "" {
		c := geo.Country(ctx, ip, timeout)
		r.DNS = &LeakIP{IP: ip, Country: c}
	}
	return r
}

func hostOf(u *url.URL) string { return u.Host }
