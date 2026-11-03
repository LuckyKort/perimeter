// The tray window. Kordon's native side keeps this page loaded in a hidden
// window and calls kordonShow() / kordonHide() when the icon is clicked, so
// opening it costs nothing. While shown it polls /api/state once a second.
"use strict";

const L = {
  ru: {
    ob_t: "Привет! Настроим за минуту", ob_ip: "Я разрешил текущий адрес:", ob_q: "Это адрес VPN?", ob_yes: "Да, это VPN", ob_no: "Нет",
    ob_off: "Тогда включи VPN: как только появится его адрес, нажми «Разрешить этот IP» ниже. До этого программы на паузе.",
    ob_country: "Разрешить всю страну", ob_apps: "Ещё запущено - защищать?", ob_ping: "Учитывать меня в счётчике пользователей Kordon",
    ob_ping_d: "Раз в день: только версия и система. Без IP, без номера установки.", ob_done: "Готово",
    ping_t: "Помочь посчитать пользователей?", ping_d: "Раз в день Kordon отправит только версию и систему. Без IP и без номера установки.", ping_yes: "Да, учитывать", ping_no: "Нет",
    h_ok: "Защита включена", h_frozen: "Приложения на паузе", h_blocked: "Приложения закрыты", h_starting: "Проверяю IP…",
    h_paused: "Защита на паузе", paused_until: "до", paused_forever: "до перезапуска",
    allowed: "разрешено", not_allowed: "не разрешено", offline: "Нет интернета", checking: "Проверяю…",
    v6_none: "IPv6 закрыт", v6: "IPv6", dns_ok: "DNS в той же стране", dns_leak: "DNS мимо VPN", others: "Другой IP у части сервисов",
    trusted: "Доверенная сеть", close_in: "закрою через", sec: "сек", s: "с",
    protect: "Защищаю", frozen_l: "На паузе", none: "Защищаемые приложения не запущены", none_set: "Приложения не выбраны.", choose: "Выбрать",
    running: "работает", frozen: "пауза", blocked: "закрыто",
    pause: "Пауза", check: "Проверить IP", trust: "Эта сеть\nдоверенная", trusted_a: "Сеть\nдоверенная", panel: "Панель",
    p15: "15 мин", p60: "1 час", pF: "До перезапуска", unpause: "Включить защиту",
    allow_ip: "Разрешить этот IP", allow_country: "Разрешить всю страну", pause15: "Пауза защиты на 15 мин",
    upd: "Kordon", upd_btn: "Обновить", whats: "Что нового", hide: "Скрыть", installing: "Обновляю до", of: "из", mb: "МБ",
    upd_note: "Проверю контрольную сумму и перезапущусь. Защита не выключится.",
    updated: "Обновлено до", all: "Все изменения",
    open: "Открыть панель →", checked: "Проверяю", trusted_t: "Сеть добавлена в доверенные",
  },
  en: {
    ob_t: "Hi! A one-minute setup", ob_ip: "I've allowed the current address:", ob_q: "Is this your VPN's address?", ob_yes: "Yes, the VPN", ob_no: "No",
    ob_off: "Then turn the VPN on: once its address shows up, press \"Allow this IP\" below. Until then the apps stay paused.",
    ob_country: "Allow the whole country", ob_apps: "Also running - protect them?", ob_ping: "Count me among Kordon's users",
    ob_ping_d: "Once a day: just the version and the OS. No IP, no install id.", ob_done: "Done",
    ping_t: "Help count Kordon's users?", ping_d: "Once a day Kordon sends only its version and the OS. No IP, no install id.", ping_yes: "Yes, count me", ping_no: "No",
    h_ok: "Protection is on", h_frozen: "Apps are paused", h_blocked: "Apps are closed", h_starting: "Checking IP…",
    h_paused: "Protection paused", paused_until: "until", paused_forever: "until restart",
    allowed: "allowed", not_allowed: "not allowed", offline: "No internet", checking: "Checking…",
    v6_none: "IPv6 is off", v6: "IPv6", dns_ok: "DNS in the same country", dns_leak: "DNS bypasses the VPN", others: "Some services see another IP",
    trusted: "Trusted network", close_in: "closing in", sec: "sec", s: "s",
    protect: "Protecting", frozen_l: "Paused", none: "No protected apps are running", none_set: "No apps selected.", choose: "Choose",
    running: "running", frozen: "paused", blocked: "closed",
    pause: "Pause", check: "Check IP", trust: "Trust this\nnetwork", trusted_a: "Trusted\nnetwork", panel: "Panel",
    p15: "15 min", p60: "1 hour", pF: "Until restart", unpause: "Turn protection on",
    allow_ip: "Allow this IP", allow_country: "Allow the whole country", pause15: "Pause protection for 15 min",
    upd: "Kordon", upd_btn: "Update", whats: "What's new", hide: "Hide", installing: "Updating to", of: "of", mb: "MB",
    upd_note: "I'll check the checksum and restart. Protection stays on.",
    updated: "Updated to", all: "All changes",
    open: "Open the panel →", checked: "Checking", trusted_t: "Network added to trusted",
  },
};

const ICONS = {
  panel: '<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M9 4v16"/>',
  pause: '<path d="M9 5v14M15 5v14"/>',
  check: '<path d="M20 12a8 8 0 1 1-2.3-5.6M20 4v5h-5"/>',
  home: '<path d="M3 11l9-7 9 7v9H3z"/>',
  homeok: '<path d="M3 11l9-7 9 7v9H3z"/><path d="M9 14.5l2 2 4-4"/>',
  play: '<path d="M7 5l12 7-12 7z"/>',
};
const svg = (n) => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${ICONS[n]}</svg>`;
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const t = (k) => (L[S?.lang] || L.ru)[k] ?? L.en[k] ?? k;
const br = (s) => esc(s).replace(/\n/g, "<br>");

let S = null;
let lastHTML = "";
let pauseOpen = false, notesOpen = false, hiddenUpdate = "";
let timer = null, inTimer = null;

// Native bridge: WKWebView message handler on macOS, bound functions on Windows.
function native(msg) {
  try {
    if (window.webkit?.messageHandlers?.kordon) window.webkit.messageHandlers.kordon.postMessage(msg);
    else if (window.chrome?.webview) window.chrome.webview.postMessage(JSON.stringify(msg));
  } catch (_) { /* no native side: opened in a browser for testing */ }
}

async function api(op, data = {}) {
  const r = await fetch("/api/do", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ op, ...data }) });
  await refresh();
  return r.ok;
}

function country(c) {
  if (!c) return "";
  return S.goos === "windows" ? `${esc(c.code)} · ${esc(c.name)}` : `${esc(c.flag)} ${esc(c.name)}`;
}

function avatar(name) {
  let h = 0;
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) % 360;
  return `background:hsl(${h} 60% 50% / .2);color:hsl(${h} 70% 62%)`;
}

let obAnswer = "", obPing = false;

// First start: confirm the auto-allowed IP is the VPN's, offer the running apps, ask about the count.
function onboardCard() {
  const o = S.onboarding;
  if (!o || !o.pending) return "";
  const ip = S.ip ? `<b>${esc(S.ip)}</b>${S.country ? ` · ${country(S.country)}` : ""}` : esc(t("checking"));
  let q = "";
  if (obAnswer === "") {
    q = `<div class="ob-q">${t("ob_q")}</div><div class="b"><button class="btn pri" data-do="ob_yes">${t("ob_yes")}</button><button class="btn" data-do="ob_no">${t("ob_no")}</button></div>`;
  } else if (obAnswer === "yes" && S.country && !(S.allow?.countries || []).some((c) => c.code === S.country.code)) {
    q = `<div class="b"><button class="btn" data-op="allow_current_country">${t("ob_country")} <small>${country(S.country)}</small></button></div>`;
  } else if (obAnswer === "no") {
    q = `<div class="note">${t("ob_off")}</div>`;
  }
  const sug = (o.suggest || []).map((a) => `<button class="sug" data-op="toggle_app" data-app="${esc(a.id)}"><span class="av" style="${avatar(a.name)}">${esc(a.name[0])}</span>${esc(a.name)} <span>+</span></button>`).join("");
  return `<div class="upd ob"><div class="h"><span class="v">${t("ob_t")}</span></div>
    <div class="ob-ip">${t("ob_ip")} ${ip}</div>${q}
    ${sug ? `<div class="ob-l">${t("ob_apps")}</div><div class="apps">${sug}</div>` : ""}
    <label class="chk"><input type="checkbox" id="obPing" ${obPing ? "checked" : ""}><span><b>${t("ob_ping")}</b><br><small>${t("ob_ping_d")}</small></span></label>
    <div class="b"><button class="btn ${obAnswer ? "pri" : ""}" data-do="ob_done">${t("ob_done")}</button></div></div>`;
}

// For people who updated from a version without the count: asked once, nothing is sent until they say yes.
function pingCard() {
  if (S.usagePing !== "ask" || S.onboarding?.pending) return "";
  return `<div class="upd ping"><div class="h"><span class="v">${t("ping_t")}</span></div>
    <div class="note" style="margin:6px 0 10px">${t("ping_d")}</div>
    <div class="b"><button class="btn pri" data-op="usage_ping" data-value="on">${t("ping_yes")}</button><button class="btn" data-op="usage_ping" data-value="off">${t("ping_no")}</button></div></div>`;
}

function updateCard() {
  const u = S.updateInfo;
  if (u && u.version && (u.installing || hiddenUpdate !== u.version)) {
    if (u.installing) {
      const pct = u.total ? Math.min(100, (u.done / u.total) * 100) : 0;
      const size = u.total ? `${(u.done / 1048576).toFixed(1)} ${t("of")} ${(u.total / 1048576).toFixed(1)} ${t("mb")}` : "";
      return `<div class="upd"><div class="h"><span class="v">${t("installing")} ${esc(u.version)}</span><span class="sp"></span><span class="note">${size}</span></div>
        <div class="bar ${u.total ? "" : "unknown"}"><i style="${u.total ? `width:${pct}%` : ""}"></i></div>
        <div class="note">${t("upd_note")}</div></div>`;
    }
    const notes = u.notes || [];
    const shown = notesOpen ? notes : notes.slice(0, 3);
    return `<div class="upd"><div class="h"><span class="new">NEW</span><span class="v">${t("upd")} ${esc(u.version)}</span><span class="sp"></span><button class="x" data-do="hideupd" title="${t("hide")}">×</button></div>
      ${shown.length ? `<ul>${shown.map((n) => `<li><b>${esc(n.title)}</b>${notesOpen && n.text ? ` <span>${esc(n.text)}</span>` : ""}</li>`).join("")}</ul>` : "<div style='height:10px'></div>"}
      <div class="b"><button class="btn pri" data-op="install_update">${t("upd_btn")}</button>${notes.length ? `<button class="btn" data-do="notes">${notesOpen ? t("hide") : t("whats")}</button>` : ""}</div></div>`;
  }
  const w = S.whatsNew;
  if (w && w.version) {
    const notes = w.notes || [];
    return `<div class="upd"><div class="h"><span class="new">✓</span><span class="v">${t("updated")} ${esc(w.version)}</span><span class="sp"></span><button class="x" data-op="dismiss_whatsnew" title="${t("hide")}">×</button></div>
      <ul>${notes.slice(0, 4).map((n) => `<li><b>${esc(n.title)}</b>${n.text ? ` <span>${esc(n.text.length > 90 ? n.text.slice(0, 88) + "…" : n.text)}</span>` : ""}</li>`).join("")}</ul>
      <div class="b"><button class="btn" data-panel="about">${t("all")}</button></div></div>`;
  }
  return "";
}

function hero() {
  const st = S.status;
  let title = t("h_" + st);
  if (S.paused) title += " " + (S.pausedUntil === "forever" ? t("paused_forever") : `${t("paused_until")} ${esc(S.pausedUntil)}`);
  const countdown = st === "frozen" && S.reason !== "net" && !S.paused;
  const grace = S.settings.grace || 20;
  const ring = countdown
    ? `<div class="ring"><svg viewBox="0 0 58 58"><circle class="bg" cx="29" cy="29" r="26"/><circle class="fg" cx="29" cy="29" r="26" style="stroke-dashoffset:${(163.4 * (1 - Math.max(0, S.graceLeft) / grace)).toFixed(1)}"/></svg><span>${S.graceLeft}<small>${t("sec")}</small></span></div>`
    : "";
  const ipText = S.offline ? t("offline") : S.ip || t("checking");
  const where = S.country ? `${country(S.country)} · ${S.allowedNow ? t("allowed") : t("not_allowed")}` : "&nbsp;";
  const tags = [];
  if (countdown && S.reasonText) tags.push(`<span class="tag">${esc(S.reasonText)}</span>`);
  if (S.network) tags.push(`<span class="tag good">${t("trusted")}: ${esc(S.network)}</span>`);
  if (S.settings.check_ipv6 && !S.offline && S.ip) {
    if (!S.ip6) tags.push(`<span class="tag good">${t("v6_none")}</span>`);
    else tags.push(`<span class="tag ${S.reason === "v6" ? "warn" : ""}">${t("v6")}: ${S.country6 ? esc(S.country6.code) : "?"}</span>`);
  }
  if (S.dns && S.settings.dns_check !== "off") tags.push(S.dnsLeak ? `<span class="tag warn">${t("dns_leak")}</span>` : `<span class="tag good">${t("dns_ok")}</span>`);
  if (S.others && S.others.length) tags.push(`<span class="tag warn">${t("others")}</span>`);
  const cls = S.paused ? "paused" : st;
  return `<div class="hero ${cls}">${ring}
    <div class="st"><span class="dot"></span>${title}</div>
    <div class="country">${where}</div>
    <div class="ip ${ipText.length > 16 ? "small" : ""}">${esc(ipText)}</div>
    ${tags.length ? `<div class="tags">${tags.join("")}</div>` : ""}</div>`;
}

function apps() {
  const run = S.apps.filter((a) => a.enabled && a.running);
  const frozen = run.some((a) => a.status === "frozen");
  if (!run.length) {
    const any = S.apps.some((a) => a.enabled);
    return `<div class="lbl">${t("protect")}</div><div class="none">${any ? t("none") : `${t("none_set")} <button data-panel="apps">${t("choose")}</button>`}</div>`;
  }
  const chip = (a) => {
    const st = a.status === "frozen" ? "frozen" : a.status === "blocked" ? "blocked" : "ok";
    const label = st === "frozen" ? "❄ " + t("frozen") : st === "blocked" ? t("blocked") : t("running");
    return `<span class="app ${st}"><span class="av" style="${avatar(a.name)}">${esc(a.name[0])}</span>${esc(a.name)} <span class="s">${label}</span></span>`;
  };
  return `<div class="lbl">${frozen ? t("frozen_l") : t("protect")}</div><div class="apps">${run.map(chip).join("")}</div>`;
}

function actions() {
  if (S.paused) return `<div class="stack"><button class="btn" data-op="unpause">${svg("play")}${t("unpause")}</button></div>`;
  const alarm = (S.status === "frozen" && S.reason !== "net") || S.status === "blocked";
  if (alarm && S.ip && !S.allowedNow && (S.reason === "ip" || S.reason === "v6")) {
    const c = S.country ? `<button class="btn" data-op="allow_current_country">${t("allow_country")} <small>${country(S.country)}</small></button>` : "";
    return `<div class="stack"><button class="btn" data-op="allow_current_ip">${t("allow_ip")}</button>${c}
      <button class="btn" data-op="pause" data-min="15">${t("pause15")}</button></div>`;
  }
  const third = S.network
    ? `<button class="act" data-panel="networks">${svg("homeok")}${br(t("trusted_a"))}</button>`
    : S.netId
      ? `<button class="act" data-do="trust">${svg("home")}${br(t("trust"))}</button>`
      : `<button class="act" data-panel="overview">${svg("panel")}${t("panel")}</button>`;
  return `<div class="acts">
    <button class="act ${pauseOpen ? "on" : ""}" data-do="pause">${svg("pause")}${t("pause")}</button>
    <button class="act" data-do="check">${svg("check")}${t("check")}</button>
    ${third}</div>
    ${pauseOpen ? `<div class="chips"><button data-op="pause" data-min="15">${t("p15")}</button><button data-op="pause" data-min="60">${t("p60")}</button><button data-op="pause" data-min="0">${t("pF")}</button></div>` : ""}`;
}

function render() {
  const html = `<div class="top"><img src="icon.png" alt=""><b>Kordon</b><span class="sp"></span>
      <button class="ib" data-panel="overview" title="${t("panel")}">${svg("panel")}</button></div>
    ${onboardCard() || updateCard() || pingCard()}${hero()}${apps()}${actions()}
    <div class="foot"><button class="open" data-panel="overview">${t("open")}</button><button class="studio" data-url="${esc(S.studioUrl)}">efimovdev.ru</button></div>`;
  if (html === lastHTML) return;
  lastHTML = html;
  document.getElementById("pop").innerHTML = html;
  fit();
}

// Tell the native window how tall the content is.
let lastH = 0;
function fit() {
  const h = Math.ceil(document.getElementById("pop").scrollHeight);
  if (h !== lastH) {
    lastH = h;
    native({ op: "resize", height: h });
  }
}

async function refresh() {
  try {
    const r = await fetch("/api/state");
    if (!r.ok) return;
    S = await r.json();
    document.documentElement.lang = S.lang;
    render();
  } catch (_) { /* Kordon is restarting */ }
}

let toastTimer;
function toast(msg) {
  document.querySelector(".toast")?.remove();
  const el = document.createElement("div");
  el.className = "toast";
  el.textContent = msg;
  document.body.appendChild(el);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.remove(), 2200);
}

document.addEventListener("click", async (e) => {
  const b = e.target.closest("button");
  if (!b) return;
  if (b.dataset.url) {
    await api("open_url", { value: b.dataset.url });
    native({ op: "hide" });
    return;
  }
  if (b.dataset.panel) {
    await api("open_panel", { value: b.dataset.panel });
    native({ op: "hide" });
    return;
  }
  if (b.dataset.op) {
    const op = b.dataset.op;
    if (op === "pause") pauseOpen = false;
    const data = {};
    if (b.dataset.min !== undefined) data.minutes = +b.dataset.min;
    if (b.dataset.value !== undefined) data.value = b.dataset.value;
    if (b.dataset.app !== undefined) data.app = b.dataset.app;
    await api(op, data);
    return;
  }
  switch (b.dataset.do) {
    case "pause": pauseOpen = !pauseOpen; lastHTML = ""; render(); break;
    case "notes": notesOpen = !notesOpen; lastHTML = ""; render(); break;
    case "hideupd": hiddenUpdate = S.updateInfo.version; lastHTML = ""; render(); break;
    case "ob_yes": obAnswer = "yes"; lastHTML = ""; render(); break;
    case "ob_no": obAnswer = "no"; await api("onboard_not_vpn"); break;
    case "ob_done": await api("onboard_done", { value: document.getElementById("obPing")?.checked ? "on" : "off" }); break;
    case "check": toast(t("checked") + "…"); await api("check_now"); break;
    case "trust": if (await api("trust_network")) toast(t("trusted_t")); break;
  }
});

document.addEventListener("change", (e) => {
  if (e.target.id === "obPing") obPing = e.target.checked; // survives the once-a-second re-render
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") native({ op: "hide" });
});

window.kordonShow = () => {
  pauseOpen = false;
  const pop = document.getElementById("pop");
  // The entrance animation plays once per opening, not on every refresh.
  pop.classList.remove("in");
  void pop.offsetWidth;
  pop.classList.add("in");
  clearTimeout(inTimer);
  inTimer = setTimeout(() => pop.classList.remove("in"), 600);
  refresh();
  clearInterval(timer);
  timer = setInterval(refresh, 1000);
};
window.kordonHide = () => {
  clearInterval(timer);
  timer = null;
};

if (navigator.userAgent.includes("Windows")) document.documentElement.classList.add("win");
else if (navigator.userAgent.includes("Mac")) document.documentElement.classList.add("mac");
new ResizeObserver(fit).observe(document.getElementById("pop"));
refresh();
// Opened on its own (in a browser while testing): behave as if shown.
if (!window.webkit?.messageHandlers?.kordon && !window.chrome?.webview) window.kordonShow();
