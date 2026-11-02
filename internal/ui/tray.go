package ui

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/ef1m03/kordon/internal/app"
	"github.com/ef1m03/kordon/internal/config"
	"github.com/ef1m03/kordon/internal/guard"
	"github.com/ef1m03/kordon/internal/procs"
)

const (
	historyItems = 12
	customItems  = 10
)

// Tray is the menu bar (macOS) / notification area (Windows, Linux) app.
type Tray struct {
	G       *guard.Guard
	Version string
	OnQuit  func()

	mu       sync.Mutex
	lastIcon string
	snap     guard.Snapshot
	haveSnap bool

	mStatus, mNow, mAllowed                 *systray.MenuItem
	mAllowIP, mAllowCountry, mCheck         *systray.MenuItem
	mPause, mP15, mP60, mPForever, mUnpause *systray.MenuItem
	mApps                                   *systray.MenuItem
	appItems                                map[string]*systray.MenuItem
	customs                                 []*systray.MenuItem
	mAddApp                                 *systray.MenuItem
	mHistory                                *systray.MenuItem
	hist                                    []*systray.MenuItem
	mSettings, mLog, mAutostart             *systray.MenuItem
	mBar, mBarIP, mBarCountry, mBarIcon     *systray.MenuItem
	mAbout, mStudio, mQuit                  *systray.MenuItem
}

// Run blocks on the main thread until Quit.
func (t *Tray) Run() { systray.Run(t.onReady, func() {}) }

func (t *Tray) onReady() {
	systray.SetTooltip("Kordon")
	t.setIcon(guard.Snapshot{Config: config.Default()})

	t.mStatus = systray.AddMenuItem(T("starting"), "")
	t.mStatus.Disable()
	t.mNow = systray.AddMenuItem(T("now.unknown"), "")
	t.mNow.Disable()
	t.mAllowed = systray.AddMenuItem("", "")
	t.mAllowed.Disable()
	systray.AddSeparator()

	t.mAllowIP = systray.AddMenuItem("", "")
	t.mAllowCountry = systray.AddMenuItem("", "")
	t.mAllowIP.Hide()
	t.mAllowCountry.Hide()
	t.mCheck = systray.AddMenuItem(T("check"), "")
	t.mPause = systray.AddMenuItem(T("pause"), "")
	t.mP15 = t.mPause.AddSubMenuItem(T("pause.15m"), "")
	t.mP60 = t.mPause.AddSubMenuItem(T("pause.1h"), "")
	t.mPForever = t.mPause.AddSubMenuItem(T("pause.forever"), "")
	t.mUnpause = systray.AddMenuItem(T("unpause"), "")
	t.mUnpause.Hide()
	systray.AddSeparator()

	t.mApps = systray.AddMenuItem(T("apps"), "")
	t.appItems = map[string]*systray.MenuItem{}
	for _, p := range procs.Presets {
		t.appItems[p.ID] = t.mApps.AddSubMenuItemCheckbox(p.Name, "", false)
	}
	t.mApps.AddSeparator()
	for i := 0; i < customItems; i++ {
		it := t.mApps.AddSubMenuItem("", "")
		it.Disable()
		it.Hide()
		t.customs = append(t.customs, it)
	}
	t.mAddApp = t.mApps.AddSubMenuItem(T("apps.edit"), "")

	t.mHistory = systray.AddMenuItem(T("history"), "")
	for i := 0; i < historyItems; i++ {
		it := t.mHistory.AddSubMenuItem("", "")
		it.Disable()
		t.hist = append(t.hist, it)
	}
	t.hist[0].SetTitle(T("history.empty"))
	for _, it := range t.hist[1:] {
		it.Hide()
	}
	systray.AddSeparator()

	t.mSettings = systray.AddMenuItem(T("settings"), "")
	t.mLog = systray.AddMenuItem(T("log"), "")
	t.mAutostart = systray.AddMenuItemCheckbox(T("autostart"), "", app.AutostartEnabled())
	t.mBar = systray.AddMenuItem(T("menubar"), "")
	t.mBarIP = t.mBar.AddSubMenuItemCheckbox(T("menubar.ip"), "", false)
	t.mBarCountry = t.mBar.AddSubMenuItemCheckbox(T("menubar.country"), "", false)
	t.mBarIcon = t.mBar.AddSubMenuItemCheckbox(T("menubar.icon"), "", false)
	systray.AddSeparator()
	t.mAbout = systray.AddMenuItem(T("about", t.Version), "")
	t.mStudio = systray.AddMenuItem(T("studio"), "")
	systray.AddSeparator()
	t.mQuit = systray.AddMenuItem(T("quit"), "")

	t.G.OnChange(t.render)
	t.render(t.G.Snapshot(time.Now()), nil)
	go t.loop()
}

func (t *Tray) loop() {
	appClicks := make(chan string)
	for id, it := range t.appItems {
		go func(id string, it *systray.MenuItem) {
			for range it.ClickedCh {
				appClicks <- id
			}
		}(id, it)
	}
	setBar := func(v string) {
		t.G.Update(time.Now(), func(c *config.Config) { c.MenuBar = v })
	}
	for {
		select {
		case <-t.mAllowIP.ClickedCh:
			t.G.AllowCurrentIP(time.Now())
		case <-t.mAllowCountry.ClickedCh:
			t.G.AllowCurrentCountry(time.Now())
		case <-t.mCheck.ClickedCh:
			t.G.CheckNow(time.Now())
		case <-t.mP15.ClickedCh:
			t.G.Pause(time.Now(), 15*time.Minute)
		case <-t.mP60.ClickedCh:
			t.G.Pause(time.Now(), time.Hour)
		case <-t.mPForever.ClickedCh:
			t.G.Pause(time.Now(), guard.Forever)
		case <-t.mUnpause.ClickedCh:
			t.G.Unpause(time.Now())
		case id := <-appClicks:
			t.G.ToggleApp(time.Now(), id)
		case <-t.mAddApp.ClickedCh:
			_ = app.OpenFile(app.ConfigPath())
		case <-t.mSettings.ClickedCh:
			_ = app.OpenFile(app.ConfigPath())
		case <-t.mLog.ClickedCh:
			_ = app.OpenFile(app.LogPath())
		case <-t.mAutostart.ClickedCh:
			on := !app.AutostartEnabled()
			if err := app.SetAutostart(on); err == nil {
				setChecked(t.mAutostart, on)
			}
		case <-t.mBarIP.ClickedCh:
			setBar("ip")
		case <-t.mBarCountry.ClickedCh:
			setBar("country")
		case <-t.mBarIcon.ClickedCh:
			setBar("icon")
		case <-t.mAbout.ClickedCh:
			_ = app.OpenURL(app.RepoURL)
		case <-t.mStudio.ClickedCh:
			_ = app.OpenURL(app.StudioURL)
		case <-t.mQuit.ClickedCh:
			if t.OnQuit != nil {
				t.OnQuit()
			}
			systray.Quit()
			return
		}
	}
}

func setChecked(it *systray.MenuItem, on bool) {
	if on {
		it.Check()
	} else {
		it.Uncheck()
	}
}

func showIf(it *systray.MenuItem, on bool) {
	if on {
		it.Show()
	} else {
		it.Hide()
	}
}

// render is called by the guard after every change.
func (t *Tray) render(s guard.Snapshot, evs []guard.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.snap, t.haveSnap = s, true
	cfg := s.Config

	for _, e := range evs {
		line := EventLine(e)
		app.Logf("%s", line[strings.Index(line, " ")+2:])
		if title, body, ok := Notification(e); ok && cfg.Notifications {
			app.Notify(title, body)
		}
	}

	t.setIcon(s)
	t.mStatus.SetTitle(dotFor(ToneOf(s)) + " " + StatusLine(s))
	switch {
	case s.Offline:
		t.mNow.SetTitle(T("now.offline"))
	case s.IP == "":
		t.mNow.SetTitle(T("now.unknown"))
	default:
		t.mNow.SetTitle(T("now", IPWithCountry(s.IP, s.Country)))
	}
	t.mAllowed.SetTitle(T("allowed.summary", allowedSummary(cfg.Allow)))

	allowed := s.Allowed()
	showIf(t.mAllowIP, s.IP != "" && !slices.Contains(cfg.Allow.IPs, s.IP) && !s.Paused())
	t.mAllowIP.SetTitle(T("allow.ip", s.IP))
	showIf(t.mAllowCountry, s.Country != "" && !slices.Contains(cfg.Allow.Countries, s.Country) && !s.Paused())
	t.mAllowCountry.SetTitle(T("allow.country", Country(s.Country)))
	showIf(t.mPause, !s.Paused())
	showIf(t.mUnpause, s.Paused())

	running := procs.AppNames(s.Running)
	if len(running) > 0 {
		t.mApps.SetTitle(T("apps.running", strings.Join(running, ", ")))
	} else {
		t.mApps.SetTitle(T("apps"))
	}
	for id, it := range t.appItems {
		setChecked(it, slices.Contains(cfg.Apps, id))
	}
	for i, it := range t.customs {
		if i < len(cfg.CustomApps) {
			it.SetTitle("✓ " + T("apps.custom", cfg.CustomApps[i].Name))
			it.Show()
		} else {
			it.Hide()
		}
	}

	if len(s.History) > 0 {
		for i, it := range t.hist {
			j := len(s.History) - 1 - i // newest first
			if j >= 0 {
				it.SetTitle(EventLine(s.History[j]))
				it.Show()
			} else {
				it.Hide()
			}
		}
	}

	setChecked(t.mBarIP, cfg.MenuBar == "ip")
	setChecked(t.mBarCountry, cfg.MenuBar == "country")
	setChecked(t.mBarIcon, cfg.MenuBar == "icon")

	app.WriteState(app.State{
		Status: statusName(s.Status), Line: StatusLine(s), IP: s.IP, Country: s.Country,
		Allowed: allowed, Apps: running, Frozen: s.Frozen, Paused: s.Paused(),
	})
}

func statusName(st guard.Status) string {
	return [...]string{"starting", "ok", "frozen", "blocked"}[st]
}

func dotFor(t Tone) string {
	return map[Tone]string{Gray: "⚪", Green: "🟢", Amber: "🟠", Red: "🔴"}[t]
}

func allowedSummary(a config.Allow) string {
	var parts []string
	if len(a.IPs) > 0 {
		parts = append(parts, T("n.ips", len(a.IPs)))
	}
	if len(a.Countries) > 0 {
		parts = append(parts, T("n.countries", strings.Join(a.Countries, ", ")))
	}
	if len(parts) == 0 {
		return T("allowed.none")
	}
	return strings.Join(parts, ", ")
}

// barText is what sits next to the icon in the macOS menu bar.
func barText(s guard.Snapshot) string {
	mode := s.Config.MenuBar
	if mode == "icon" {
		return ""
	}
	if s.Offline {
		return T("now.offline")[strings.Index(T("now.offline"), ":")+2:]
	}
	if s.IP == "" {
		return "…"
	}
	var b strings.Builder
	if f := Flag(s.Country); f != "" {
		b.WriteString(f + " ")
	}
	if mode == "country" {
		b.WriteString(s.Country)
	} else {
		b.WriteString(s.IP)
	}
	if s.Status == guard.Suspect && s.Reason != guard.NetChanged && !s.Paused() {
		b.WriteString(" · " + Dur(s.GraceLeft))
	}
	return b.String()
}

func (t *Tray) setIcon(s guard.Snapshot) {
	tone := ToneOf(s)
	cc := s.Country
	if s.Config.MenuBar == "icon" || s.Offline {
		cc = ""
	}
	key := fmt.Sprint(tone, cc)
	if runtime.GOOS == "darwin" {
		if key != t.lastIcon {
			systray.SetIcon(Dot(tone))
		}
		systray.SetTitle(barText(s))
	} else if key != t.lastIcon {
		if runtime.GOOS == "windows" {
			systray.SetIcon(ICO(Badge(tone, cc, 16), Badge(tone, cc, 32), Badge(tone, cc, 48)))
		} else {
			systray.SetIcon(pngBytes(Badge(tone, cc, 64)))
		}
	}
	t.lastIcon = key

	tip := map[Tone]string{Gray: T("tip.paused"), Green: T("tip.ok"), Amber: T("tip.frozen"), Red: T("tip.blocked")}[tone]
	if s.Status == guard.Starting && !s.Paused() {
		tip = "Kordon: " + T("starting")
	}
	if s.IP != "" && s.Config.MenuBar != "icon" {
		if s.Config.MenuBar == "country" {
			tip += "\n" + Country(s.Country)
		} else {
			tip += "\n" + IPWithCountry(s.IP, s.Country)
		}
	}
	systray.SetTooltip(tip)
}

// HeadlessRender logs and notifies without a tray icon (kordon headless).
func HeadlessRender(s guard.Snapshot, evs []guard.Event) {
	for _, e := range evs {
		line := EventLine(e)
		fmt.Println(line)
		app.Logf("%s", line[strings.Index(line, " ")+2:])
		if title, body, ok := Notification(e); ok && s.Config.Notifications {
			app.Notify(title, body)
		}
	}
	app.WriteState(app.State{
		Status: statusName(s.Status), Line: StatusLine(s), IP: s.IP, Country: s.Country,
		Allowed: s.Allowed(), Apps: procs.AppNames(s.Running), Frozen: s.Frozen, Paused: s.Paused(),
	})
}
