import { withDefaults, type Config } from "./config";
import { enqueue } from "./core/queue";
import { flushOnce } from "./core/flush";
import type { WatchEvent } from "./core/event";

const CONFIG_KEY = "watchtrail_config";
const QUEUE_KEY = "watchtrail_queue";
const BATCH_MAX = 50;
const FLUSH_ALARM = "watchtrail-flush";

async function loadConfig(): Promise<Config> {
  const got = await chrome.storage.local.get(CONFIG_KEY);
  return withDefaults(got[CONFIG_KEY] ?? {});
}

async function loadQueue(): Promise<WatchEvent[]> {
  const got = await chrome.storage.local.get(QUEUE_KEY);
  return (got[QUEUE_KEY] as WatchEvent[]) ?? [];
}

async function saveQueue(q: WatchEvent[]): Promise<void> {
  await chrome.storage.local.set({ [QUEUE_KEY]: q });
}

async function post(batch: WatchEvent[]): Promise<boolean> {
  const cfg = await loadConfig();
  const url = cfg.coreUrl.replace(/\/$/, "") + "/ingest";
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  try {
    const res = await fetch(url, { method: "POST", headers, body: JSON.stringify(batch) });
    return res.ok;
  } catch {
    return false;
  }
}

async function flush(): Promise<void> {
  const cfg = await loadConfig();
  if (!cfg.enabled) return;
  await flushOnce({ loadQueue, saveQueue, post, batchMax: BATCH_MAX });
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg?.kind !== "watchtrail-event") return false;
  (async () => {
    const cfg = await loadConfig();
    if (!cfg.enabled) {
      sendResponse({ ok: false });
      return;
    }
    const { queue } = enqueue(await loadQueue(), msg.event as WatchEvent);
    await saveQueue(queue);
    void flush();
    sendResponse({ ok: true });
  })();
  return true; // keep the message channel open for the async response
});

chrome.alarms.create(FLUSH_ALARM, { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener((a) => {
  if (a.name === FLUSH_ALARM) void flush();
});
chrome.runtime.onStartup.addListener(() => void flush());
