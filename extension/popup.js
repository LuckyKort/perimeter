import { DEFAULT_SITES, clean } from "./sites.js";

const ru = !navigator.language.startsWith("en");
const T = ru ? {
  ok: "IP в порядке - ИИ-сайты работают", pause: "ИИ-сайты на паузе", none: "Perimeter не запущен",
  noneNote: "Без запущенного приложения Perimeter вкладки не защищены. Запустите его.",
  pauseNote: "Вернутся сами, как только VPN снова заработает.",
  sites: "Что ставить на паузу", extra: "Свои сайты блокируются на уровне сети; открытые вкладки с ними лучше перезагрузить.",
  rtcAbout: "Браузер может показать сайтам ваш настоящий адрес через WebRTC, даже когда весь остальной трафик идёт через VPN или прокси.",
  check: "Проверить", fixOn: "Закрыть утечку", fixOff: "Вернуть как было", checking: "Проверяю…",
  rtcNone: "WebRTC не выдаёт сайтам ни одного адреса. Утечки нет.",
  rtcSame: "WebRTC показывает тот же адрес, что и VPN: ", rtcLeak: "Утечка: WebRTC показывает сайтам адрес ",
  rtcLeak2: " - он не совпадает с адресом VPN. Нажмите «Закрыть утечку».", rtcUnknown: "WebRTC видит адрес ", rtcUnknown2: ". Запустите Perimeter, чтобы сравнить его с IP.",
  fixed: "Готово: WebRTC больше не выдаёт адрес в обход прокси и VPN.",
} : {
  ok: "IP is fine - AI sites work", pause: "AI sites are paused", none: "Perimeter isn't running",
  noneNote: "Without the Perimeter app running, tabs aren't protected. Start it.",
  pauseNote: "They carry on by themselves once the VPN is back.",
  sites: "What to pause", extra: "Your own sites are blocked at the network level; reload their open tabs.",
  rtcAbout: "A browser can show sites your real address through WebRTC even when all other traffic goes through the VPN or a proxy.",
  check: "Check", fixOn: "Close the leak", fixOff: "Undo", checking: "Checking…",
  rtcNone: "WebRTC gives sites no address at all. No leak.",
  rtcSame: "WebRTC shows the same address as the VPN: ", rtcLeak: "Leak: WebRTC shows sites the address ",
  rtcLeak2: " - not the VPN's. Press \"Close the leak\".", rtcUnknown: "WebRTC sees the address ", rtcUnknown2: ". Start Perimeter to compare it with the IP.",
  fixed: "Done: WebRTC no longer gives out an address around the proxy and VPN.",
};
const $ = (id) => document.getElementById(id);
let S = null;

async function renderState() {
  S = await chrome.runtime.sendMessage({ type: "state" });
  const cls = !S.ok ? "" : S.pause ? "warn" : "ok";
  $("state").className = "state " + cls;
  $("stateText").textContent = !S.ok ? T.none : S.pause ? T.pause : T.ok;
  $("ip").textContent = S.ok && S.ip ? `${S.country?.flag || ""} ${S.ip}` : "";
  $("stateNote").textContent = !S.ok ? T.noneNote : S.pause ? (S.reason ? S.reason + ". " : "") + T.pauseNote : "";
  $("ver").textContent = S.ok ? "Perimeter " + S.perimeter : "";
}

async function renderSites() {
  const { off = [], extra = [] } = await chrome.storage.local.get(["off", "extra"]);
  $("sites").innerHTML = "";
  for (const d of [...DEFAULT_SITES, ...extra]) {
    const l = document.createElement("label");
    l.className = "site";
    l.innerHTML = `<input type="checkbox" ${off.includes(d) ? "" : "checked"}><span></span>`;
    l.querySelector("span").textContent = d;
    l.querySelector("input").addEventListener("change", async (e) => {
      const cur = (await chrome.storage.local.get("off")).off || [];
      const next = e.target.checked ? cur.filter((x) => x !== d) : [...new Set([...cur, d])];
      await chrome.storage.local.set({ off: next });
      await chrome.runtime.sendMessage({ type: "sites-changed" });
    });
    $("sites").appendChild(l);
  }
  $("extraNote").textContent = extra.length ? T.extra : "";
}

$("addSite").addEventListener("click", async () => {
  const d = clean($("newSite").value);
  if (!d) return;
  const { extra = [] } = await chrome.storage.local.get("extra");
  if (!extra.includes(d) && !DEFAULT_SITES.includes(d)) await chrome.storage.local.set({ extra: [...extra, d] });
  $("newSite").value = "";
  await chrome.runtime.sendMessage({ type: "sites-changed" });
  renderSites();
});

// The addresses WebRTC would hand a site: ask a public STUN server, as a page would.
async function webrtcAddresses() {
  const pc = new RTCPeerConnection({ iceServers: [{ urls: "stun:stun.l.google.com:19302" }] });
  const found = new Set();
  pc.createDataChannel("k");
  pc.onicecandidate = (e) => {
    const c = e.candidate && e.candidate.candidate;
    const m = c && / typ srflx/.test(c) && c.match(/ (\d+\.\d+\.\d+\.\d+|[0-9a-f:]+:[0-9a-f:]+) \d+ typ/i);
    if (m) found.add(m[1]);
  };
  await pc.setLocalDescription(await pc.createOffer());
  await new Promise((r) => setTimeout(r, 2500));
  pc.close();
  return [...found];
}

$("rtcCheck").addEventListener("click", async () => {
  $("rtcResult").textContent = T.checking;
  const ips = await webrtcAddresses().catch(() => []);
  await renderState();
  if (!ips.length) return ($("rtcResult").textContent = T.rtcNone);
  if (!S.ok) return ($("rtcResult").textContent = T.rtcUnknown + ips.join(", ") + T.rtcUnknown2);
  const leak = ips.find((ip) => ip !== S.ip && ip !== S.ip6);
  $("rtcResult").textContent = leak ? T.rtcLeak + leak + T.rtcLeak2 : T.rtcSame + ips.join(", ");
  $("rtcResult").style.color = leak ? "var(--bad)" : "";
});

const policy = chrome.privacy.network.webRTCIPHandlingPolicy;
async function renderFix() {
  const { value } = await policy.get({});
  const on = value === "disable_non_proxied_udp";
  $("rtcFix").textContent = on ? T.fixOff : T.fixOn;
  $("rtcFix").className = on ? "" : "pri";
  return on;
}
$("rtcFix").addEventListener("click", async () => {
  if (await renderFix()) await policy.clear({});
  else {
    await policy.set({ value: "disable_non_proxied_udp" });
    $("rtcResult").textContent = T.fixed;
    $("rtcResult").style.color = "";
  }
  renderFix();
});

$("hSites").textContent = T.sites;
$("rtcAbout").textContent = T.rtcAbout;
$("rtcCheck").textContent = T.check;
renderState();
renderSites();
renderFix();
setInterval(renderState, 1500);
