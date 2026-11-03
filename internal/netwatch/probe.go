package netwatch

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrOffline = errors.New("offline")

// client builds a fresh HTTP client for one check: no keep-alive, because a reused
// connection stays on the old route and reports the old IP after the VPN switches;
// the system proxy read right now; one address family, so IPv4 and IPv6 are
// checked separately (a VPN that tunnels only IPv4 leaks the real IPv6).
func client(proxy *url.URL, timeout time.Duration, network string) *http.Client {
	d := &net.Dialer{Timeout: timeout}
	tr := &http.Transport{
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: timeout,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return d.DialContext(ctx, network, addr)
		},
	}
	if proxy != nil {
		tr.Proxy = http.ProxyURL(proxy)
		// The proxy is usually 127.0.0.1: reach it any way, it picks the family upstream,
		// and the answer is still filtered by family in PublicIP.
		tr.DialContext = func(ctx context.Context, _, addr string) (net.Conn, error) {
			return d.DialContext(ctx, "tcp", addr)
		}
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

func getText(ctx context.Context, c *http.Client, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "curl/8.7.1") // services answer plain text to curl
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.TrimSpace(string(b)), err
}

// IPResult is what the IP services said.
type IPResult struct {
	IP     string   // the address most services agree on
	Votes  int      // how many services returned it
	Others []string // different addresses other services returned: a sign of a leak or a lying service
}

// Confirmed reports whether at least need services agree and none disagrees.
func (r IPResult) Confirmed(need int) bool { return r.Votes >= need && len(r.Others) == 0 }

// PublicIP asks all services at once and returns as soon as need of them agree.
// v6 selects IPv6; with no IPv6 connectivity it returns ErrOffline quickly.
func PublicIP(ctx context.Context, services []string, timeout time.Duration, v6 bool, need int) (IPResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	network := "tcp4"
	if v6 {
		network = "tcp6"
	}
	c := client(SystemProxy(), timeout, network)
	answers := make(chan string, len(services))
	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			t, err := getText(ctx, c, u)
			a, perr := netip.ParseAddr(t)
			if err != nil || perr != nil || a.Is4() == v6 || a.Is4In6() {
				return
			}
			answers <- a.String()
		}(s)
	}
	go func() { wg.Wait(); close(answers) }()

	votes := map[string]int{}
	for ip := range answers {
		votes[ip]++
		if votes[ip] >= need {
			break
		}
	}
	return tally(votes)
}

func tally(votes map[string]int) (IPResult, error) {
	var r IPResult
	for ip, n := range votes {
		if n > r.Votes || (n == r.Votes && ip < r.IP) {
			r.IP, r.Votes = ip, n
		}
	}
	if r.IP == "" {
		return r, ErrOffline
	}
	for ip := range votes {
		if ip != r.IP {
			r.Others = append(r.Others, ip)
		}
	}
	sort.Strings(r.Others)
	return r, nil
}

// Geo looks up and caches the country of an IP.
type Geo struct {
	mu    sync.Mutex
	cache map[string]string
}

func NewGeo() *Geo { return &Geo{cache: map[string]string{}} }

var geoServices = []string{"https://ipinfo.io/%s/country", "https://ipapi.co/%s/country/", "https://api.country.is/%s"}

func (g *Geo) Country(ctx context.Context, ip string, timeout time.Duration) string {
	g.mu.Lock()
	cc, ok := g.cache[ip]
	g.mu.Unlock()
	if ok {
		return cc
	}
	// The IP is in the URL, so the lookup may go over any route.
	c := client(SystemProxy(), timeout, "tcp")
	for _, f := range geoServices {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		t, err := getText(ctx, c, strings.Replace(f, "%s", ip, 1))
		cancel()
		if err != nil {
			continue
		}
		if strings.HasPrefix(t, "{") { // country.is answers {"ip":"..","country":"NL"}
			if i := strings.Index(t, `"country":"`); i >= 0 && len(t) >= i+13 {
				t = t[i+11 : i+13]
			}
		}
		if len(t) == 2 && isLetters(t) {
			cc = strings.ToUpper(t)
			g.mu.Lock()
			g.cache[ip] = cc
			g.mu.Unlock()
			return cc
		}
	}
	return ""
}

func isLetters(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// HTTPClient is a plain client through the system proxy, for update checks and downloads.
func HTTPClient(timeout time.Duration) *http.Client {
	return client(SystemProxy(), timeout, "tcp")
}

// Answer is one IP service's reply in a leak check.
type Answer struct {
	Service string `json:"service"`
	IP      string `json:"ip"`
	Error   bool   `json:"error"`
}

// EachService asks every service and returns all the answers, unlike PublicIP
// that stops at the first agreement. direct skips the system proxy: that's the
// address programs see that ignore the proxy (most terminal tools do).
func EachService(ctx context.Context, services []string, timeout time.Duration, v6, direct bool) []Answer {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	network := "tcp4"
	if v6 {
		network = "tcp6"
	}
	proxy := SystemProxy()
	if direct {
		proxy = nil
	}
	c := client(proxy, timeout, network)
	out := make([]Answer, len(services))
	var wg sync.WaitGroup
	for i, s := range services {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			out[i] = Answer{Service: u, Error: true}
			t, err := getText(ctx, c, u)
			a, perr := netip.ParseAddr(t)
			if err == nil && perr == nil && a.Is4() != v6 && !a.Is4In6() {
				out[i] = Answer{Service: u, IP: a.String()}
			}
		}(i, s)
	}
	wg.Wait()
	return out
}
