// AI sites the extension pauses when Kordon says so. A site matches itself and
// its subdomains. People can switch any off and add their own in the popup.
export const DEFAULT_SITES = [
  "claude.ai",
  "chatgpt.com",
  "chat.openai.com",
  "platform.openai.com",
  "sora.com",
  "gemini.google.com",
  "aistudio.google.com",
  "notebooklm.google.com",
  "perplexity.ai",
  "copilot.microsoft.com",
  "grok.com",
  "chat.deepseek.com",
  "chat.mistral.ai",
  "poe.com",
  "console.anthropic.com",
  "v0.dev",
  "lovable.dev",
  "bolt.new",
];

// clean turns what a person typed ("https://Claude.ai/new") into a domain.
export function clean(s) {
  s = String(s || "").trim().toLowerCase();
  try {
    if (s.includes("/")) s = new URL(s.includes("://") ? s : "https://" + s).hostname;
  } catch {
    return "";
  }
  s = s.replace(/^www\./, "");
  return /^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(s) ? s : "";
}

export async function enabledSites() {
  const { off = [], extra = [] } = await chrome.storage.local.get(["off", "extra"]);
  return [...DEFAULT_SITES, ...extra].filter((d) => !off.includes(d));
}
