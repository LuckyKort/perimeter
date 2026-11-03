// A page load to an AI site while paused lands here; it goes on to the site
// as soon as Kordon says the IP is right again (or Kordon isn't running).
const ru = !navigator.language.startsWith("en");
let target = location.hash.slice(1);
try { target = decodeURIComponent(target); } catch { /* already plain */ }
document.getElementById("t").textContent = ru ? "Сайт на паузе" : "This site is paused";
document.getElementById("b").textContent = ru
  ? "IP сейчас не тот, поэтому Kordon не пускает сюда. Страница откроется сама, как только VPN снова заработает."
  : "The IP isn't right now, so Kordon holds this site back. The page opens by itself once the VPN is back.";
document.getElementById("u").textContent = target;

const go = () => /^https?:\/\//.test(target) && location.replace(target);
async function tick() {
  const s = await chrome.runtime.sendMessage({ type: "state" }).catch(() => null);
  if (!s || !s.ok || !s.pause) return go();
  document.getElementById("why").textContent = s.reason ? (ru ? "Причина: " : "Reason: ") + s.reason : "";
}
setInterval(tick, 1000);
tick();
