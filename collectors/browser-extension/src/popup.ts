import { withDefaults, type Config } from "./config";
import { registerGeneric, unregisterGeneric } from "./registration";

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

async function currentHost(): Promise<string | null> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.url) return null;
  try {
    return new URL(tab.url).hostname;
  } catch {
    return null;
  }
}

async function init(): Promise<void> {
  const cfg = await load();
  el<HTMLInputElement>("enabled").checked = cfg.enabled;
  el<HTMLInputElement>("coreUrl").value = cfg.coreUrl;
  el<HTMLInputElement>("token").value = cfg.token;

  const host = await currentHost();
  el("host").textContent = host ?? "(no site)";

  el("enabled").addEventListener("change", async () => {
    const c = await load();
    c.enabled = el<HTMLInputElement>("enabled").checked;
    await save(c);
  });

  el("save").addEventListener("click", async () => {
    const c = await load();
    c.coreUrl = el<HTMLInputElement>("coreUrl").value.trim();
    c.token = el<HTMLInputElement>("token").value;
    await save(c);
    setStatus("Saved.");
  });

  el("allow").addEventListener("click", async () => {
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
    setStatus(`Tracking ${host}.`);
  });

  el("disallow").addEventListener("click", async () => {
    if (!host) return;
    const c = await load();
    c.allowlist = c.allowlist.filter((h) => h !== host);
    await save(c);
    await unregisterGeneric(host);
    setStatus(`Stopped ${host}.`);
  });
}

void init();
