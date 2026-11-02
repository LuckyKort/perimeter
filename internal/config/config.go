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

type Allow struct {
	IPs       []string `yaml:"ips"`
	Countries []string `yaml:"countries"`
}

type Config struct {
	Apps       []string    `yaml:"apps"`
	CustomApps []CustomApp `yaml:"custom_apps"`
	Allow      Allow       `yaml:"allow"`

	FreezeOnNetworkChange bool     `yaml:"freeze_on_network_change"`
	BlockWhenOffline      bool     `yaml:"block_when_offline"`
	Grace                 Duration `yaml:"grace"`
	CloseTimeout          Duration `yaml:"close_timeout"`
	CheckInterval         Duration `yaml:"check_interval"`

	IPServices    []string `yaml:"ip_services"`
	MenuBar       string   `yaml:"menubar"`
	Language      string   `yaml:"language"`
	Notifications bool     `yaml:"notifications"`
}

var DefaultIPServices = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://v4.ident.me",
}

func Default() *Config {
	return &Config{
		Apps:                  []string{"claude"},
		FreezeOnNetworkChange: true,
		BlockWhenOffline:      true,
		Grace:                 Duration(20 * time.Second),
		CloseTimeout:          Duration(5 * time.Second),
		CheckInterval:         Duration(10 * time.Second),
		IPServices:            append([]string(nil), DefaultIPServices...),
		MenuBar:               "ip",
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
	switch c.MenuBar {
	case "ip", "country", "icon":
	default:
		c.MenuBar = "ip"
	}
	switch c.Language {
	case "auto", "ru", "en":
	default:
		c.Language = "auto"
	}
	return nil
}

// Clone returns a deep copy, so the guard and the menu never share slices.
func (c *Config) Clone() *Config {
	n := *c
	n.Apps = append([]string(nil), c.Apps...)
	n.CustomApps = append([]CustomApp(nil), c.CustomApps...)
	n.Allow.IPs = append([]string(nil), c.Allow.IPs...)
	n.Allow.Countries = append([]string(nil), c.Allow.Countries...)
	n.IPServices = append([]string(nil), c.IPServices...)
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
