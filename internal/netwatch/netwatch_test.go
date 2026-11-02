package netwatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFingerprintFilters(t *testing.T) {
	addrs := []ifaceAddr{
		{"en0", "192.168.0.10"},
		{"utun6", "172.19.0.1"},
		{"utun0", "fe80::1"},          // IPv6: ignored
		{"bridge100", "192.168.64.1"}, // VM bridge: ignored
		{"en0", "169.254.3.3"},        // link-local: ignored
	}
	if got := fingerprintOf(addrs, "darwin"); got != "en0=192.168.0.10,utun6=172.19.0.1" {
		t.Fatalf("darwin: %q", got)
	}
	win := []ifaceAddr{{"Wi-Fi", "192.168.1.5"}, {"vEthernet (WSL)", "172.20.0.1"}, {"WireGuard Tunnel", "10.8.0.2"}}
	if got := fingerprintOf(win, "windows"); got != "Wi-Fi=192.168.1.5,WireGuard Tunnel=10.8.0.2" {
		t.Fatalf("windows: %q", got)
	}
	// Order of interfaces doesn't matter.
	if fingerprintOf([]ifaceAddr{{"a", "1.1.1.1"}, {"b", "2.2.2.2"}}, "linux") != fingerprintOf([]ifaceAddr{{"b", "2.2.2.2"}, {"a", "1.1.1.1"}}, "linux") {
		t.Fatal("not sorted")
	}
}

func TestParseScutil(t *testing.T) {
	out := `<dictionary> {
  ExceptionsList : <array> {
    0 : *.local
  }
  HTTPEnable : 1
  HTTPPort : 7890
  HTTPProxy : 127.0.0.1
  HTTPSEnable : 1
  HTTPSPort : 7890
  HTTPSProxy : 127.0.0.1
}`
	if u := parseScutil(out); u == nil || u.String() != "http://127.0.0.1:7890" {
		t.Fatalf("got %v", u)
	}
	if u := parseScutil("<dictionary> {\n  SOCKSEnable : 1\n  SOCKSProxy : 127.0.0.1\n  SOCKSPort : 1080\n}"); u == nil || u.String() != "socks5://127.0.0.1:1080" {
		t.Fatalf("socks: %v", u)
	}
	if u := parseScutil("<dictionary> {\n  HTTPEnable : 0\n}"); u != nil {
		t.Fatalf("disabled: %v", u)
	}
}

func TestParseWinProxy(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:10809":                          "http://127.0.0.1:10809",
		"http=127.0.0.1:8080;https=127.0.0.1:8443": "http://127.0.0.1:8443",
		"socks=127.0.0.1:1080":                     "socks5://127.0.0.1:1080",
	}
	for in, want := range cases {
		if u := parseWinProxy(true, in); u == nil || u.String() != want {
			t.Errorf("%q -> %v, want %s", in, u, want)
		}
	}
	if parseWinProxy(false, "127.0.0.1:1") != nil || parseWinProxy(true, "") != nil {
		t.Error("disabled/empty must be nil")
	}
}

func TestPublicIPFirstValidWins(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "<html>nope") }))
	defer bad.Close()
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, "9.9.9.9")
	}))
	defer slow.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Connection") != "close" && !r.Close {
			t.Error("keep-alive must be off")
		}
		fmt.Fprint(w, "64.137.54.87\n")
	}))
	defer good.Close()

	good2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "64.137.54.87") }))
	defer good2.Close()
	liar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "95.24.1.1") }))
	defer liar.Close()

	start := time.Now()
	r, err := PublicIP(context.Background(), []string{bad.URL, slow.URL, good.URL, good2.URL}, 4*time.Second, false, 2)
	if err != nil || r.IP != "64.137.54.87" || !r.Confirmed(2) {
		t.Fatalf("got %+v %v", r, err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("waited for the slow service")
	}
	// One honest service and one disagreeing: not confirmed.
	r, err = PublicIP(context.Background(), []string{good.URL, liar.URL}, 2*time.Second, false, 2)
	if err != nil || r.Confirmed(2) || len(r.Others) != 1 {
		t.Fatalf("disagreement not reported: %+v %v", r, err)
	}
	// IPv4 answers don't count as IPv6.
	if _, err := PublicIP(context.Background(), []string{good.URL}, time.Second, true, 1); err != ErrOffline {
		t.Fatalf("v4 answer accepted as v6: %v", err)
	}
	if _, err := PublicIP(context.Background(), []string{bad.URL}, time.Second, false, 1); err != ErrOffline {
		t.Fatalf("all bad: %v", err)
	}
}

// TestChangesStarts makes sure the OS subscription opens and closes cleanly.
func TestChangesStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := Changes(ctx)
	select {
	case <-ch: // a change happened to arrive, fine
	case <-time.After(200 * time.Millisecond):
	}
	cancel()
}
