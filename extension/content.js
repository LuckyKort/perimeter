// On an AI site: asks the extension once a second whether to pause, tells the
// page script (main.js) to cut connections, and covers the tab while paused.
(() => {
  const ru = !navigator.language.startsWith("en");
  const T = ru
    ? { t: "Вкладка на паузе", b: "Perimeter остановил эту вкладку, пока IP не тот. Как только VPN снова заработает, она продолжит сама - ничего не закрывайте.", why: "Причина" }
    : { t: "This tab is paused", b: "Perimeter stopped this tab while the IP is wrong. Once the VPN is back it carries on by itself - don't close anything.", why: "Reason" };
  let paused = false;
  let host = null;

  function cover(on, reason) {
    if (!on) {
      host?.remove();
      host = null;
      return;
    }
    if (!host) {
      host = document.createElement("perimeter-pause");
      const root = host.attachShadow({ mode: "closed" });
      root.innerHTML = `<style>
        :host { all: initial; }
        .wrap { position: fixed; inset: 0; z-index: 2147483647; display: grid; place-items: center;
          background: rgba(12,14,20,.55); backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
          font: 15px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif; animation: in .25s cubic-bezier(.2,.8,.2,1) both; }
        @keyframes in { from { opacity: 0 } to { opacity: 1 } }
        .card { width: min(420px, calc(100vw - 40px)); background: #14161D; color: #fff; border-radius: 22px; padding: 26px 26px 22px;
          box-shadow: 0 0 0 1px rgba(255,255,255,.08) inset, 0 30px 80px rgba(0,0,0,.5); }
        .top { display: flex; align-items: center; gap: 10px; font-weight: 700; font-size: 15px; }
        .dot { width: 10px; height: 10px; border-radius: 50%; background: #F5A524; box-shadow: 0 0 0 0 rgba(245,165,36,.6); animation: p 2s infinite; }
        @keyframes p { 70% { box-shadow: 0 0 0 10px rgba(245,165,36,0) } 100% { box-shadow: 0 0 0 0 rgba(245,165,36,0) } }
        h1 { font-size: 22px; margin: 16px 0 8px; letter-spacing: -.02em; }
        p { margin: 0; color: rgba(255,255,255,.72); }
        .why { margin-top: 14px; font-size: 13px; color: rgba(255,255,255,.5); }
        </style><div class="wrap"><div class="card"><div class="top"><span class="dot"></span>Perimeter</div>
        <h1>${T.t}</h1><p>${T.b}</p><div class="why"></div></div></div>`;
      host._why = root.querySelector(".why");
      (document.body || document.documentElement).appendChild(host);
    }
    host._why.textContent = reason ? `${T.why}: ${reason}` : "";
  }

  async function tick() {
    let s;
    try {
      s = await chrome.runtime.sendMessage({ type: "state" });
    } catch {
      clearInterval(timer); // the extension was updated or removed
      return;
    }
    const want = !!(s && s.ok && s.pause);
    if (want !== paused) {
      paused = want;
      window.postMessage({ source: "perimeter-extension", pause: paused }, "*");
    }
    cover(paused, s && s.reason);
  }
  const timer = setInterval(tick, 1000);
  tick();
})();
