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
  host.replaceChildren();
  if (cfg.allowlist.length === 0) {
    const empty = document.createElement("div");
    empty.className = "empty";
    empty.textContent = "No sites tracked yet.";
    host.appendChild(empty);
    return;
  }
  const table = document.createElement("table");
  table.className = "ledger";
  const thead = document.createElement("thead");
  const headRow = document.createElement("tr");
  const th1 = document.createElement("th");
  th1.textContent = "Host";
  const th2 = document.createElement("th");
  headRow.append(th1, th2);
  thead.appendChild(headRow);
  const tbody = document.createElement("tbody");
  for (const h of cfg.allowlist) {
    const tr = document.createElement("tr");
    const tdHost = document.createElement("td");
    tdHost.className = "host";
    tdHost.textContent = h;
    const tdAct = document.createElement("td");
    tdAct.style.textAlign = "right";
    const btn = document.createElement("button");
    btn.className = "btn danger";
    btn.textContent = "Stop";
    btn.addEventListener("click", async () => {
      const c = await load();
      c.allowlist = c.allowlist.filter((x) => x !== h);
      await save(c);
      await unregisterGeneric(h);
      renderSites(c);
    });
    tdAct.appendChild(btn);
    tr.append(tdHost, tdAct);
    tbody.appendChild(tr);
  }
  table.append(thead, tbody);
  host.appendChild(table);
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
    if (r.state === "reachable")
      el("testResult").textContent = `Reachable (HTTP ${r.status}).`;
    else if (r.state === "not-configured")
      el("testResult").textContent = "Core URL not set.";
    else el("testResult").textContent = `Unreachable: ${r.reason}`;
  });
}

void init();
