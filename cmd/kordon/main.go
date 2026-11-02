// Kordon freezes chosen apps the moment your network or public IP changes,
// and closes them if an allowed IP doesn't come back.
//
//	kordon                 start in the menu bar / tray
//	kordon status          what's going on right now
//	kordon allow [IP|CC]   allow the current IP, a given IP/subnet or a country
//	kordon config          print the config path
//	kordon autostart on|off
//	kordon headless        run without a tray icon (servers, Linux without a tray)
//	kordon version
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofrs/flock"

	"github.com/ef1m03/kordon/internal/app"
	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/netwatch"
	"github.com/ef1m03/kordon/internal/ui"
)

var version = "dev"

func main() {
	attachConsole()
	if len(os.Args) > 1 {
		if err := cli(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "kordon:", err)
			os.Exit(1)
		}
		return
	}
	run(true)
}

func setupLang(cfg *config.Config) {
	ui.Lang = app.Language(cfg.Language)
	config.CommentLang = ui.Lang
}

func run(tray bool) {
	if err := os.MkdirAll(app.Dir(), 0o755); err != nil {
		fatal(err)
	}
	ui.Lang = app.Language("auto")
	if supervise() {
		return
	}
	// One instance only. A second one (double-click, autostart) just says where the first is.
	lock := flock.New(app.LockPath())
	if ok, _ := lock.TryLock(); !ok {
		if data, err := os.ReadFile(app.ConfigPath()); err == nil {
			if c, err := config.Parse(data); err == nil {
				setupLang(c)
			}
		}
		app.Notify(ui.T("already.t"), ui.T("already.b"))
		return // exit 0: launchd must not restart us for this
	}
	defer lock.Unlock()

	// The comment language of a fresh config.yaml follows the system.
	ui.Lang = app.Language("auto")
	config.CommentLang = ui.Lang
	cfg, created, err := config.Load(app.ConfigPath())
	if err != nil {
		app.Logf("config.yaml: %v - default settings are used", err)
		app.Notify(ui.T("cfg.error.t"), ui.T("cfg.error.b", err))
		cfg = config.Default()
	}
	setupLang(cfg)
	if created && os.Getenv("KORDON_DIR") == "" {
		// First start: run at login by default, it's a kill switch after all.
		_ = app.SetAutostart(true)
	}
	app.Logf("Kordon %s started", version)

	ctx, cancel := context.WithCancel(context.Background())
	r := app.NewRunner(ctx, cfg)
	r.OnConfigError = func(err error) { app.Notify(ui.T("cfg.error.t"), ui.T("cfg.error.b", err)) }

	done := make(chan struct{})
	go func() { r.Run(); close(done) }()
	stop := func() {
		cancel()
		<-done // Run un-freezes everything before returning
		app.Logf("Kordon stopped")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	if !tray {
		r.G.OnChange(ui.HeadlessRender)
		<-sig
		stop()
		return
	}
	t := &ui.Tray{G: r.G, Version: version, OnQuit: stop}
	go func() {
		<-sig
		stop()
		os.Exit(0)
	}()
	t.Run()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "kordon:", err)
	os.Exit(1)
}

func cli(args []string) error {
	cfg, _, err := config.Load(app.ConfigPath())
	if err != nil {
		return fmt.Errorf("%s: %w", app.ConfigPath(), err)
	}
	setupLang(cfg)
	switch args[0] {
	case "status":
		return status()
	case "allow":
		return allow(cfg, args[1:])
	case "config":
		fmt.Println(app.ConfigPath())
		return nil
	case "autostart":
		if len(args) < 2 || (args[1] != "on" && args[1] != "off") {
			return errors.New("usage: kordon autostart on|off")
		}
		return app.SetAutostart(args[1] == "on")
	case "headless":
		run(false)
		return nil
	case "version", "--version", "-v":
		fmt.Println("kordon", version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	}
	return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
}

const usage = `Kordon - kill switch for apps: freezes them when your network or IP changes.

  kordon                 start in the menu bar / tray
  kordon status          current IP, country and state
  kordon allow           allow the current IP
  kordon allow 1.2.3.4   allow an IP or a subnet (1.2.3.0/24)
  kordon allow NL        allow a country
  kordon config          print the config file path
  kordon autostart on|off
  kordon headless        run without a tray icon
  kordon version

https://github.com/ef1m03/kordon
`

func status() error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	geo := netwatch.NewGeo()
	r, err := netwatch.PublicIP(ctx, config.DefaultIPServices, 4*time.Second, false, 2)
	if err != nil {
		fmt.Println("IP:      -", "(offline)")
	} else {
		fmt.Println("IP:     ", ui.IPWithCountry(r.IP, geo.Country(ctx, r.IP, 4*time.Second)))
		if len(r.Others) > 0 {
			fmt.Println("         other services say:", strings.Join(r.Others, ", "))
		}
	}
	if r6, err := netwatch.PublicIP(ctx, config.DefaultIPv6Services, 4*time.Second, true, 1); err == nil {
		fmt.Println("IPv6:   ", ui.IPWithCountry(r6.IP, geo.Country(ctx, r6.IP, 4*time.Second)))
	}
	s, err := app.ReadState()
	running := err == nil && processAlive(s.PID) && time.Since(s.Updated) < 24*time.Hour
	if !running {
		fmt.Println("Kordon:  not running")
	} else {
		fmt.Println("Kordon: ", s.Line)
		if len(s.Apps) > 0 {
			fmt.Println("Apps:   ", strings.Join(s.Apps, ", "))
		}
	}
	fmt.Println("Config: ", app.ConfigPath())
	return nil
}

func allow(cfg *config.Config, args []string) error {
	var what string
	switch {
	case len(args) == 0:
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		r, err := netwatch.PublicIP(ctx, cfg.IPServices, 4*time.Second, false, cfg.Need())
		if err != nil {
			return errors.New("no internet - nothing to allow")
		}
		cfg.Allow.AddIP(r.IP)
		what = r.IP
		if r6, err := netwatch.PublicIP(ctx, cfg.IPv6Services, 4*time.Second, true, 1); err == nil && cfg.CheckIPv6 {
			cfg.Allow.AddIP(r6.IP)
			what += ", " + r6.IP
		}
	case len(args[0]) == 2 && !strings.ContainsAny(args[0], ".:/"):
		cfg.Allow.AddCountry(args[0])
		what = strings.ToUpper(args[0])
	default:
		if _, err := config.ParseIPOrNet(args[0]); err != nil {
			return err
		}
		cfg.Allow.AddIP(args[0])
		what = args[0]
	}
	if err := config.Save(app.ConfigPath(), cfg); err != nil {
		return err
	}
	fmt.Println("allowed:", what)
	return nil
}
