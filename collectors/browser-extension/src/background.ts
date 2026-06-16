import { withDefaults, type Config } from "./config";
import { enqueue } from "./core/queue";
import { flushOnce } from "./core/flush";
import type { WatchEvent } from "./core/event";

const CONFIG_KEY = "watchtrail_config";
const QUEUE_KEY = "watchtrail_queue";
const BATCH_MAX = 50;
const FLUSH_ALARM = "watchtrail-flush";

// Serializes all queue load->mutate->save sequences. chrome.storage.local has no
// locking, so without this two interleaving async handlers would clobber each
// other's write and drop buffered events. Runs fn after any prior work settles
// (resolve or reject), so a thrown task never wedges the chain.
let lock: Promise<unknown> = Promise.resolve();
function withLock<T>(fn: () => Promise<T>): Promise<T> {
  const run = lock.then(fn, fn);
  lock = run.then(
    () => undefined,
    () => undefined,
  );
  return run;
}

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
  // The whole drain->post->ack->save runs under the lock so a concurrent enqueue
  // cannot invalidate the snapshot flushOnce acks against.
  await withLock(() => flushOnce({ loadQueue, saveQueue, post, batchMax: BATCH_MAX }));
}

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg?.kind !== "watchtrail-event") return false;
  (async () => {
    try {
      const cfg = await loadConfig();
      if (!cfg.enabled) {
        sendResponse({ ok: false });
        return;
      }
      await withLock(async () => {
        const { queue } = enqueue(await loadQueue(), msg.event as WatchEvent);
        await saveQueue(queue);
      });
      void flush();
      sendResponse({ ok: true });
    } catch {
      sendResponse({ ok: false });
    }
  })();
  return true; // keep the message channel open for the async response
});

chrome.alarms.create(FLUSH_ALARM, { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener((a) => {
  if (a.name === FLUSH_ALARM) void flush();
});
chrome.runtime.onStartup.addListener(() => void flush());
