"use strict";

// Perimeter panel. Talks to the local Perimeter over /api; every value from it is
// escaped before it goes into the page (custom app names come from config.yaml).

const L = {
  ru: {
    overview: "Обзор", apps: "Приложения", allow: "Разрешено", networks: "Сети", stats: "Статистика",
    history: "История", settings: "Настройки", about: "О программе", leaks: "Проверка",
    lead_leaks: "Глубокая проверка: каждый сервис по отдельности, IPv6, DNS и программы, которые не смотрят на системный прокси. Занимает несколько секунд.",
    lk_run: "Проверить на утечки", lk_again: "Проверить ещё раз", lk_running: "Проверяю… несколько секунд", lk_at: "Проверено в",
    lk_ok: "Всё чисто", lk_warn: "Есть на что посмотреть", lk_bad: "Найдена утечка",
    lk_v4: "Внешний IP", lk_v4_ok: "Все сервисы видят один адрес, и он разрешён.", lk_v4_split: "Сервисы видят разные адреса — часть трафика идёт мимо VPN.",
    lk_v4_notallowed: "Адрес не разрешён: защищаемые программы будут на паузе.", lk_v4_none: "Ни один сервис не ответил — нет интернета?",
    lk_direct: "Программы без прокси", lk_direct_none: "Системный прокси не задан — все программы идут одним путём.",
    lk_direct_ok: "Через прокси и напрямую — один адрес: VPN заворачивает весь трафик.",
    lk_direct_bad: "Программы, которые не смотрят на системный прокси (терминал, Claude Code, git, многие CLI), выходят с другого адреса. Включите в VPN-клиенте режим TUN («весь трафик»).",
    lk_direct_warn: "Без прокси трафик идёт с другого адреса, но он тоже разрешён.",
    lk_v6: "IPv6", lk_v6_off: "Проверка IPv6 выключена в настройках.", lk_v6_none: "IPv6 нет — утечки через него быть не может.",
    lk_v6_ok: "IPv6 идёт через разрешённый адрес.", lk_v6_bad: "По IPv6 программы выходят мимо VPN. Выключите IPv6 в системе или включите его поддержку в VPN.",
    lk_dns: "DNS", lk_dns_none: "Не удалось определить DNS-сервер.", lk_dns_ok: "DNS-сервер в той же стране, что и IP.",
    lk_dns_warn: "Запросы к сайтам идут через DNS в другой стране — скорее всего, у провайдера. Включите в VPN «DNS через туннель».",
    lk_web: "Браузер", lk_web_d: "Утечку через WebRTC проверяет расширение Perimeter для браузера.",
    lk_services: "Ответы сервисов", lk_noans: "нет ответа", lk_svc: "Сервис", lk_addr: "Адрес", lk_cc: "Страна",
    lead_overview: "Что сейчас с сетью и защищёнными приложениями",
    lead_apps: "Perimeter замораживает выбранные программы, когда сеть или IP не те. У каждой могут быть свои правила.",
    lead_allow: "Где приложениям можно работать. Подходит хоть что-то одно — значит можно.",
    lead_networks: "В доверенных сетях (дом, офис) приложения работают без проверки IP. Сеть узнаётся по её роутеру.",
    lead_stats: "Сколько раз Perimeter вмешивался за последние 14 дней",
    lead_history: "Последние события",
    lead_settings: "Изменения сохраняются сразу",
    ip: "Внешний IP", ipv6: "IPv6", dns: "DNS", network: "Сеть", none: "нет", offline: "нет интернета", checking: "проверяю…",
    ipv6_none: "IPv6 нет или он выключен", dns_ok: "та же страна, что у IP", dns_leak: "идёт мимо VPN",
    h_ok: "Защита включена", h_frozen: "Приложения заморожены", h_blocked: "Приложения закрыты", h_starting: "Проверяю IP…", close_in: "закрою через",
    no_apps: "Защищаемые приложения не запущены", dns_unknown: "не удалось определить",
    trusted_net: "Доверенная сеть", usual_net: "Обычная сеть", unknown_net: "Роутер не виден",
    trust: "Доверять этой сети", allow_ip: "Разрешить этот IP", allow_country: "Разрешить страну", check: "Проверить",
    pause: "Пауза", pause15: "15 минут", pause60: "1 час", pauseF: "До перезапуска", unpause: "Снять паузу",
    others: "Другие сервисы видят", running: "запущено", not_running: "не запущено", frozen: "заморожено", blocked: "закрыто", ok: "работает",
    own_rules: "Свои правила", shared_rules: "Общие правила", rule_hint: "Где можно работать именно этому приложению. Пусто — значит общие правила со страницы «Разрешено».",
    remove_rule: "Вернуть общие правила", browsers: "Браузеры", browsers_hint: "Для ИИ в браузере. Браузер замирает целиком, со всеми вкладками, поэтому удобнее открывать ИИ в отдельном браузере и защищать только его.", browsers_safari: " Safari защитить нельзя: его вкладки работают в общих системных процессах macOS.", custom: "Свои приложения", add_custom: "Добавить", custom_name: "Название", custom_match: "Имя файла или часть пути, через запятую",
    remove: "Удалить", ips: "IP и подсети", countries: "Страны", add: "Добавить", ip_ph: "64.137.54.87 или 64.137.54.0/24",
    sec_c_hint: "Приложения работают, если IP из одной из этих стран.", sec_ip_hint: "Отдельные адреса или подсети — например, адрес вашего VPN.",
    country_ph: "Начните вводить страну", no_ips: "Нет разрешённых IP", no_countries: "Нет разрешённых стран",
    current_net: "Текущая сеть", router: "Роутер", net_name: "Название, например «Дом»", no_trusted: "Доверенных сетей пока нет",
    here: "вы здесь", rename: "Переименовать",
    s_saves: "VPN вернулся вовремя", s_freezes: "Заморозок", s_blocks: "Приложения закрыты", s_time: "Время в заморозке",
    legend_saves: "вернулся вовремя", legend_blocks: "закрыто", legend_other: "заморожено", trusted_list: "Доверенные сети", session: "С запуска Perimeter",
    g_protect: "Защита", g_iface: "Интерфейс", g_system: "Система", g_files: "Файлы и помощь",
    o_freeze: "Замораживать при смене сети", o_freeze_d: "Пока Perimeter проверяет IP, обычно доля секунды",
    o_offline: "Замораживать без интернета", o_offline_d: "Когда интернет пропал целиком",
    o_ipv6: "Проверять IPv6", o_ipv6_d: "Ловит VPN, которые заворачивают только IPv4",
    o_dns: "Утечка DNS", o_dns_d: "Запросы к сайтам идут мимо VPN", dns_warn: "Предупреждать", dns_block: "Замораживать", dns_off: "Не проверять",
    o_closeapps: "Закрывать приложения", o_closeapps_d: "Если разрешённый IP не вернулся вовремя",
    o_closeapps_off: "Выключено: ждут замороженными, сколько нужно",
    o_grace: "Ждать разрешённый IP", o_grace_d: "Потом приложения закрываются", o_close: "Ждать закрытия", o_close_d: "Потом закрыть принудительно",
    o_interval: "Сверять IP раз в", o_interval_d: "Когда всё в порядке", o_agree: "Сервисов должны совпасть", o_agree_d: "Чтобы поверить IP",
    o_menubar: "Рядом со значком", mb_ip: "Страна и IP", mb_country: "Только страна", mb_icon: "Ничего",
    o_lang: "Язык", lang_auto: "Как в системе", o_notify: "Уведомления",
    o_autostart: "Запускать при входе в систему", o_update: "Проверять обновления", o_update_d: "Несколько раз в день, ставятся только по клику", on: "Да", off: "Нет",
    open_config: "Открыть config.yaml", open_log: "Открыть журнал", report: "Сообщить о проблеме", sec: "с",
    update_avail: "Доступна версия", update_btn: "Обновить", saved: "Сохранено", whats_new: "Что нового в", all_changes: "Все версии", about_text: "Kill switch для приложений: замораживает выбранные программы, когда отваливается VPN или меняется IP. Открытый код, ничего не отправляет без вашего согласия.",
    based_on: "Основано на Kordon © Efimov Dev", github: "Perimeter на GitHub", license: "Лицензия GPL-3.0",
    history_empty: "Пока ничего не случилось", g_log: "Журнал",
    es_hist: "Здесь появятся заморозки, возвраты VPN и закрытые приложения.",
    es_net: "Подключитесь к Wi-Fi или по кабелю — сеть появится здесь.",
    es_trusted: "Добавьте текущую сеть, и в ней приложения будут работать без проверки IP.", paused_until: "Пауза до", paused_forever: "Пауза до перезапуска",
  },
  en: {
    overview: "Overview", apps: "Apps", allow: "Allowed", networks: "Networks", stats: "Statistics",
    history: "History", settings: "Settings", about: "About", leaks: "Leak check",
    lead_leaks: "A deep check: every service on its own, IPv6, DNS, and programs that ignore the system proxy. Takes a few seconds.",
    lk_run: "Check for leaks", lk_again: "Check again", lk_running: "Checking… a few seconds", lk_at: "Checked at",
    lk_ok: "All clean", lk_warn: "Worth a look", lk_bad: "A leak found",
    lk_v4: "Public IP", lk_v4_ok: "All services see one address, and it is allowed.", lk_v4_split: "Services see different addresses — some traffic bypasses the VPN.",
    lk_v4_notallowed: "The address isn't allowed: protected apps will be paused.", lk_v4_none: "No service answered — no internet?",
    lk_direct: "Programs without the proxy", lk_direct_none: "No system proxy — all programs take the same way.",
    lk_direct_ok: "With and without the proxy — the same address: the VPN carries all traffic.",
    lk_direct_bad: "Programs that ignore the system proxy (terminal, Claude Code, git, most CLIs) go out from another address. Turn on TUN (\"all traffic\") mode in the VPN client.",
    lk_direct_warn: "Without the proxy traffic leaves from another address, but it is allowed too.",
    lk_v6: "IPv6", lk_v6_off: "IPv6 checking is off in the settings.", lk_v6_none: "No IPv6 — nothing can leak through it.",
    lk_v6_ok: "IPv6 goes through an allowed address.", lk_v6_bad: "Over IPv6 programs bypass the VPN. Turn IPv6 off in the system or enable it in the VPN.",
    lk_dns: "DNS", lk_dns_none: "Couldn't tell the DNS server.", lk_dns_ok: "The DNS server is in the same country as the IP.",
    lk_dns_warn: "Site lookups go through DNS in another country — probably your ISP's. Turn on \"DNS through the tunnel\" in the VPN.",
    lk_web: "Browser", lk_web_d: "The Perimeter browser extension checks for WebRTC leaks.",
    lk_services: "Service answers", lk_noans: "no answer", lk_svc: "Service", lk_addr: "Address", lk_cc: "Country",
    lead_overview: "Your network and protected apps right now",
    lead_apps: "Perimeter freezes the chosen apps when the network or IP isn't right. Each one can have its own rules.",
    lead_allow: "Where apps may run. Matching any one entry is enough.",
    lead_networks: "On trusted networks (home, office) apps run without the IP check. A network is recognised by its router.",
    lead_stats: "How often Perimeter stepped in over the last 14 days",
    lead_history: "Recent events",
    lead_settings: "Changes are saved right away",
    ip: "Public IP", ipv6: "IPv6", dns: "DNS", network: "Network", none: "none", offline: "no internet", checking: "checking…",
    ipv6_none: "No IPv6, or it's off", dns_ok: "same country as the IP", dns_leak: "bypasses the VPN",
    h_ok: "Protection is on", h_frozen: "Apps are frozen", h_blocked: "Apps are closed", h_starting: "Checking IP…", close_in: "closing in",
    no_apps: "No protected apps are running", dns_unknown: "couldn't tell",
    trusted_net: "Trusted network", usual_net: "Regular network", unknown_net: "Router not visible",
    trust: "Trust this network", allow_ip: "Allow this IP", allow_country: "Allow country", check: "Check",
    pause: "Pause", pause15: "15 minutes", pause60: "1 hour", pauseF: "Until restart", unpause: "Resume",
    others: "Other services see", running: "running", not_running: "not running", frozen: "frozen", blocked: "closed", ok: "running",
    own_rules: "Own rules", shared_rules: "Shared rules", rule_hint: "Where this app may run. Empty means the shared rules from the Allowed page.",
    remove_rule: "Use shared rules", browsers: "Browsers", browsers_hint: "For AI on the web. A browser freezes whole, with every tab, so it is easier to use a separate browser for AI and protect only that one.", browsers_safari: " Safari can't be protected: its tabs run in shared macOS system processes.", custom: "Your apps", add_custom: "Add", custom_name: "Name", custom_match: "File name or part of the path, comma-separated",
    remove: "Remove", ips: "IPs and subnets", countries: "Countries", add: "Add", ip_ph: "64.137.54.87 or 64.137.54.0/24",
    sec_c_hint: "Apps run when the IP is in one of these countries.", sec_ip_hint: "Single addresses or subnets — your VPN's address, for example.",
    country_ph: "Start typing a country", no_ips: "No allowed IPs", no_countries: "No allowed countries",
    current_net: "Current network", router: "Router", net_name: "Name, e.g. \"Home\"", no_trusted: "No trusted networks yet",
    here: "you're here", rename: "Rename",
    s_saves: "VPN back in time", s_freezes: "Freezes", s_blocks: "Apps closed", s_time: "Time frozen",
    legend_saves: "back in time", legend_blocks: "closed", legend_other: "frozen", trusted_list: "Trusted networks", session: "Since Perimeter started",
    g_protect: "Protection", g_iface: "Interface", g_system: "System", g_files: "Files and help",
    o_freeze: "Freeze on network change", o_freeze_d: "While Perimeter checks the IP, usually a split second",
    o_offline: "Freeze when offline", o_offline_d: "When the internet is gone entirely",
    o_ipv6: "Check IPv6", o_ipv6_d: "Catches VPNs that tunnel IPv4 only",
    o_dns: "DNS leaks", o_dns_d: "Lookups bypass the VPN", dns_warn: "Warn", dns_block: "Freeze", dns_off: "Don't check",
    o_closeapps: "Close apps", o_closeapps_d: "If an allowed IP isn't back in time",
    o_closeapps_off: "Off: they wait frozen as long as it takes",
    o_grace: "Wait for an allowed IP", o_grace_d: "Then the apps are closed", o_close: "Wait for apps to quit", o_close_d: "Then force them",
    o_interval: "Check the IP every", o_interval_d: "While all is well", o_agree: "Services must agree", o_agree_d: "To trust an IP",
    o_menubar: "Next to the icon", mb_ip: "Country and IP", mb_country: "Country only", mb_icon: "Nothing",
    o_lang: "Language", lang_auto: "System", o_notify: "Notifications",
    o_autostart: "Start at login", o_update: "Check for updates", o_update_d: "A few times a day, installed only when you click", on: "Yes", off: "No",
    open_config: "Open config.yaml", open_log: "Open log", report: "Report a problem", sec: "s",
    update_avail: "Version available:", update_btn: "Update", saved: "Saved", whats_new: "What's new in", all_changes: "All versions", about_text: "A kill switch for apps: freezes the apps you choose when the VPN drops or your IP changes. Open source, sends nothing without your consent.",
    based_on: "Based on Kordon © Efimov Dev", github: "Perimeter on GitHub", license: "GPL-3.0 license",
    history_empty: "Nothing has happened yet", g_log: "Log",
    es_hist: "Freezes, VPN comebacks and closed apps will show up here.",
    es_net: "Connect to Wi-Fi or a cable and the network shows up here.",
    es_trusted: "Add the current network, and apps will run there without the IP check.", paused_until: "Paused until", paused_forever: "Paused until restart",
  },
};

const ICONS = {
  overview: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',
  apps: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
  allow: '<path d="M12 3l8 3v6c0 4.5-3.4 8-8 9-4.6-1-8-4.5-8-9V6z"/><path d="M8.5 12l2.5 2.5 4.5-5"/>',
  networks: '<path d="M2 9a15 15 0 0120 0"/><path d="M5.5 12.5a10 10 0 0113 0"/><path d="M9 16a5 5 0 016 0"/><circle cx="12" cy="19.5" r="1"/>',
  stats: '<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>',
  history: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.7 1.7 0 00-1.8-.3 1.7 1.7 0 00-1 1.5V21a2 2 0 11-4 0v-.1a1.7 1.7 0 00-1.1-1.5 1.7 1.7 0 00-1.8.3l-.1.1a2 2 0 11-2.8-2.8l.1-.1a1.7 1.7 0 00.3-1.8 1.7 1.7 0 00-1.5-1H3a2 2 0 110-4h.1a1.7 1.7 0 001.5-1.1 1.7 1.7 0 00-.3-1.8l-.1-.1a2 2 0 112.8-2.8l.1.1a1.7 1.7 0 001.8.3H9a1.7 1.7 0 001-1.5V3a2 2 0 114 0v.1a1.7 1.7 0 001 1.5 1.7 1.7 0 001.8-.3l.1-.1a2 2 0 112.8 2.8l-.1.1a1.7 1.7 0 00-.3 1.8V9a1.7 1.7 0 001.5 1H21a2 2 0 110 4h-.1a1.7 1.7 0 00-1.5 1z"/>',
  about: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7.5v.5"/>',
  leaks: '<circle cx="11" cy="11" r="6.5"/><path d="M16 16l4.5 4.5"/><path d="M8.5 11l1.8 1.8 3.2-3.6"/>',
  check: '<path d="M5 12.5l4.5 4.5L19 7.5"/>',
  snow: '<path d="M12 2v20M4.9 7l14.2 10M4.9 17L19.1 7"/>',
  stop: '<path d="M6 6l12 12M18 6L6 18"/>',
  pause: '<path d="M9 5v14M15 5v14"/>',
  chev: '<path d="M7 10l5 5 5-5"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.7 3.8 5.7 3.8 9s-1.3 6.3-3.8 9c-2.5-2.7-3.8-5.7-3.8-9S9.5 5.7 12 3z"/>',
  recheck: '<path d="M20 12a8 8 0 1 1-2.3-5.6M20 4v5h-5"/>',
  pin: '<rect x="4" y="4" width="16" height="16" rx="4"/><path d="M8 12h8M12 8v8"/>',
};
const svg = (name) => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${ICONS[name]}</svg>`;
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

const PAGES = ["overview", "apps", "allow", "networks", "leaks", "stats", "history", "settings", "about"];
// Sidebar: two labelled groups, settings and about pinned to the bottom.
const NAV = [["g_protect", ["overview", "apps", "allow", "networks", "leaks"]], ["g_log", ["stats", "history"]], ["", ["settings", "about"]]];
// Sidebar tile colours (Apple system palette), one per page.
const NAV_TINT = { overview: "#34C759", apps: "#007AFF", allow: "#30B0C7", networks: "#5856D6", leaks: "#FF9500", stats: "#AF52DE", history: "#FF2D55", settings: "#8E8E93", about: "#8E8E93" };
let S = null; // last state
let page = (location.hash || "#overview").slice(1);
let countries = [];
let openRule = null; // app id whose rule editor is open
let lastJSON = "";
const t = (k) => (L[S?.lang] || L.ru)[k] || L.en[k] || k;

async function api(op, data = {}) {
  const r = await fetch("/api/do", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ op, ...data }) });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) {
    toast(j.error || r.statusText);
    return false;
  }
  await refresh(true);
  return true;
}

let toastTimer;
function toast(msg) {
  const el = document.getElementById("toast");
  el.textContent = msg;
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (el.hidden = true), 3500);
}

function country(c) {
  if (!c) return "";
  return S.goos === "windows" ? `${esc(c.name)} (${esc(c.code)})` : `<span class="flag">${esc(c.flag)}</span>${esc(c.name)}`;
}

function fmtDur(sec) {
  sec = Math.round(sec || 0);
  const en = S.lang === "en";
  if (sec < 60) return `${sec} ${t("sec")}`;
  const m = Math.floor(sec / 60), h = Math.floor(m / 60);
  if (!h) return `${m} ${en ? "min" : "мин"}`;
  return `${h} ${en ? "h" : "ч"} ${m % 60} ${en ? "min" : "мин"}`;
}

// ---------- pages ----------

function heroActions() {
  const a = [];
  if (S.paused) return [`<button class="btn primary" data-op="unpause">${t("unpause")}</button>`];
  // Way out of a wrong IP first; then "check" shrinks to an icon (the IP is checked
  // all the time anyway) and the country to its flag, so the row stays one line.
  const fix = S.ip && !S.allowedNow && (S.reason === "ip" || S.reason === "v6");
  if (fix) {
    a.push(`<button class="btn primary" data-op="allow_current_ip">${t("allow_ip")}</button>`);
    if (S.country) {
      const c = S.goos === "windows" ? esc(S.country.code) : `<span class="flag">${esc(S.country.flag)}</span>`;
      a.push(`<button class="btn" data-op="allow_current_country" title="${t("allow_country")}: ${esc(S.country.name)}">${t("allow_country")} ${c}</button>`);
    }
  }
  a.push(fix
    ? `<button class="btn icon" data-op="check_now" title="${t("check")}" aria-label="${t("check")}">${svg("recheck")}</button>`
    : `<button class="btn" data-op="check_now">${t("check")}</button>`);
  a.push(`<select class="btn" id="pauseSel"><option value="">${t("pause")}…</option><option value="15">${t("pause15")}</option><option value="60">${t("pause60")}</option><option value="0">${t("pauseF")}</option></select>`);
  return a;
}

function heroTitle() {
  if (S.paused) return S.pausedUntil === "forever" ? t("paused_forever") : `${t("paused_until")} ${esc(S.pausedUntil)}`;
  return t("h_" + S.status);
}

// The reason and countdown, then every running app with its own state:
// apps with their own rules can be fine while the rest are frozen.
function heroSub() {
  const lines = [];
  if (!S.paused && (S.status === "frozen" || S.status === "blocked") && S.reasonText) {
    lines.push(esc(S.reasonText) + (S.status === "frozen" && S.reason !== "net" && S.settings.close_apps ? ` · ${t("close_in")} ${S.graceLeft}&nbsp;${t("sec")}` : ""));
  }
  const run = S.apps.filter((a) => a.running);
  if (!run.length) lines.push(t("no_apps"));
  else lines.push(run.map((a) => `${esc(a.name)} — ${a.status === "frozen" ? t("frozen") : a.status === "blocked" ? t("blocked") : t("ok")}`).join(" · "));
  return lines.join("<br>");
}

function overview() {
  const icon = { ok: "check", frozen: "snow", starting: "snow", blocked: "stop", paused: "pause" }[S.status];
  const grace = S.settings.grace || 1;
  const bar = S.status === "frozen" && S.reason !== "net" && S.settings.close_apps ? `<div class="bar" style="width:${Math.max(0, (S.graceLeft / grace) * 100)}%"></div>` : "";
  const upd = S.update ? `<div class="banner"><b>${t("update_avail")} ${esc(S.update)}</b><span class="grow"></span><button class="btn primary" data-op="install_update">${t("update_btn")}</button></div>` : "";
  const ipV = S.offline ? t("offline") : S.ip ? esc(S.ip) : t("checking");
  const others = S.others && S.others.length ? `<div class="s">${t("others")}: ${S.others.map(esc).join(", ")}</div>` : "";
  const dnsTile = S.dns
    ? `<div class="card tile ${S.dnsLeak ? "warn" : ""}"><div class="k">${t("dns")}</div><div class="v">${esc(S.dns)}</div><div class="s">${country(S.dnsCountry)} · ${S.dnsLeak ? t("dns_leak") : t("dns_ok")}</div></div>`
    : `<div class="card tile"><div class="k">${t("dns")}</div><div class="v muted">—</div><div class="s">${t("dns_unknown")}</div></div>`;
  const netTile = S.network
    ? `<div class="card tile"><div class="k">${t("network")}</div><div class="v">${esc(S.network)}</div><div class="s">${t("trusted_net")}</div></div>`
    : `<div class="card tile"><div class="k">${t("network")}</div><div class="v">${S.netId ? t("usual_net") : t("unknown_net")}</div>${S.netId ? `<div class="s selectable">${t("router")} ${esc(S.netId)}</div><div class="row"><button class="btn small" data-go="networks">${t("trust")}…</button></div>` : ""}</div>`;
  const acts = heroActions();
  // Up to two actions sit at the end of the row; more go under the text.
  return `${upd}
  <div class="card hero ${S.status} ${acts.length > 2 ? "stack" : ""}">
    <div class="orb">${svg(icon)}</div>
    <div class="hero-text"><div class="title">${heroTitle()}</div><div class="sub">${heroSub()}</div></div>
    <div class="actions">${acts.join("")}</div>${bar}
  </div>
  <div class="grid">
    <div class="card tile"><div class="k">${t("ip")}</div><div class="v selectable">${ipV}</div><div class="s">${country(S.country)}</div>${others}</div>
    <div class="card tile ${S.reason === "v6" ? "warn" : ""}"><div class="k">${t("ipv6")}</div><div class="v selectable">${S.ip6 ? esc(S.ip6) : "—"}</div><div class="s">${S.ip6 ? country(S.country6) : t("ipv6_none")}</div></div>
    ${dnsTile}
    ${netTile}
  </div>
  <h2>${t("stats")}</h2>
  ${statsNumbers()}`;
}

// An empty state: an icon, what is missing, and what to do about it.
function emptyState(icon, title, hint = "") {
  return `<div class="card es"><i>${svg(icon)}</i><b>${title}</b>${hint ? `<span>${hint}</span>` : ""}</div>`;
}

function ruleEditor(app) {
  const r = app.rule || { ips: [], countries: [] };
  return `<div class="rule">
    <div class="hint">${t("rule_hint")}</div>
    ${allowEditor(r, app.id)}
    ${app.rule ? `<button class="btn small" style="margin-top:12px" data-op="remove_rule" data-app="${esc(app.id)}">${t("remove_rule")}</button>` : ""}
  </div>`;
}

// A steady colour per app name: a small app-icon-like tile, so the list is easy to scan.
function avatarStyle(name) {
  let h = 0;
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) % 360;
  return `background:linear-gradient(145deg,hsl(${h} 75% 62%),hsl(${(h + 28) % 360} 70% 46%));color:#fff`;
}

// The app's own icon when this computer has the app, else its letter.
function appAvatar(a) {
  if (a.icon) return `<img class="avatar app-ic" src="/api/icon/${encodeURIComponent(a.id)}" alt="">`;
  return `<div class="avatar" style="${avatarStyle(a.name)}">${esc(a.name[0])}</div>`;
}

function appRow(a) {
  const st = !a.running ? `<span class="badge">${t("not_running")}</span>`
    : `<span class="badge ${esc(a.status)}">${a.status === "frozen" ? t("frozen") : a.status === "blocked" ? t("blocked") : t("running")}</span>`;
  const rules = a.enabled
    ? `<button class="btn small" data-rule="${esc(a.id)}">${a.rule ? t("own_rules") : t("shared_rules")}<i class="chev">${svg("chev")}</i></button>` : "";
  const toggle = a.custom
    ? `<button class="btn small danger" data-op="remove_custom" data-app="${esc(a.id)}">${t("remove")}</button>`
    : `<label class="switch"><input type="checkbox" data-toggle="${esc(a.id)}" ${a.enabled ? "checked" : ""}><span></span></label>`;
  const meta = a.custom ? esc((a.match || []).join(", ")) : (a.enabled ? st : "");
  return `<div class="item">${appAvatar(a)}
    <div><div class="name">${esc(a.name)}</div><div class="meta">${meta}</div></div>
    <div class="end">${a.custom ? st : ""}${rules}${toggle}</div></div>
    ${openRule === a.id && a.enabled ? ruleEditor(a) : ""}`;
}

function apps() {
  const byState = (a, b) => (b.enabled - a.enabled) || a.name.localeCompare(b.name);
  const pre = S.apps.filter((a) => !a.custom && !a.browser).sort(byState);
  const web = S.apps.filter((a) => a.browser).sort(byState);
  const cus = S.apps.filter((a) => a.custom);
  return `<div class="card list">${pre.map(appRow).join("")}</div>
  <h2>${t("browsers")}</h2>
  <p class="hint">${t("browsers_hint")}${S.goos === "darwin" ? t("browsers_safari") : ""}</p>
  <div class="card list">${web.map(appRow).join("")}</div>
  <h2>${t("custom")}</h2>
  <div class="card list">${cus.map(appRow).join("")}
    <div class="item"><div class="add" style="flex:1;margin:0">
      <input id="cName" placeholder="${t("custom_name")}" style="max-width:180px">
      <input id="cMatch" placeholder="${t("custom_match")}">
      <button class="btn" id="cAdd">${t("add_custom")}</button></div></div>
  </div>`;
}

function allowEditor(r, app = "") {
  const rm = (op, v) => `<button title="${t("remove")}" data-op="${op}" data-app="${esc(app)}" data-value="${esc(v)}">×</button>`;
  const cs = r.countries.map((c) => `<span class="chip">${country(c)}${rm("remove_country", c.code)}</span>`).join("");
  const ips = r.ips.map((ip) => `<span class="chip mono selectable">${esc(ip)}${rm("remove_ip", ip)}</span>`).join("");
  const opts = countries.map((c) => `<option value="${esc(c.name)} (${esc(c.code)})">`).join("");
  const sec = (icon, tint, title, hint, n, chips, empty, input) => `<section class="sec">
    <div class="sec-h"><i class="ni" style="--ni:${tint}">${svg(icon)}</i><b>${title}</b>${n ? `<span class="count">${n}</span>` : ""}</div>
    <p class="sec-d">${hint}</p>
    ${chips ? `<div class="chips">${chips}</div>` : `<div class="empty">${empty}</div>`}
    <div class="ig">${input}</div></section>`;
  return sec("globe", "#30B0C7", t("countries"), t("sec_c_hint"), r.countries.length, cs, t("no_countries"),
      `<input list="cl-${esc(app)}" data-addc="${esc(app)}" placeholder="${t("country_ph")}"><datalist id="cl-${esc(app)}">${opts}</datalist><button class="btn" data-addcbtn="${esc(app)}">${t("add")}</button>`) +
    sec("pin", "#5856D6", t("ips"), t("sec_ip_hint"), r.ips.length, ips, t("no_ips"),
      `<input class="mono" data-addip="${esc(app)}" placeholder="${t("ip_ph")}"><button class="btn" data-addipbtn="${esc(app)}">${t("add")}</button>`);
}

function allow() {
  return `<div class="secs">${allowEditor(S.allow)}</div>`;
}

function networks() {
  const list = (S.trusted || []).map((n) => `<div class="item"><div class="avatar">${svg("networks")}</div>
    <div><div class="name">${esc(n.name)} ${n.current ? `<span class="badge ok">${t("here")}</span>` : ""}</div><div class="meta">${t("router")} <span class="mono selectable">${esc(n.id)}</span></div></div>
    <div class="end"><button class="btn small" data-rename="${esc(n.id)}" data-name="${esc(n.name)}">${t("rename")}</button><button class="btn small danger" data-op="untrust_network" data-value="${esc(n.id)}">${t("remove")}</button></div></div>`).join("");
  const cur = S.network ? "" : `<h2>${t("current_net")}</h2>${S.netId
    ? `<div class="card pad"><div class="meta">${t("router")} <span class="mono selectable">${esc(S.netId)}</span></div><div class="add"><input id="netName" placeholder="${t("net_name")}"><button class="btn primary" id="trustBtn">${t("trust")}</button></div></div>`
    : emptyState("networks", t("unknown_net"), t("es_net"))}`;
  return `${cur}<h2>${t("trusted_list")}</h2>${list ? `<div class="card list">${list}</div>` : emptyState("allow", t("no_trusted"), S.netId ? t("es_trusted") : "")}`;
}

function statsNumbers() {
  const tot = S.stats.days.reduce((a, d) => ({ saves: a.saves + d.saves, freezes: a.freezes + d.freezes, blocks: a.blocks + d.blocks, sec: a.sec + d.frozen_seconds }), { saves: 0, freezes: 0, blocks: 0, sec: 0 });
  const stat = (kind, n, label) => `<div class="card stat ${kind}"><div class="n">${n}</div><div class="l"><i></i>${label}</div></div>`;
  return `<div class="stats">${stat("ok", tot.saves, t("s_saves"))}${stat("warn", tot.freezes, t("s_freezes"))}${stat("bad", tot.blocks, t("s_blocks"))}${stat("", fmtDur(tot.sec), t("s_time"))}</div>`;
}

function chart() {
  const days = S.stats.days;
  const max = Math.max(1, ...days.map((d) => d.saves + d.blocks + Math.max(0, d.freezes - d.saves - d.blocks)));
  const W = 700, H = 150, base = H - 22, top = 8, bw = W / days.length;
  const grid = [0, 0.5, 1].map((k) => `<line x1="0" x2="${W}" y1="${base - (base - top) * k}" y2="${base - (base - top) * k}" stroke="var(--line)" ${k ? 'stroke-dasharray="2 4"' : ""}/>`).join("");
  const bars = days.map((d, i) => {
    const x = i * bw + bw * 0.28, w = bw * 0.44;
    const other = Math.max(0, d.freezes - d.saves - d.blocks);
    const sum = d.saves + d.blocks + other;
    let y = base, out = "";
    if (!sum) out = `<rect x="${x}" y="${base - 3}" width="${w}" height="3" rx="1.5" fill="var(--fill-h)"/>`;
    for (const [v, c] of [[d.saves, "var(--ok)"], [d.blocks, "var(--bad)"], [other, "var(--warn)"]]) {
      if (v <= 0) continue;
      const h = (v / max) * (base - top);
      y -= h;
      out += `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="3" fill="${c}"/>`;
    }
    const label = i % 2 === days.length % 2 || i === days.length - 1
      ? `<text x="${x + w / 2}" y="${H - 6}" text-anchor="middle" font-size="10.5" fill="var(--muted)">${d.date.slice(8)}.${d.date.slice(5, 7)}</text>` : "";
    return out + label;
  }).join("");
  return `<div class="card chart"><svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="none">${grid}${bars}</svg>
    <div class="legend"><span><i style="background:var(--ok)"></i>${t("legend_saves")}</span><span><i style="background:var(--bad)"></i>${t("legend_blocks")}</span><span><i style="background:var(--warn)"></i>${t("legend_other")}</span></div></div>`;
}

function stats() {
  const s = S.stats.session;
  return `${statsNumbers()}<h2>14 ${S.lang === "en" ? "days" : "дней"}</h2>${chart()}
    <h2>${t("session")}</h2><div class="card kv">${[[t("s_freezes"), s.Freezes], [t("s_saves"), s.Saves], [t("s_blocks"), s.Blocks], [t("s_time"), fmtDur(s.FrozenSeconds)]]
      .map(([k, v]) => `<div><span>${k}</span><b>${v}</b></div>`).join("")}</div>`;
}

function history() {
  if (!S.history || !S.history.length) return emptyState("history", t("history_empty"), t("es_hist"));
  return `<div class="card list hist">${S.history.map((h) => `<div class="item"><span class="dot ${esc(h.kind)}"></span><span class="time">${esc(h.date)} ${esc(h.time)}</span><span class="selectable">${esc(h.text)}</span></div>`).join("")}</div>`;
}

function toggleSet(key, title, desc, value) {
  return `<div class="set"><div class="txt"><b>${title}</b>${desc ? `<span>${desc}</span>` : ""}</div><label class="switch"><input type="checkbox" data-set="${key}" ${value ? "checked" : ""}><span></span></label></div>`;
}
function numSet(key, title, desc, value, unit = t("sec")) {
  return `<div class="set"><div class="txt"><b>${title}</b><span>${desc}</span></div><label class="num"><input type="number" min="0" data-num="${key}" value="${value}">${unit ? `<span>${unit}</span>` : ""}</label></div>`;
}
function selSet(key, title, desc, value, opts) {
  return `<div class="set"><div class="txt"><b>${title}</b>${desc ? `<span>${desc}</span>` : ""}</div><div class="field"><select data-sel="${key}">${opts.map(([v, l]) => `<option value="${v}" ${v === value ? "selected" : ""}>${l}</option>`).join("")}</select></div></div>`;
}

function settings() {
  const s = S.settings;
  return `<h2>${t("g_protect")}</h2><div class="card">
    ${toggleSet("freeze_on_network_change", t("o_freeze"), t("o_freeze_d"), s.freeze_on_network_change)}
    ${toggleSet("block_when_offline", t("o_offline"), t("o_offline_d"), s.block_when_offline)}
    ${toggleSet("check_ipv6", t("o_ipv6"), t("o_ipv6_d"), s.check_ipv6)}
    ${selSet("dns_check", t("o_dns"), t("o_dns_d"), s.dns_check, [["warn", t("dns_warn")], ["block", t("dns_block")], ["off", t("dns_off")]])}
    ${toggleSet("close_apps", t("o_closeapps"), s.close_apps ? t("o_closeapps_d") : t("o_closeapps_off"), s.close_apps)}
    ${s.close_apps ? numSet("grace", t("o_grace"), t("o_grace_d"), s.grace) + numSet("close_timeout", t("o_close"), t("o_close_d"), s.close_timeout) : ""}
    ${numSet("check_interval", t("o_interval"), t("o_interval_d"), s.check_interval)}
    ${numSet("min_agree", t("o_agree"), t("o_agree_d"), s.min_agree, "")}
  </div>
  <h2>${t("g_iface")}</h2><div class="card">
    ${selSet("menubar", t("o_menubar"), "", s.menubar, [["ip", t("mb_ip")], ["country", t("mb_country")], ["icon", t("mb_icon")]])}
    ${selSet("language", t("o_lang"), "", s.language, [["auto", t("lang_auto")], ["ru", "Русский"], ["en", "English"]])}
    ${toggleSet("notifications", t("o_notify"), "", s.notifications)}
  </div>
  <h2>${t("g_system")}</h2><div class="card">
    <div class="set"><div class="txt"><b>${t("o_autostart")}</b></div><label class="switch"><input type="checkbox" id="autostart" ${S.autostart ? "checked" : ""}><span></span></label></div>
    ${S.repoUrl ? toggleSet("update_check", t("o_update"), t("o_update_d"), s.update_check) : ""}
  </div>
  <h2>${t("g_files")}</h2><div class="card pad" style="display:flex;gap:8px;flex-wrap:wrap">
    <button class="btn" data-op="open_config">${t("open_config")}</button>
    <button class="btn" data-op="open_log">${t("open_log")}</button>
    ${S.repoUrl ? `<button class="btn" data-op="report">${t("report")}</button>` : ""}
  </div>`;
}

function about() {
  return `<div class="card about"><img src="icon.png" alt=""><div class="big">Perimeter ${esc(S.version)}</div>
    <p>${t("about_text")}</p>
    ${S.repoUrl ? `<div class="links"><button class="btn" data-url="${esc(S.repoUrl)}">${t("github")}</button></div>` : ""}
    <p style="font-size:12px">${t("based_on")} · ${t("license")}</p></div>
  ${S.notes && S.notes.length ? `<h2>${t("whats_new")} ${esc(S.version)}</h2>
  <div class="card notes"><ul>${S.notes.map((n) => `<li><b>${esc(n.title)}</b>${n.text ? ` - ${esc(n.text)}` : ""}</li>`).join("")}</ul>
  ${S.repoUrl ? `<button class="btn small" data-url="${esc(S.repoUrl)}/blob/main/CHANGELOG.md">${t("all_changes")}</button>` : ""}</div>` : ""}`;
}

// ---------- leak check ----------

function ccName(code) {
  const c = countries.find((x) => x.code === code);
  return code ? (c ? country(c) : esc(code)) : "?";
}

// Each finding: a level (ok, warn, bad, info), a title and a line of explanation.
function leakFindings(r) {
  const out = [];
  const v4 = r.v4.filter((a) => !a.error), ips = [...new Set(v4.map((a) => a.ip))];
  const addr = (a) => `<b class="selectable">${esc(a.ip)}</b> · ${ccName(a.country)}`;
  if (!v4.length) out.push(["warn", t("lk_v4"), t("lk_v4_none")]);
  else if (ips.length > 1) out.push(["bad", t("lk_v4"), `${t("lk_v4_split")} ${ips.map((ip) => addr(v4.find((a) => a.ip === ip))).join(", ")}`]);
  else if (!v4[0].allowed) out.push(["warn", t("lk_v4"), `${addr(v4[0])}. ${t("lk_v4_notallowed")}`]);
  else out.push(["ok", t("lk_v4"), `${addr(v4[0])}. ${t("lk_v4_ok")}`]);

  if (!r.proxy) out.push(["info", t("lk_direct"), t("lk_direct_none")]);
  else {
    const d = r.direct.find((a) => !a.error);
    if (!d || ips.includes(d.ip)) out.push(["ok", t("lk_direct"), `${t("lk_direct_ok")} <span class="muted">(${esc(r.proxy)})</span>`]);
    else if (!d.allowed) out.push(["bad", t("lk_direct"), `${addr(d)}. ${t("lk_direct_bad")}`]);
    else out.push(["warn", t("lk_direct"), `${addr(d)}. ${t("lk_direct_warn")}`]);
  }

  if (r.v6off) out.push(["info", t("lk_v6"), t("lk_v6_off")]);
  else {
    const v6 = r.v6.filter((a) => !a.error);
    if (!v6.length) out.push(["ok", t("lk_v6"), t("lk_v6_none")]);
    else if (v6.some((a) => !a.allowed)) out.push(["bad", t("lk_v6"), `${addr(v6.find((a) => !a.allowed))}. ${t("lk_v6_bad")}`]);
    else out.push(["ok", t("lk_v6"), `${addr(v6[0])}. ${t("lk_v6_ok")}`]);
  }

  if (!r.dns) out.push(["info", t("lk_dns"), t("lk_dns_none")]);
  else if (v4.length && r.dns.country && v4[0].country && r.dns.country !== v4[0].country) out.push(["warn", t("lk_dns"), `${addr(r.dns)}. ${t("lk_dns_warn")}`]);
  else out.push(["ok", t("lk_dns"), `${addr(r.dns)}. ${t("lk_dns_ok")}`]);

  out.push(["info", t("lk_web"), t("lk_web_d")]);
  return out;
}

function leaks() {
  const L = S.leaks || {};
  const btn = `<button class="btn primary" data-op="check_leaks" ${L.running ? "disabled" : ""}>${L.running ? t("lk_running") : L.report ? t("lk_again") : t("lk_run")}</button>`;
  if (!L.report) return `<div class="card pad leak-start">${btn}</div>`;
  const f = leakFindings(L.report);
  const worst = f.some((x) => x[0] === "bad") ? "bad" : f.some((x) => x[0] === "warn") ? "warn" : "ok";
  const at = new Date(L.report.at).toLocaleTimeString(S.lang === "en" ? "en-GB" : "ru-RU", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  const icon = { ok: "check", warn: "snow", bad: "stop", info: "about" };
  const rows = (list) => list.map((a) => `<tr><td class="mono muted">${esc(a.service.replace(/^https?:\/\//, ""))}</td><td class="mono selectable">${a.error ? `<span class="na">${t("lk_noans")}</span>` : esc(a.ip)}</td><td>${a.error ? "" : ccName(a.country)}</td></tr>`).join("");
  return `<div class="card leak-sum ${worst}"><div class="orb">${svg(icon[worst])}</div>
      <div><div class="title">${t("lk_" + worst)}</div><div class="sub">${t("lk_at")} ${esc(at)}</div></div><span class="grow"></span>${btn}</div>
    <div class="card list">${f.map(([lv, title, text]) => `<div class="item leak ${lv}"><span class="lv">${svg(icon[lv])}</span><div><div class="name">${title}</div><div class="meta">${text}</div></div></div>`).join("")}</div>
    <h2>${t("lk_services")}</h2>
    <div class="card pad"><table class="lk-table"><thead><tr><th>${t("lk_svc")}</th><th>${t("lk_addr")}</th><th>${t("lk_cc")}</th></tr></thead>${rows(L.report.v4)}${rows(L.report.direct || [])}${rows(L.report.v6 || [])}</table></div>`;
}

const RENDER = { overview, apps, allow, networks, leaks, stats, history, settings, about };

function nav() {
  const bad = S.status === "frozen" || S.status === "blocked";
  const link = (p) => `<a href="#${p}" class="${p === page ? "on" : ""}"><i class="ni" style="--ni:${NAV_TINT[p]}">${svg(p)}</i><span>${t(p)}</span>${p === "overview" && bad ? `<span class="badge ${S.status}">!</span>` : ""}</a>`;
  document.getElementById("nav").innerHTML = NAV.map(([label, pages]) =>
    `<div class="ng${label ? "" : " bottom"}">${label ? `<div class="nl">${t(label)}</div>` : ""}${pages.map(link).join("")}</div>`).join("");
  document.getElementById("ver").textContent = S.version;
}

function render() {
  if (!PAGES.includes(page)) page = "overview";
  document.documentElement.lang = S.lang;
  document.documentElement.classList.toggle("mac", S.goos === "darwin");
  nav();
  const main = document.getElementById("main");
  const keep = main.scrollTop;
  const lead = t("lead_" + page) === "lead_" + page ? "" : t("lead_" + page);
  main.innerHTML = `<div class="page"><header class="ph"><h1>${t(page)}</h1>${lead ? `<p class="lead">${lead}</p>` : ""}</header>${RENDER[page]()}</div>`;
  main.scrollTop = keep;
}

// Re-render on changes, but never under the user's fingers.
async function refresh(force = false) {
  try {
    const r = await fetch("/api/state");
    if (!r.ok) return;
    const txt = await r.text();
    if (!force && txt === lastJSON) return;
    lastJSON = txt;
    S = JSON.parse(txt);
    const a = document.activeElement;
    if (!force && a && a.closest("main") && (a.tagName === "INPUT" || a.tagName === "SELECT")) return;
    render();
  } catch (_) { /* Perimeter restarting: try again on the next tick */ }
}

// ---------- events ----------

function countryCode(v) {
  const m = /\(([A-Za-z]{2})\)\s*$/.exec(v.trim());
  if (m) return m[1].toUpperCase();
  const byName = countries.find((c) => c.name.toLowerCase() === v.trim().toLowerCase());
  return byName ? byName.code : v.trim().toUpperCase();
}

document.addEventListener("click", async (e) => {
  const el = e.target.closest("[data-op],[data-url],[data-go],[data-rule],[data-addcbtn],[data-addipbtn],[data-rename],#cAdd,#trustBtn");
  if (!el) return;
  e.preventDefault();
  if (el.dataset.url) return api("open_url", { value: el.dataset.url });
  if (el.dataset.go) { location.hash = el.dataset.go; return; }
  if (el.dataset.rule !== undefined) { openRule = openRule === el.dataset.rule ? null : el.dataset.rule; return render(); }
  if (el.dataset.addcbtn !== undefined) {
    const app = el.dataset.addcbtn, inp = document.querySelector(`[data-addc="${CSS.escape(app)}"]`);
    if (!inp.value.trim()) return;
    if (app && !S.apps.find((a) => a.id === app).rule) await api("set_rule", { app, countries: [countryCode(inp.value)] });
    else await api("add_country", { app, value: countryCode(inp.value) });
    return;
  }
  if (el.dataset.addipbtn !== undefined) {
    const app = el.dataset.addipbtn, inp = document.querySelector(`[data-addip="${CSS.escape(app)}"]`);
    if (!inp.value.trim()) return;
    if (app && !S.apps.find((a) => a.id === app).rule) await api("set_rule", { app, ips: [inp.value.trim()] });
    else await api("add_ip", { app, value: inp.value.trim() });
    return;
  }
  if (el.dataset.rename) {
    // Inline rename (WebKit windows have no prompt()).
    const row = el.closest(".item"), nameEl = row.querySelector(".name");
    nameEl.innerHTML = `<input class="rn" value="${esc(el.dataset.name)}" style="border:1px solid var(--line);border-radius:8px;padding:4px 8px">`;
    const inp = nameEl.querySelector("input");
    inp.focus();
    inp.select();
    const done = async () => { inp.onblur = null; if (inp.value.trim()) await api("rename_network", { app: el.dataset.rename, value: inp.value }); else render(); };
    inp.onblur = done;
    inp.onkeydown = (k) => { if (k.key === "Enter") inp.blur(); if (k.key === "Escape") { inp.onblur = null; render(); } };
    return;
  }
  if (el.id === "cAdd") {
    const name = document.getElementById("cName").value, match = document.getElementById("cMatch").value.split(",");
    if (await api("add_custom", { value: name, ips: match })) toast(t("saved"));
    return;
  }
  if (el.id === "trustBtn") return api("trust_network", { value: document.getElementById("netName").value });
  const d = el.dataset;
  if (d.op === "remove_rule" && openRule === d.app) openRule = null;
  await api(d.op, { app: d.app || "", value: d.value || "" });
});

document.addEventListener("change", async (e) => {
  const el = e.target;
  if (el.id === "pauseSel" && el.value !== "") return api("pause", { minutes: +el.value });
  if (el.dataset.toggle) return api("toggle_app", { app: el.dataset.toggle });
  if (el.id === "autostart") return api("autostart", { value: el.checked ? "on" : "off" });
  const set = (k, v) => api("settings", { settings: { [k]: v } }).then((ok) => ok && toast(t("saved")));
  if (el.dataset.set) return set(el.dataset.set, el.checked);
  if (el.dataset.num) return set(el.dataset.num, +el.value);
  if (el.dataset.sel) return set(el.dataset.sel, el.value);
});

document.addEventListener("keydown", (e) => {
  if (e.key !== "Enter") return;
  const el = e.target;
  const btn = el.dataset.addc !== undefined ? document.querySelector(`[data-addcbtn="${CSS.escape(el.dataset.addc)}"]`)
    : el.dataset.addip !== undefined ? document.querySelector(`[data-addipbtn="${CSS.escape(el.dataset.addip)}"]`)
    : el.id === "netName" ? document.getElementById("trustBtn") : el.id === "cMatch" ? document.getElementById("cAdd") : null;
  if (btn) btn.click();
});

window.addEventListener("hashchange", () => { page = location.hash.slice(1); openRule = null; render(); });

(async () => {
  await refresh(true);
  countries = await fetch("/api/countries").then((r) => r.json()).catch(() => []);
  if (page === "allow" || page === "apps") render();
  setInterval(refresh, 1000);
})();
