// Package config reads and writes Kordon's config.yaml.
//
// The file is meant to be edited by hand, so it is always written back from a
// commented template (see render.go) instead of a bare yaml.Marshal.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that reads and writes as "20s", "1m30s".
type Duration time.Duration

func (d Duration) D() time.Duration { return time.Duration(d) }

func (d Duration) String() string {
	sec := int64(time.Duration(d) / time.Second)
	if sec >= 60 && sec%60 == 0 {
		return fmt.Sprintf("%dm", sec/60)
	}
	return fmt.Sprintf("%ds", sec)
}

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	v, err := time.ParseDuration(strings.TrimSpace(n.Value))
	if err != nil {
		return fmt.Errorf("строка %d: %q - не длительность (пример: 20s, 2m)", n.Line, n.Value)
	}
	*d = Duration(v)
	return nil
}

// CustomApp is an app that is not in the built-in presets.
type CustomApp struct {
	Name  string   `yaml:"name"`
	Match []string `yaml:"match"`
}

// TrustedNetwork is a local network (identified by its router) where apps may run as they are.
type TrustedNetwork struct {
	Name string `yaml:"name"`
	ID   string `yaml:"id"`
}

type Allow struct {
	IPs       []string `yaml:"ips"`
	Countries []string `yaml:"countries"`
}

type Config struct {
	Apps       []string    `yaml:"apps"`
	CustomApps []CustomApp `yaml:"custom_apps"`
	Allow      Allow       `yaml:"allow"`
	// AppRules gives an app its own allow-list instead of the shared one.
	// The key is a preset id (cursor) or a custom app's name.
	AppRules        map[string]Allow `yaml:"app_rules"`
	TrustedNetworks []TrustedNetwork `yaml:"trusted_networks"`

	FreezeOnNetworkChange bool     `yaml:"freeze_on_network_change"`
	BlockWhenOffline      bool     `yaml:"block_when_offline"`
	Grace                 Duration `yaml:"grace"`
	CloseTimeout          Duration `yaml:"close_timeout"`
	CheckInterval         Duration `yaml:"check_interval"`

	IPServices    []string `yaml:"ip_services"`
	MinAgree      int      `yaml:"min_agree"`
	CheckIPv6     bool     `yaml:"check_ipv6"`
	DNSCheck      string   `yaml:"dns_check"`
	IPv6Services  []string `yaml:"ipv6_services"`
	MenuBar       string   `yaml:"menubar"`
	UpdateCheck   bool     `yaml:"update_check"`
	UsagePing     string   `yaml:"usage_ping"` // ask, on, off: count me among Kordon's users (anonymous)
	Language      string   `yaml:"language"`
	Notifications bool     `yaml:"notifications"`
}

var DefaultIPServices = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://v4.ident.me",
	"https://checkip.amazonaws.com",
	"https://ipinfo.io/ip",
}

var DefaultIPv6Services = []string{
	"https://api6.ipify.org",
	"https://ipv6.icanhazip.com",
	"https://v6.ident.me",
}

func Default() *Config {
	return &Config{
		Apps:                  []string{"claude"},
		FreezeOnNetworkChange: true,
		BlockWhenOffline:      true,
		Grace:                 Duration(20 * time.Second),
		CloseTimeout:          Duration(5 * time.Second),
		CheckInterval:         Duration(5 * time.Second),
		IPServices:            append([]string(nil), DefaultIPServices...),
		MinAgree:              2,
		CheckIPv6:             true,
		DNSCheck:              "warn",
		IPv6Services:          append([]string(nil), DefaultIPv6Services...),
		MenuBar:               "ip",
		UpdateCheck:           true,
		UsagePing:             "ask",
		Language:              "auto",
		Notifications:         true,
	}
}

// Parse reads YAML on top of the defaults, so a missing key keeps its default.
func Parse(data []byte) (*Config, error) {
	c := Default()
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if err := c.normalize(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) normalize() error {
	for i, cc := range c.Allow.Countries {
		cc = strings.ToUpper(strings.TrimSpace(cc))
		if len(cc) != 2 {
			return fmt.Errorf("allow.countries: %q - нужен двухбуквенный код страны (NL, DE)", cc)
		}
		c.Allow.Countries[i] = cc
	}
	for _, s := range c.Allow.IPs {
		if _, err := ParseIPOrNet(s); err != nil {
			return fmt.Errorf("allow.ips: %w", err)
		}
	}
	for i, a := range c.Apps {
		c.Apps[i] = strings.ToLower(strings.TrimSpace(a))
	}
	for _, a := range c.CustomApps {
		if strings.TrimSpace(a.Name) == "" || len(a.Match) == 0 {
			return errors.New("custom_apps: у каждого приложения нужны name и match")
		}
	}
	if c.Grace < 0 || c.CloseTimeout < 0 {
		return errors.New("grace и close_timeout не могут быть отрицательными")
	}
	if c.CheckInterval.D() < 3*time.Second {
		c.CheckInterval = Duration(3 * time.Second)
	}
	if len(c.IPServices) == 0 {
		c.IPServices = append([]string(nil), DefaultIPServices...)
	}
	if len(c.IPv6Services) == 0 {
		c.IPv6Services = append([]string(nil), DefaultIPv6Services...)
	}
	if c.MinAgree < 1 {
		c.MinAgree = 1
	}
	switch c.DNSCheck {
	case "warn", "block", "off":
	default:
		c.DNSCheck = "warn"
	}
	rules := map[string]Allow{}
	for k, a := range c.AppRules {
		for i, cc := range a.Countries {
			a.Countries[i] = strings.ToUpper(strings.TrimSpace(cc))
		}
		for _, ip := range a.IPs {
			if _, err := ParseIPOrNet(ip); err != nil {
				return fmt.Errorf("app_rules.%s: %w", k, err)
			}
		}
		rules[strings.ToLower(strings.TrimSpace(k))] = a
	}
	c.AppRules = rules
	switch c.MenuBar {
	case "ip", "country", "icon":
	default:
		c.MenuBar = "ip"
	}
	switch c.UsagePing {
	case "ask", "on", "off":
	default:
		c.UsagePing = "ask"
	}
	switch c.Language {
	case "auto", "ru", "en":
	default:
		c.Language = "auto"
	}
	return nil
}

// Trusted returns the trusted network with this id, if any.
func (c *Config) Trusted(id string) (TrustedNetwork, bool) {
	for _, n := range c.TrustedNetworks {
		if id != "" && n.ID == id {
			return n, true
		}
	}
	return TrustedNetwork{}, false
}

// Need is how many IP services must agree; never more than there are services.
func (c *Config) Need() int { return min(c.MinAgree, len(c.IPServices)) }

// Need6 is the same for IPv6.
func (c *Config) Need6() int { return min(c.MinAgree, len(c.IPv6Services)) }

// Clone returns a deep copy, so the guard and the menu never share slices.
func (c *Config) Clone() *Config {
	n := *c
	n.Apps = append([]string(nil), c.Apps...)
	n.CustomApps = append([]CustomApp(nil), c.CustomApps...)
	n.Allow.IPs = append([]string(nil), c.Allow.IPs...)
	n.Allow.Countries = append([]string(nil), c.Allow.Countries...)
	n.IPServices = append([]string(nil), c.IPServices...)
	n.IPv6Services = append([]string(nil), c.IPv6Services...)
	n.TrustedNetworks = append([]TrustedNetwork(nil), c.TrustedNetworks...)
	n.AppRules = make(map[string]Allow, len(c.AppRules))
	for k, a := range c.AppRules {
		n.AppRules[k] = Allow{IPs: append([]string(nil), a.IPs...), Countries: append([]string(nil), a.Countries...)}
	}
	return &n
}

// Load reads the file, creating it with defaults on first run.
// created reports whether the file did not exist before.
func Load(path string) (c *Config, created bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		c = Default()
		return c, true, Save(path, c)
	}
	if err != nil {
		return nil, false, err
	}
	c, err = Parse(data)
	return c, false, err
}

// Save writes atomically: a half-written config must never be read back.
func Save(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(Render(c)), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
