// Package ui is the menu-bar / tray app and everything the user reads.
package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/ef1m03/kordon/internal/guard"
)

// Lang is "ru" or "en", set once at start.
var Lang = "ru"

var msgs = map[string]map[string]string{
	"ru": {
		"starting":        "Проверяю IP…",
		"ok":              "Защита включена",
		"netchanged":      "Сеть изменилась, проверяю IP…",
		"frozen.ip":       "IP не разрешён: заморожено, закрою через %s",
		"frozen.offline":  "Нет интернета: заморожено, закрою через %s",
		"blocked.ip":      "Заблокировано: IP не разрешён",
		"blocked.offline": "Заблокировано: нет интернета",
		"paused.until":    "Пауза до %s",
		"paused.forever":  "Пауза до перезапуска Kordon",
		"now":             "Сейчас: %s",
		"now.offline":     "Сейчас: нет интернета",
		"now.unknown":     "Сейчас: проверяю…",
		"allowed.summary": "Разрешено: %s",
		"allowed.none":    "ничего",
		"n.ips":           "%d IP",
		"n.countries":     "страны %s",
		"allow.ip":        "Разрешить этот IP (%s)",
		"allow.country":   "Разрешить всю страну: %s",
		"check":           "Проверить IP сейчас",
		"pause":           "Пауза",
		"pause.15m":       "На 15 минут",
		"pause.1h":        "На 1 час",
		"pause.forever":   "До перезапуска",
		"unpause":         "Снять паузу",
		"apps":            "Защищаемые приложения",
		"apps.running":    "Защищаемые приложения (запущено: %s)",
		"apps.custom":     "%s (свои, в настройках)",
		"apps.edit":       "Добавить своё приложение…",
		"history":         "История",
		"history.empty":   "Пока ничего не случилось",
		"settings":        "Настройки…",
		"log":             "Открыть журнал",
		"autostart":       "Запускать при входе в систему",
		"menubar":         "Показывать рядом со значком",
		"menubar.ip":      "Страну и IP",
		"menubar.country": "Только страну",
		"menubar.icon":    "Ничего (для созвонов)",
		"about":           "Kordon %s на GitHub",
		"studio":          "Сделано в Efimov Dev · efimovdev.ru",
		"quit":            "Выйти (защита выключится)",
		"tip.ok":          "Kordon: защита включена",
		"tip.frozen":      "Kordon: приложения заморожены",
		"tip.blocked":     "Kordon: заблокировано",
		"tip.paused":      "Kordon: пауза",
		"sec":             "%d с",
		"min":             "%d мин",

		"ev.autoallowed":    "Разрешён текущий IP %s",
		"ev.autoallowed.t":  "Kordon запущен",
		"ev.autoallowed.b":  "Разрешён текущий IP: %s. Если VPN меняет адреса, разрешите всю страну в меню.",
		"ev.netchanged":     "Сеть изменилась",
		"ev.ipchanged":      "Новый IP: %s",
		"ev.offline":        "Пропал интернет",
		"ev.online":         "Интернет вернулся: %s",
		"ev.frozen":         "Заморожено (%s): %s",
		"ev.frozen.t":       "%s: заморожено",
		"ev.frozen.ip.b":    "IP %s не разрешён. Если разрешённый не вернётся за %s, закрою.",
		"ev.frozen.off.b":   "Пропал интернет. Если не вернётся за %s, закрою.",
		"ev.resumed":        "Разморожено: %s",
		"ev.resumed.t":      "%s: снова работает",
		"ev.resumed.b":      "IP %s разрешён.",
		"ev.blocked":        "Закрыто: %s",
		"ev.blocked.t":      "%s: закрыто",
		"ev.blocked.ip.b":   "IP %s не разрешён. Открыть можно, когда вернётся разрешённый IP.",
		"ev.blocked.off.b":  "Нет интернета. Открыть можно, когда он вернётся с разрешённым IP.",
		"ev.killednew":      "Не дал запуститься: %s",
		"ev.killednew.t":    "%s не запустится",
		"ev.killednew.b":    "Сейчас IP %s не разрешён.",
		"ev.unblocked":      "IP снова разрешён: %s",
		"ev.unblocked.t":    "Можно работать",
		"ev.unblocked.b":    "IP %s разрешён, приложения можно открывать.",
		"ev.paused":         "Пауза: %s",
		"ev.unpaused":       "Пауза закончилась",
		"ev.allowedip":      "Разрешён IP %s",
		"ev.allowedcountry": "Разрешена страна: %s",
		"reason.ip":         "IP не разрешён",
		"reason.offline":    "нет интернета",
		"reason.net":        "сеть изменилась",
		"cfg.error.t":       "Ошибка в настройках",
		"cfg.error.b":       "%v. Работаю со старыми настройками.",
		"already.t":         "Kordon уже запущен",
		"already.b":         "Значок в строке меню (на Windows - у часов, может прятаться под стрелкой ^).",
	},
	"en": {
		"starting":        "Checking IP…",
		"ok":              "Protection is on",
		"netchanged":      "Network changed, checking IP…",
		"frozen.ip":       "IP not allowed: frozen, closing in %s",
		"frozen.offline":  "No internet: frozen, closing in %s",
		"blocked.ip":      "Blocked: IP not allowed",
		"blocked.offline": "Blocked: no internet",
		"paused.until":    "Paused until %s",
		"paused.forever":  "Paused until Kordon restarts",
		"now":             "Now: %s",
		"now.offline":     "Now: no internet",
		"now.unknown":     "Now: checking…",
		"allowed.summary": "Allowed: %s",
		"allowed.none":    "nothing",
		"n.ips":           "%d IP",
		"n.countries":     "countries %s",
		"allow.ip":        "Allow this IP (%s)",
		"allow.country":   "Allow the whole country: %s",
		"check":           "Check IP now",
		"pause":           "Pause",
		"pause.15m":       "For 15 minutes",
		"pause.1h":        "For 1 hour",
		"pause.forever":   "Until restart",
		"unpause":         "Resume protection",
		"apps":            "Protected apps",
		"apps.running":    "Protected apps (running: %s)",
		"apps.custom":     "%s (custom, in settings)",
		"apps.edit":       "Add your own app…",
		"history":         "History",
		"history.empty":   "Nothing has happened yet",
		"settings":        "Settings…",
		"log":             "Open log",
		"autostart":       "Start at login",
		"menubar":         "Show next to the icon",
		"menubar.ip":      "Country and IP",
		"menubar.country": "Country only",
		"menubar.icon":    "Nothing (for screen sharing)",
		"about":           "Kordon %s on GitHub",
		"studio":          "Made by Efimov Dev · efimovdev.ru",
		"quit":            "Quit (turns protection off)",
		"tip.ok":          "Kordon: protection is on",
		"tip.frozen":      "Kordon: apps frozen",
		"tip.blocked":     "Kordon: blocked",
		"tip.paused":      "Kordon: paused",
		"sec":             "%ds",
		"min":             "%d min",

		"ev.autoallowed":    "Allowed the current IP %s",
		"ev.autoallowed.t":  "Kordon is running",
		"ev.autoallowed.b":  "Allowed the current IP: %s. If your VPN rotates addresses, allow the whole country in the menu.",
		"ev.netchanged":     "Network changed",
		"ev.ipchanged":      "New IP: %s",
		"ev.offline":        "Internet is gone",
		"ev.online":         "Internet is back: %s",
		"ev.frozen":         "Frozen (%s): %s",
		"ev.frozen.t":       "%s: frozen",
		"ev.frozen.ip.b":    "IP %s is not allowed. If an allowed one isn't back within %s, I'll close it.",
		"ev.frozen.off.b":   "No internet. If it isn't back within %s, I'll close it.",
		"ev.resumed":        "Resumed: %s",
		"ev.resumed.t":      "%s: running again",
		"ev.resumed.b":      "IP %s is allowed.",
		"ev.blocked":        "Closed: %s",
		"ev.blocked.t":      "%s: closed",
		"ev.blocked.ip.b":   "IP %s is not allowed. You can reopen it once an allowed IP is back.",
		"ev.blocked.off.b":  "No internet. You can reopen it once it's back with an allowed IP.",
		"ev.killednew":      "Didn't let start: %s",
		"ev.killednew.t":    "%s won't start",
		"ev.killednew.b":    "IP %s is not allowed right now.",
		"ev.unblocked":      "IP allowed again: %s",
		"ev.unblocked.t":    "Good to go",
		"ev.unblocked.b":    "IP %s is allowed, you can open your apps.",
		"ev.paused":         "Paused: %s",
		"ev.unpaused":       "Pause is over",
		"ev.allowedip":      "Allowed IP %s",
		"ev.allowedcountry": "Allowed country: %s",
		"reason.ip":         "IP not allowed",
		"reason.offline":    "no internet",
		"reason.net":        "network changed",
		"cfg.error.t":       "Settings error",
		"cfg.error.b":       "%v. Keeping the previous settings.",
		"already.t":         "Kordon is already running",
		"already.b":         "Look for the icon in the menu bar (on Windows - by the clock, maybe under the ^ arrow).",
	},
}

// T formats a message in the current language.
func T(key string, args ...any) string {
	m := msgs[Lang][key]
	if m == "" {
		m = msgs["en"][key]
	}
	if len(args) == 0 {
		return m
	}
	return fmt.Sprintf(m, args...)
}

// Flags render on macOS and most Linux desktops, but Windows shows them as two letters.
var useFlags = runtime.GOOS != "windows"

// Flag turns "NL" into 🇳🇱.
func Flag(cc string) string {
	if len(cc) != 2 || !useFlags {
		return ""
	}
	cc = strings.ToUpper(cc)
	return string(rune(0x1F1E6+rune(cc[0]-'A'))) + string(rune(0x1F1E6+rune(cc[1]-'A')))
}

// CountryName gives "Нидерланды" / "Netherlands".
func CountryName(cc string) string {
	r, err := language.ParseRegion(cc)
	if err != nil {
		return cc
	}
	n := display.Regions(language.Make(Lang)).Name(r)
	if n == "" {
		return cc
	}
	return n
}

// Country is "🇳🇱 Нидерланды" on macOS and "Нидерланды (NL)" on Windows.
func Country(cc string) string {
	if cc == "" {
		return ""
	}
	if f := Flag(cc); f != "" {
		return f + " " + CountryName(cc)
	}
	return CountryName(cc) + " (" + cc + ")"
}

// IPWithCountry is "64.137.54.87, 🇳🇱 Нидерланды".
func IPWithCountry(ip, cc string) string {
	if cc == "" {
		return ip
	}
	return ip + ", " + Country(cc)
}

func Dur(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return T("min", int(d/time.Minute))
	}
	return T("sec", int((d+time.Second-1)/time.Second))
}

func appsText(apps []string) string {
	if len(apps) == 0 {
		return "-"
	}
	return strings.Join(apps, ", ")
}

// StatusLine is the first line of the menu.
func StatusLine(s guard.Snapshot) string {
	if s.Paused() {
		if s.PausedUntil.Sub(time.Now()) > 24*time.Hour {
			return T("paused.forever")
		}
		return T("paused.until", s.PausedUntil.Format("15:04"))
	}
	switch s.Status {
	case guard.Starting:
		return T("starting")
	case guard.OK:
		return T("ok")
	case guard.Suspect:
		switch s.Reason {
		case guard.NetChanged:
			return T("netchanged")
		case guard.Offline:
			return T("frozen.offline", Dur(s.GraceLeft))
		}
		return T("frozen.ip", Dur(s.GraceLeft))
	case guard.Blocked:
		if s.Reason == guard.Offline {
			return T("blocked.offline")
		}
		return T("blocked.ip")
	}
	return ""
}

func reasonText(r guard.Reason) string {
	switch r {
	case guard.Offline:
		return T("reason.offline")
	case guard.NetChanged:
		return T("reason.net")
	}
	return T("reason.ip")
}

// EventLine is one line of the history.
func EventLine(e guard.Event) string {
	ipc := IPWithCountry(e.IP, e.Country)
	var s string
	switch e.Kind {
	case guard.EvAutoAllowed:
		s = T("ev.autoallowed", ipc)
	case guard.EvNetChanged:
		s = T("ev.netchanged")
	case guard.EvIPChanged:
		s = T("ev.ipchanged", ipc)
	case guard.EvOffline:
		s = T("ev.offline")
	case guard.EvOnline:
		s = T("ev.online", ipc)
	case guard.EvFrozen:
		s = T("ev.frozen", reasonText(e.Reason), appsText(e.Apps))
	case guard.EvResumed:
		s = T("ev.resumed", appsText(e.Apps))
	case guard.EvBlocked:
		s = T("ev.blocked", appsText(e.Apps))
	case guard.EvKilledNew:
		s = T("ev.killednew", appsText(e.Apps))
	case guard.EvUnblocked:
		s = T("ev.unblocked", ipc)
	case guard.EvPaused:
		if e.Dur == guard.Forever {
			s = T("ev.paused", T("pause.forever"))
		} else {
			s = T("ev.paused", Dur(e.Dur))
		}
	case guard.EvUnpaused:
		s = T("ev.unpaused")
	case guard.EvAllowedIP:
		s = T("ev.allowedip", ipc)
	case guard.EvAllowedCountry:
		s = T("ev.allowedcountry", Country(e.Country))
	}
	return e.At.Local().Format("15:04:05") + "  " + s
}

// Notification returns the title and body for events worth a notification.
func Notification(e guard.Event) (title, body string, ok bool) {
	if !e.Notify {
		return "", "", false
	}
	ipc := IPWithCountry(e.IP, e.Country)
	apps := appsText(e.Apps)
	switch e.Kind {
	case guard.EvAutoAllowed:
		return T("ev.autoallowed.t"), T("ev.autoallowed.b", ipc), true
	case guard.EvFrozen:
		if e.Reason == guard.Offline {
			return T("ev.frozen.t", apps), T("ev.frozen.off.b", Dur(e.Dur)), true
		}
		return T("ev.frozen.t", apps), T("ev.frozen.ip.b", ipc, Dur(e.Dur)), true
	case guard.EvResumed:
		return T("ev.resumed.t", apps), T("ev.resumed.b", ipc), true
	case guard.EvBlocked:
		if e.Reason == guard.Offline {
			return T("ev.blocked.t", apps), T("ev.blocked.off.b"), true
		}
		return T("ev.blocked.t", apps), T("ev.blocked.ip.b", ipc), true
	case guard.EvKilledNew:
		return T("ev.killednew.t", apps), T("ev.killednew.b", ipc), true
	case guard.EvUnblocked:
		return T("ev.unblocked.t"), T("ev.unblocked.b", ipc), true
	}
	return "", "", false
}
