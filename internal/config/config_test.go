package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRenderParseRoundTrip(t *testing.T) {
	for _, lang := range []string{"ru", "en"} {
		CommentLang = lang
		c := Default()
		c.Apps = []string{"claude", "cursor"}
		c.CustomApps = []CustomApp{{Name: "My App", Match: []string{"/MyApp.app/", "myapp.exe"}}}
		c.Allow.IPs = []string{"64.137.54.87", "10.0.0.0/8"}
		c.Allow.Countries = []string{"NL", "DE"}
		c.Grace = Duration(90 * time.Second)
		c.MenuBar = "country"
		got, err := Parse([]byte(Render(c)))
		if err != nil {
			t.Fatalf("%s: %v\n%s", lang, err, Render(c))
		}
		if Render(got) != Render(c) {
			t.Fatalf("%s: round trip differs:\n%s\n---\n%s", lang, Render(c), Render(got))
		}
		if got.Grace.D() != 90*time.Second || got.CustomApps[0].Match[1] != "myapp.exe" {
			t.Fatalf("%s: values lost: %+v", lang, got)
		}
	}
}

func TestDurationString(t *testing.T) {
	for d, want := range map[time.Duration]string{20 * time.Second: "20s", 90 * time.Second: "90s", 2 * time.Minute: "2m", 0: "0s"} {
		if got := Duration(d).String(); got != want {
			t.Errorf("%v -> %q, want %q", d, got, want)
		}
	}
}

func TestParseValidation(t *testing.T) {
	bad := []string{
		"allow:\n  countries: [Netherlands]",
		"allow:\n  ips: [\"not-an-ip\"]",
		"grace: soon",
		"custom_apps:\n  - name: X",
	}
	for _, s := range bad {
		if _, err := Parse([]byte(s)); err == nil {
			t.Errorf("accepted bad config %q", s)
		}
	}
	c, err := Parse([]byte("allow:\n  countries: [nl]\ncheck_interval: 1s\nlanguage: fr\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Allow.Countries[0] != "NL" || c.CheckInterval.D() != 3*time.Second || c.Language != "auto" {
		t.Fatalf("not normalized: %+v", c)
	}
	if !c.FreezeOnNetworkChange || c.Grace.D() != 20*time.Second {
		t.Fatal("missing keys must keep defaults")
	}
}

func TestLoadCreatesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.yaml")
	c, created, err := Load(p)
	if err != nil || !created || c.Apps[0] != "claude" {
		t.Fatalf("first load: %v %v %+v", err, created, c)
	}
	_, created, err = Load(p)
	if err != nil || created {
		t.Fatalf("second load: %v %v", err, created)
	}
}

func TestAllowMatches(t *testing.T) {
	a := Allow{IPs: []string{"64.137.54.87", "10.1.0.0/16"}, Countries: []string{"NL"}}
	cases := []struct {
		ip, cc string
		want   bool
	}{
		{"64.137.54.87", "", true},
		{"64.137.54.88", "", false},
		{"10.1.200.3", "RU", true},
		{"10.2.0.1", "RU", false},
		{"185.1.1.1", "NL", true},
		{"185.1.1.1", "nl", true},
		{"185.1.1.1", "", false},
		{"garbage", "NL", false},
	}
	for _, c := range cases {
		if got := a.Matches(c.ip, c.cc); got != c.want {
			t.Errorf("Matches(%q,%q) = %v", c.ip, c.cc, got)
		}
	}
	if !(Allow{}).IsEmpty() || a.IsEmpty() {
		t.Fatal("IsEmpty")
	}
	if a.AddIP("64.137.54.87") || !a.AddIP("1.1.1.1") || a.AddCountry("nl") || !a.AddCountry("de") {
		t.Fatal("Add dedup")
	}
}

func TestAddIPv6AsNetwork(t *testing.T) {
	var a Allow
	if !a.AddIP("2a02:2698:1:2:aaaa::1") || a.IPs[0] != "2a02:2698:1:2::/64" {
		t.Fatalf("%v", a.IPs)
	}
	if a.AddIP("2a02:2698:1:2:bbbb::2") {
		t.Fatal("same /64 added twice")
	}
	if !a.Matches("2a02:2698:1:2:cccc::3", "") {
		t.Fatal("/64 doesn't match")
	}
	a.IPs = []string{"10.0.0.0/8"}
	if a.AddIP("10.1.2.3") {
		t.Fatal("address inside an allowed subnet added again")
	}
}
