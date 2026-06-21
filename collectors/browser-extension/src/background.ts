import { withDefaults, type Config } from "./config";
import { enqueue } from "./core/queue";
import { flushOnce } from "./core/flush";
import type { WatchEvent } from "./core/event";

const CONFIG_KEY = "watchtrail_config";
const QUEUE_KEY = "watchtrail_queue";
const BATCH_MAX = 50;
const FLUSH_ALARM = "watchtrail-flush";

type BadgeState = "off" | "tracked" | "recording";
const tabBadge = new Map<number, BadgeState>();

function compileHostMatchers(): Array<(host: string) => boolean> {
  const m = chrome.runtime.getManifest();
  const patterns = (m.content_scripts ?? []).flatMap((cs) => cs.matches ?? []);
  return patterns.map((p) => {
    const hostMatch = p.match(/^[^:]+:\/\/([^/]+)/);
    if (!hostMatch) {
      console.warn("[watchtrail] unparseable content_script match pattern:", p);
      return () => false;
    }
    const host = hostMatch[1];
    if (host === "*") return () => true;
    if (host.startsWith("*.")) {
      const suffix = host.slice(2);
      return (h: string) => h === suffix || h.endsWith("." + suffix);
    }
    return (h: string) => h === host;
  });
}
const HOST_MATCHERS = compileHostMatchers();

function isTrackedUrl(url: string | undefined): boolean {
  if (!url) return false;
  try {
    const host = new URL(url).hostname;
    return HOST_MATCHERS.some((fn) => fn(host));
  } catch {
    return false;
  }
}

async function setBadge(tabId: number, state: BadgeState): Promise<void> {
  if (tabBadge.get(tabId) === state) return;
  tabBadge.set(tabId, state);
  if (state === "off") {
    await chrome.action.setBadgeText({ tabId, text: "" });
    return;
  }
  const color = state === "recording" ? "#f0a040" : "#3a8489";
  await Promise.all([
    chrome.action.setBadgeText({ tabId, text: "●" }),
    chrome.action.setBadgeBackgroundColor({ tabId, color }),
  ]);
}

async function refreshTabBadge(tabId: number, url?: string): Promise<void> {
  const cfg = await loadConfig();
  if (!cfg.enabled || !isTrackedUrl(url)) {
    if (tabBadge.get(tabId) !== "off") await setBadge(tabId, "off");
    return;
  }
  // Don't downgrade an active recording when only the URL changed; the event
  // stream is the authoritative signal for that transition.
  if (tabBadge.get(tabId) !== "recording") await setBadge(tabId, "tracked");
}

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

let configCache: Config | null = null;
async function loadConfig(): Promise<Config> {
  if (configCache) return configCache;
  const got = await chrome.storage.local.get(CONFIG_KEY);
  configCache = withDefaults(got[CONFIG_KEY] ?? {});
  return configCache;
}
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === "local" && CONFIG_KEY in changes) configCache = null;
});

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
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;
  try {
    const res = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify(batch),
    });
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
  await withLock(() =>
    flushOnce({ loadQueue, saveQueue, post, batchMax: BATCH_MAX }),
  );
}

async function applyBadgeFromEvent(
  tabId: number,
  evType: WatchEvent["type"],
): Promise<void> {
  // "seek" can fire while paused (scrubbing on a paused video still emits
  // "seeked"), so it isn't a reliable signal of active playback. Only the
  // playback-state transitions promote to recording.
  if (evType === "start" || evType === "resume" || evType === "progress") {
    await setBadge(tabId, "recording");
    return;
  }
  // The content script only runs on tracked hosts, so the tab is on a tracked
  // URL by construction — no need to re-fetch tab.url.
  const cfg = await loadConfig();
  await setBadge(tabId, cfg.enabled ? "tracked" : "off");
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg?.kind !== "watchtrail-event") return false;
  (async () => {
    try {
      const cfg = await loadConfig();
      if (!cfg.enabled) {
        sendResponse({ ok: false });
        return;
      }
      const event = msg.event as WatchEvent;
      await withLock(async () => {
        const { queue } = enqueue(await loadQueue(), event);
        await saveQueue(queue);
      });
      const tabId = sender.tab?.id;
      if (tabId !== undefined) void applyBadgeFromEvent(tabId, event.type);
      void flush();
      sendResponse({ ok: true });
    } catch {
      sendResponse({ ok: false });
    }
  })();
  return true; // keep the message channel open for the async response
});

chrome.tabs.onActivated.addListener(async ({ tabId }) => {
  try {
    const tab = await chrome.tabs.get(tabId);
    await refreshTabBadge(tabId, tab.url);
  } catch {
    /* tab gone */
  }
});

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (info.url === undefined) return;
  void refreshTabBadge(tabId, tab.url);
});

chrome.tabs.onRemoved.addListener((tabId) => {
  tabBadge.delete(tabId);
});

chrome.alarms.create(FLUSH_ALARM, { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener((a) => {
  if (a.name === FLUSH_ALARM) void flush();
});
chrome.runtime.onStartup.addListener(() => void flush());
