// Runs inside the AI site's own page (world: MAIN). Network rules stop new
// requests, but a reply already streaming or an open WebSocket would go on:
// this keeps track of them and cuts them when the tab pauses. While paused, new
// fetch / XHR / WebSocket / EventSource calls fail at once.
(() => {
  if (window.__perimeterPause) return;
  let paused = false;
  const controllers = new Set(); // AbortControllers of fetches
  const sockets = new Set(); // WebSockets and EventSources
  const xhrs = new Set();
  window.__perimeterPause = true;

  const oFetch = window.fetch;
  window.fetch = function (input, init) {
    if (paused) return Promise.reject(new TypeError("Perimeter: paused, the VPN is off"));
    const ac = new AbortController();
    const sig = (init && init.signal) || (input instanceof Request ? input.signal : null);
    if (sig) {
      if (sig.aborted) ac.abort(sig.reason);
      else sig.addEventListener("abort", () => ac.abort(sig.reason), { once: true });
    }
    controllers.add(ac);
    setTimeout(() => controllers.delete(ac), 30 * 60 * 1000); // a long stream at most
    return oFetch.call(this, input, { ...(init || {}), signal: ac.signal });
  };

  const track = (Orig) =>
    new Proxy(Orig, {
      construct(target, args, nt) {
        const s = Reflect.construct(target, args, nt);
        if (paused) {
          s.close();
        } else {
          sockets.add(s);
          s.addEventListener("close", () => sockets.delete(s));
          s.addEventListener("error", () => sockets.delete(s));
        }
        return s;
      },
    });
  if (window.WebSocket) window.WebSocket = track(window.WebSocket);
  if (window.EventSource) window.EventSource = track(window.EventSource);

  const oSend = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.send = function (...args) {
    if (paused) {
      this.abort();
      return;
    }
    xhrs.add(this);
    this.addEventListener("loadend", () => xhrs.delete(this));
    return oSend.apply(this, args);
  };

  if (navigator.sendBeacon) {
    const oBeacon = navigator.sendBeacon.bind(navigator);
    navigator.sendBeacon = (...args) => (paused ? false : oBeacon(...args));
  }

  window.addEventListener("message", (e) => {
    if (e.source !== window || !e.data || e.data.source !== "perimeter-extension") return;
    const was = paused;
    paused = !!e.data.pause;
    if (paused && !was) {
      controllers.forEach((c) => c.abort(new DOMException("Perimeter: paused, the VPN is off", "AbortError")));
      controllers.clear();
      sockets.forEach((s) => s.close());
      sockets.clear();
      xhrs.forEach((x) => x.abort());
      xhrs.clear();
    }
  });
})();
