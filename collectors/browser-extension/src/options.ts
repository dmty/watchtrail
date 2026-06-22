import { withDefaults, type Config } from "./config";
import { unregisterGeneric } from "./registration";
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

function renderSites(cfg: Config): void {
  const host = el<HTMLDivElement>("sites");
  if (cfg.allowlist.length === 0) {
    host.innerHTML = `<div class="empty">No sites tracked yet.</div>`;
    return;
  }
  const rows = cfg.allowlist
    .map(
      (h) => `<tr><td class="host">${h}</td><td style="text-align:right"><button class="btn danger" data-host="${h}">Stop</button></td></tr>`,
    )
    .join("");
  host.innerHTML = `<table class="ledger"><thead><tr><th>Host</th><th></th></tr></thead><tbody>${rows}</tbody></table>`;
  host.querySelectorAll<HTMLButtonElement>("button[data-host]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const h = btn.dataset.host!;
      const c = await load();
      c.allowlist = c.allowlist.filter((x) => x !== h);
      await save(c);
      await unregisterGeneric(h);
      renderSites(c);
    });
  });
}

async function init(): Promise<void> {
  const cfg = await load();
  el<HTMLInputElement>("coreUrl").value = cfg.coreUrl;
  el<HTMLInputElement>("token").value = cfg.token;
  el<HTMLInputElement>("enabled").checked = cfg.enabled;
  renderSites(cfg);

  const manifest = chrome.runtime.getManifest();
  el("footer").textContent = `WatchTrail extension · v${manifest.version}`;

  el("closeLink").addEventListener("click", (e) => {
    e.preventDefault();
    window.close();
  });

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
    el("testResult").textContent = "Saved.";
  });

  el("test").addEventListener("click", async () => {
    const c = await load();
    el("testResult").textContent = "Testing…";
    const r = await probeCore(c.coreUrl, c.token);
    if (r.state === "reachable") el("testResult").textContent = `Reachable (HTTP ${r.status}).`;
    else if (r.state === "not-configured") el("testResult").textContent = "Core URL not set.";
    else el("testResult").textContent = `Unreachable: ${r.reason}`;
  });
}

void init();
