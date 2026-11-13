// Asks the Perimeter app whether the VPN/IP is right, and pauses AI sites when it
// isn't: network rules block their requests and redirect new page loads to a
// "paused" page; the content scripts cut open connections and cover the tab.
// If Perimeter isn't running, nothing is blocked (and the icon goes grey).
import { enabledSites } from "./sites.js";

const PERIMETER = "http://127.0.0.1:47843/ext/v1/state";
const RULE_IDS = [1, 2, 3];
let cache = { at: 0, state: null };
let applied = null; // the pause state the rules are in now
let inflight = null;

async function fetchState() {
  try {
    const r = await fetch(PERIMETER, { cache: "no-store" });
    if (!r.ok) throw new Error(r.statusText);
    return { ok: true, ...(await r.json()) };
  } catch {
    return { ok: false, pause: false };
  }
}

export async function state() {
  if (Date.now() - cache.at < 700 && cache.state) return cache.state;
  if (!inflight) {
    inflight = (async () => {
      try {
        const s = await fetchState();
        cache = { at: Date.now(), state: s };
        // The tabs pause on their own from the state; a failure in the rules must not stop that.
        await apply(s).catch((e) => console.warn("Perimeter: rules", e.message));
        return s;
      } finally {
        inflight = null;
      }
    })();
  }
  return inflight;
}

async function apply(s) {
  const pause = !!(s.ok && s.pause);
  const sites = await enabledSites();
  const key = pause + "|" + sites.join(",");
  if (key !== applied) {
    const addRules = [];
    if (pause && sites.length) {
      const blocked = chrome.runtime.getURL("blocked.html");
      addRules.push(
        // New page loads go to the "paused" page, which returns once the VPN is back.
        {
          id: 1, priority: 3,
          action: { type: "redirect", redirect: { regexSubstitution: blocked + "#\\0" } },
          condition: { regexFilter: "^https?://.*", requestDomains: sites, resourceTypes: ["main_frame"] },
        },
        // Every other request to the sites...
        { id: 2, priority: 2, action: { type: "block" }, condition: { requestDomains: sites, excludedResourceTypes: ["main_frame"] } },
        // ...and everything the sites' pages load from elsewhere.
        { id: 3, priority: 1, action: { type: "block" }, condition: { initiatorDomains: sites, excludedResourceTypes: ["main_frame"] } },
      );
    }
    await chrome.declarativeNetRequest.updateDynamicRules({ removeRuleIds: RULE_IDS, addRules });
    applied = key; // only once the rules are really in
  }
  const badge = !s.ok ? ["!", "#8B93A1"] : s.pause ? ["II", "#F5A524"] : ["", "#22C55E"];
  await chrome.action.setBadgeText({ text: badge[0] });
  await chrome.action.setBadgeBackgroundColor({ color: badge[1] });
  await chrome.action.setIcon({ path: s.ok ? { 16: "icons/icon16.png", 32: "icons/icon32.png" } : { 16: "icons/off16.png", 32: "icons/off32.png" } });
}

chrome.runtime.onMessage.addListener((msg, _sender, reply) => {
  if (msg?.type === "state") {
    state().then(reply);
    return true;
  }
  if (msg?.type === "sites-changed") {
    applied = null;
    cache.at = 0;
    state().then(reply);
    return true;
  }
});

// Keep the rules in step even with no AI tab open (tabs poll every second themselves).
chrome.alarms.create("poll", { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener(() => state());
chrome.runtime.onStartup.addListener(() => state());
chrome.runtime.onInstalled.addListener(() => state());
