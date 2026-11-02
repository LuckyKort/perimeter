package netwatch

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrOffline = errors.New("offline")

// client builds a fresh HTTP client for one check: no keep-alive, because a reused
// connection stays on the old route and reports the old IP after the VPN switches;
// the system proxy read right now; IPv4 only, since allow-lists are IPv4.
func client(proxy *url.URL, timeout time.Duration) *http.Client {
	d := &net.Dialer{Timeout: timeout}
	tr := &http.Transport{
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: timeout,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return d.DialContext(ctx, "tcp4", addr)
		},
	}
	if proxy != nil {
		tr.Proxy = http.ProxyURL(proxy)
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

// PublicIP asks all services at once and takes the first valid IPv4 answer.
func PublicIP(ctx context.Context, services []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	c := client(SystemProxy(), timeout)
	res := make(chan string, len(services))
	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			t, err := getText(ctx, c, u)
			if a, perr := netip.ParseAddr(t); err == nil && perr == nil && a.Is4() {
				res <- a.String()
			}
		}(s)
	}
	go func() { wg.Wait(); close(res) }()
	if ip, ok := <-res; ok {
		return ip, nil
	}
	return "", ErrOffline
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
	c := client(SystemProxy(), timeout)
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
