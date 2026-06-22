import { withDefaults, type Config } from "./config";
import { registerGeneric, unregisterGeneric } from "./registration";
import { probeCore } from "./probe";

const CONFIG_KEY = "watchtrail_config";

async function load(): Promise<Config> {
  const got = await chrome.storage.local.get(CONFIG_KEY);
  return withDefaults(got[CONFIG_KEY] ?? {});
}

async function save(cfg: Config): Promise<void> {
  await chrome.storage.local.set({ [CONFIG_KEY]: cfg });
}

function el<T extends HTMLElement>(id: string): T {
  return document.getElementById(id) as T;
}

function setStatus(text: string): void {
  el("status").textContent = text;
}

function renderPill(state: "ok" | "warn" | "err", text: string): void {
  const pill = el("pill");
  pill.classList.remove("ok", "warn", "err");
  pill.classList.add(state);
  el("pillText").textContent = text;
}

async function currentHost(): Promise<string | null> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.url) return null;
  try {
    return new URL(tab.url).hostname;
  } catch {
    return null;
  }
}

function renderTrackButtons(tracked: boolean): void {
  el<HTMLButtonElement>("track").hidden = tracked;
  el<HTMLButtonElement>("stop").hidden = !tracked;
}

async function init(): Promise<void> {
  const cfg = await load();
  el<HTMLInputElement>("enabled").checked = cfg.enabled;

  const host = await currentHost();
  el("host").textContent = host ?? "(no site)";
  renderTrackButtons(host !== null && cfg.allowlist.includes(host));

  void probeCore(cfg.coreUrl, cfg.token).then((r) => {
    if (r.state === "not-configured") renderPill("warn", "Not configured");
    else if (r.state === "reachable") renderPill("ok", "Core reachable");
    else renderPill("err", "Unreachable");
  });

  el("enabled").addEventListener("change", async () => {
    const c = await load();
    c.enabled = el<HTMLInputElement>("enabled").checked;
    await save(c);
  });

  el("track").addEventListener("click", async () => {
    if (!host) return;
    const granted = await chrome.permissions.request({
      origins: [`*://${host}/*`],
    });
    if (!granted) {
      setStatus("Permission denied.");
      return;
    }
    const c = await load();
    if (!c.allowlist.includes(host)) c.allowlist.push(host);
    await save(c);
    await registerGeneric(host);
    renderTrackButtons(true);
    setStatus(`Tracking ${host}.`);
  });

  el("stop").addEventListener("click", async () => {
    if (!host) return;
    const c = await load();
    c.allowlist = c.allowlist.filter((h) => h !== host);
    await save(c);
    await unregisterGeneric(host);
    renderTrackButtons(false);
    setStatus(`Stopped ${host}.`);
  });

  el("settings").addEventListener("click", (e) => {
    e.preventDefault();
    chrome.runtime.openOptionsPage();
  });
}

void init();
