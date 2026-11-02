package config

import (
	"fmt"
	"strconv"
	"strings"
)

// CommentLang picks the language of the comments in config.yaml ("ru" or "en").
// main sets it from the detected UI language before the first Save.
var CommentLang = "ru"

type tpl struct {
	header, apps, appsList, custom, allow, ips, countries,
	freeze, offline, grace, closeT, interval, services, agree, ipv6, services6, menubar, update, lang, notify string
}

var templates = map[string]tpl{
	"ru": {
		header: "# Kordon - настройки. Сохраните файл, и Kordon подхватит изменения сам, перезапуск не нужен.\n" +
			"# Справка: https://github.com/ef1m03/kordon#настройки",
		apps: "# Какие приложения защищать. Готовые: claude, chatgpt, codex, cursor, windsurf,\n" +
			"# copilot, gemini, perplexity, telegram, discord, slack, zoom.",
		appsList: "# Список и как их ищет Kordon - в меню «Защищаемые приложения».",
		custom: "# Свои приложения. match - имя файла (myapp, myapp.exe) или часть пути (/MyApp.app/),\n" +
			"# для скриптов - часть командной строки с префиксом args: (args:my-cli.js).\n" +
			"# Пример:\n#   - name: My App\n#     match: [\"/MyApp.app/\", \"myapp.exe\"]",
		allow: "# Где приложениям можно работать. Подходит хоть что-то одно - значит можно.",
		ips:   "# IP или подсети: \"64.137.54.87\", \"64.137.54.0/24\"",
		countries: "# Коды стран: NL, DE, US. VPN часто меняет IP внутри страны -\n" +
			"# разрешите страну, и не придётся подтверждать каждый новый адрес.",
		freeze: "# Сменилась сеть (VPN, Wi-Fi) - заморозить приложения, пока не проверим IP.\n" +
			"# Обычно это доля секунды, и если IP прежний, приложение даже не заметит.",
		offline:   "# Пропал интернет - тоже заморозить.",
		grace:     "# Сколько держать приложения замороженными, ожидая разрешённый IP. Потом - закрыть.",
		closeT:    "# Сколько ждать, пока приложение закроется само, прежде чем завершить принудительно.",
		interval:  "# Как часто сверять внешний IP, когда всё в порядке (не меньше 3s).",
		services:  "# Откуда узнавать внешний IP. Можно поставить свой сервер, отдающий IP текстом.",
		agree:     "# Сколько сервисов должны назвать один и тот же IP, чтобы ему поверить.\n# Если хоть один называет другой адрес - это похоже на утечку, приложения не разморозятся.",
		ipv6:      "# Проверять и внешний IPv6. Многие VPN заворачивают только IPv4, а по IPv6\n# программы продолжают ходить напрямую с вашим настоящим адресом.",
		services6: "# Откуда узнавать внешний IPv6.",
		menubar:   "# Что показывать рядом со значком (строка меню macOS, подсказка в Windows):\n# ip - страну и IP, country - только страну, icon - ничего (удобно на созвонах с показом экрана).",
		lang:      "# Язык: auto, ru, en.",
		notify:    "# Показывать уведомления.",
	},
	"en": {
		header: "# Kordon settings. Save the file and Kordon picks the changes up, no restart needed.\n" +
			"# Docs: https://github.com/ef1m03/kordon/blob/main/README.en.md#settings",
		apps: "# Which apps to protect. Built-in: claude, chatgpt, codex, cursor, windsurf,\n" +
			"# copilot, gemini, perplexity, telegram, discord, slack, zoom.",
		appsList: "# See the \"Protected apps\" menu for how each one is matched.",
		custom: "# Your own apps. match is a file name (myapp, myapp.exe) or part of a path (/MyApp.app/),\n" +
			"# for scripts - part of the command line prefixed with args: (args:my-cli.js).\n" +
			"# Example:\n#   - name: My App\n#     match: [\"/MyApp.app/\", \"myapp.exe\"]",
		allow: "# Where apps may run. Matching any one entry is enough.",
		ips:   "# IPs or subnets: \"64.137.54.87\", \"64.137.54.0/24\"",
		countries: "# Country codes: NL, DE, US. VPNs often rotate IPs within a country -\n" +
			"# allow the country so you don't have to confirm every new address.",
		freeze: "# Network changed (VPN, Wi-Fi) - freeze the apps until the IP is checked.\n" +
			"# Usually a fraction of a second; if the IP is the same, the app won't notice.",
		offline:   "# Internet is gone - freeze too.",
		grace:     "# How long to keep apps frozen waiting for an allowed IP before closing them.",
		closeT:    "# How long to wait for an app to quit on its own before force-closing it.",
		interval:  "# How often to re-check the public IP while everything is fine (3s minimum).",
		services:  "# Where to get the public IP from. Any server that returns the IP as plain text works.",
		agree:     "# How many services must report the same IP to trust it.\n# If any one reports a different address it looks like a leak, and apps stay frozen.",
		ipv6:      "# Check the public IPv6 too. Many VPNs only tunnel IPv4, and apps keep\n# going out over IPv6 with your real address.",
		services6: "# Where to get the public IPv6 from.",
		menubar:   "# What to show next to the icon (macOS menu bar, Windows tooltip):\n# ip - country and IP, country - country only, icon - nothing (handy when sharing your screen).",
		lang:      "# Language: auto, ru, en.",
		notify:    "# Show notifications.",
	},
}

func list(items []string) string {
	if len(items) == 0 {
		return " []"
	}
	var b strings.Builder
	for _, it := range items {
		b.WriteString("\n  - ")
		b.WriteString(strconv.Quote(it))
	}
	return b.String()
}

func indentList(items []string) string {
	if len(items) == 0 {
		return " []"
	}
	var b strings.Builder
	for _, it := range items {
		b.WriteString("\n    - ")
		b.WriteString(strconv.Quote(it))
	}
	return b.String()
}

// Render writes the config as commented YAML.
func Render(c *Config) string {
	t, ok := templates[CommentLang]
	if !ok {
		t = templates["en"]
	}
	var custom strings.Builder
	if len(c.CustomApps) == 0 {
		custom.WriteString(" []")
	}
	for _, a := range c.CustomApps {
		fmt.Fprintf(&custom, "\n  - name: %s\n    match:", strconv.Quote(a.Name))
		custom.WriteString(indentList(a.Match))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", t.header)
	fmt.Fprintf(&b, "%s\n%s\napps:%s\n\n", t.apps, t.appsList, list(c.Apps))
	fmt.Fprintf(&b, "%s\ncustom_apps:%s\n\n", t.custom, custom.String())
	fmt.Fprintf(&b, "%s\nallow:\n  %s\n  ips:%s\n  %s\n  countries:%s\n\n",
		t.allow, t.ips, indentList(c.Allow.IPs), strings.ReplaceAll(t.countries, "\n", "\n  "), indentList(c.Allow.Countries))
	fmt.Fprintf(&b, "%s\nfreeze_on_network_change: %t\n\n", t.freeze, c.FreezeOnNetworkChange)
	fmt.Fprintf(&b, "%s\nblock_when_offline: %t\n\n", t.offline, c.BlockWhenOffline)
	fmt.Fprintf(&b, "%s\ngrace: %s\n\n", t.grace, c.Grace)
	fmt.Fprintf(&b, "%s\nclose_timeout: %s\n\n", t.closeT, c.CloseTimeout)
	fmt.Fprintf(&b, "%s\ncheck_interval: %s\n\n", t.interval, c.CheckInterval)
	fmt.Fprintf(&b, "%s\nip_services:%s\n\n", t.services, list(c.IPServices))
	fmt.Fprintf(&b, "%s\nmin_agree: %d\n\n", t.agree, c.MinAgree)
	fmt.Fprintf(&b, "%s\ncheck_ipv6: %t\n\n", t.ipv6, c.CheckIPv6)
	fmt.Fprintf(&b, "%s\nipv6_services:%s\n\n", t.services6, list(c.IPv6Services))
	fmt.Fprintf(&b, "%s\nmenubar: %s\n\n", t.menubar, c.MenuBar)
	fmt.Fprintf(&b, "%s\nupdate_check: %t\n\n", t.update, c.UpdateCheck)
	fmt.Fprintf(&b, "%s\nlanguage: %s\n\n", t.lang, c.Language)
	fmt.Fprintf(&b, "%s\nnotifications: %t\n", t.notify, c.Notifications)
	return b.String()
}
