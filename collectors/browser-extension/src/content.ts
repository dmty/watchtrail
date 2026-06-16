import { youtubeAdapter } from "./adapters/youtube";
import { genericAdapter } from "./adapters/generic";
import { newSession, step, type NativeKind, type SessionState } from "./core/session";
import { buildEvent } from "./core/event";
import { withDefaults } from "./config";
import type { Adapter } from "./adapters/types";

const CONFIG_KEY = "watchtrail_config";

function pickAdapter(): Adapter {
  return youtubeAdapter.matches() ? youtubeAdapter : genericAdapter;
}

async function isEnabled(): Promise<boolean> {
  const got = await chrome.storage.local.get(CONFIG_KEY);
  return withDefaults(got[CONFIG_KEY] ?? {}).enabled;
}

async function main(): Promise<void> {
  if (!(await isEnabled())) return;

  const adapter = pickAdapter();
  let state: SessionState = newSession();
  let currentId: string | null = adapter.identity()?.external_id ?? null;

  function emit(native: NativeKind, video: HTMLVideoElement): void {
    const idy = adapter.identity();
    if (!idy) return;
    if (idy.external_id !== currentId) {
      state = newSession();
      currentId = idy.external_id;
    }
    const r = step(state, native, Date.now());
    state = r.state;
    if (!r.type) return;
    const d = adapter.details(video);
    const event = buildEvent({
      type: r.type,
      identity: idy,
      position_seconds: Math.round(video.currentTime),
      title: d.title,
      duration_seconds: d.duration_seconds,
      url_or_path: d.url_or_path,
      kind: d.kind,
      meta: d.meta,
    });
    void chrome.runtime.sendMessage({ kind: "watchtrail-event", event });
  }

  const BOUND = "__wt_bound";
  function bind(video: HTMLVideoElement): void {
    if ((video as unknown as Record<string, unknown>)[BOUND]) return;
    (video as unknown as Record<string, unknown>)[BOUND] = true;
    video.addEventListener("play", () => emit("play", video));
    video.addEventListener("pause", () => emit("pause", video));
    video.addEventListener("seeked", () => emit("seeked", video));
    video.addEventListener("ended", () => emit("ended", video));
    video.addEventListener("timeupdate", () => emit("timeupdate", video));
  }

  function scan(): void {
    document.querySelectorAll("video").forEach((v) => bind(v as HTMLVideoElement));
  }

  function stopActive(): void {
    const v = document.querySelector("video") as HTMLVideoElement | null;
    if (v) emit("hide", v);
  }

  scan();
  new MutationObserver(scan).observe(document.documentElement, { childList: true, subtree: true });
  window.addEventListener("pagehide", stopActive);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") stopActive();
  });
}

void main();
