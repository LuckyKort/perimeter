// Perimeter freezes chosen apps the moment your network or public IP changes,
// and closes them if an allowed IP doesn't come back.
//
//	perimeter                 start in the menu bar / tray
//	perimeter status          what's going on right now
//	perimeter allow [IP|CC]   allow the current IP, a given IP/subnet or a country
//	perimeter config          print the config path
//	perimeter autostart on|off
//	perimeter headless        run without a tray icon (servers, Linux without a tray)
//	perimeter version
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

	"perimeter/internal/app"
	"perimeter/internal/config"
	"perimeter/internal/guard"
	"perimeter/internal/netwatch"
	"perimeter/internal/ui"
	"perimeter/internal/update"
	"perimeter/internal/window"
)

var version = "dev"

func main() {
	attachConsole()
	if len(os.Args) > 1 {
		if err := cli(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "perimeter:", err)
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
	// After an update the old version may still be exiting: wait a little for the lock.
	ok := false
	for i := 0; i < 10 && !ok; i++ {
		if ok, _ = lock.TryLock(); !ok {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if !ok {
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
	if created && os.Getenv("PERIMETER_DIR") == "" {
		// First start: run at login by default, it's a kill switch after all.
		_ = app.SetAutostart(true)
	}
	app.Logf("Perimeter %s started", version)
	update.Cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	r := app.NewRunner(ctx, cfg)
	r.OnConfigError = func(err error) { app.Notify(ui.T("cfg.error.t"), ui.T("cfg.error.b", err)) }
	stats := app.LoadStats()
	srv := &ui.Server{G: r.G, Stats: stats, Version: version}
	if err := srv.Start(ctx); err != nil {
		app.Logf("panel server: %v", err)
	}
	srv.StartExt(ctx)

	done := make(chan struct{})
	go func() { r.Run(); close(done) }()
	stop := func() {
		cancel()
		<-done // Run un-freezes everything before returning
		app.Logf("Perimeter stopped")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	if !tray {
		r.G.OnChange(func(snap guard.Snapshot, evs []guard.Event) {
			stats.Record(snap.Counters, time.Now())
			ui.HeadlessRender(snap, evs)
		})
		<-sig
		stop()
		return
	}
	t := &ui.Tray{G: r.G, Version: version, OnQuit: stop, UpdateCheck: cfg.UpdateCheck, Stats: stats}
	srv.UpdateAvailable, srv.UpdateInfo, srv.InstallUpdate = t.UpdateAvailable, t.UpdateInfo, t.InstallUpdate
	srv.OpenPanel = ui.OpenPanel
	t.Onboarding = srv.OnboardingPending
	go func() {
		<-sig
		stop()
		os.Exit(0)
	}()
	t.Run()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "perimeter:", err)
	os.Exit(1)
}

func cli(args []string) error {
	// Read, never create: the first start of the app itself must see a missing
	// config, that's how it knows to turn on autostart and allow the current IP.
	cfg := config.Default()
	if data, err := os.ReadFile(app.ConfigPath()); err == nil {
		if cfg, err = config.Parse(data); err != nil {
			return fmt.Errorf("%s: %w", app.ConfigPath(), err)
		}
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
			return errors.New("usage: perimeter autostart on|off")
		}
		return app.SetAutostart(args[1] == "on")
	case "headless":
		run(false)
		return nil
	case "update":
		return selfUpdate()
	case "panel":
		page := ""
		if len(args) > 1 {
			page = args[1]
		}
		return openPanel(page)
	case "popover":
		return runPopover()
	case "version", "--version", "-v":
		fmt.Println("perimeter", version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	}
	return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
}

const usage = `Perimeter - kill switch for apps: freezes them when your network or IP changes.

  perimeter                 start in the menu bar / tray
  perimeter status          current IP, country and state
  perimeter allow           allow the current IP
  perimeter allow 1.2.3.4   allow an IP or a subnet (1.2.3.0/24)
  perimeter allow NL        allow a country
  perimeter config          print the config file path
  perimeter autostart on|off
  perimeter panel           open the Perimeter panel
  perimeter update          install the latest version
  perimeter headless        run without a tray icon
  perimeter version
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
		fmt.Println("Status:  not running")
	} else {
		fmt.Println("Status: ", s.Line)
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

func selfUpdate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	rel, err := update.Latest(ctx)
	if err != nil {
		return err
	}
	if !update.Newer(version, rel.Version()) {
		fmt.Println("perimeter", version, "is the latest version")
		return nil
	}
	fmt.Println("installing", rel.Version(), "…")
	restart, err := update.Install(ctx, rel, nil)
	if err != nil {
		return err
	}
	fmt.Println("installed", rel.Version())
	// A running instance still runs the old code: stop it politely (it un-freezes
	// apps on the way out) and start the new one.
	if s, err := app.ReadState(); err == nil && processAlive(s.PID) {
		if stopProcess(s.PID) {
			return restart()
		}
		fmt.Println("restart Perimeter (Quit in the menu and open it again) to use it")
	}
	return nil
}

// runPopover is the tray window: Perimeter starts it hidden and sends it "toggle"
// lines on stdin when the icon is clicked. Not for people, so not in usage.
func runPopover() error {
	p, err := ui.ReadPanelInfo()
	if err != nil {
		return errors.New("Perimeter is not running")
	}
	return window.Popover(p.URL+"pop.html?t="+p.Token, os.Stdin)
}

// openPanel shows the running Perimeter's panel in a window (or the browser on Linux).
func openPanel(page string) error {
	p, err := ui.ReadPanelInfo()
	if err != nil {
		return errors.New("Perimeter is not running - start it first")
	}
	url := p.URL + "?t=" + p.Token
	if page != "" && !strings.ContainsAny(page, "?#&/") {
		url += "#" + page
	}
	nextPage := func() string { b, _ := os.ReadFile(ui.PanelPagePath()); return strings.TrimSpace(string(b)) }
	if err := window.Open(url, "Perimeter", ui.T("panel.close"), ui.T("panel.quit"), nextPage); err != nil {
		return app.OpenURL(url)
	}
	return nil
}
