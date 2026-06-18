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

// The selected audio track / spoken language lives in the page's own JS world,
// which the isolated content script cannot read. Inject a MAIN-world probe
// (bundled separately) that republishes it onto <html> data attributes for the
// adapter to read. Injecting it from here — instead of registering it as a
// standalone world:"MAIN" content script — ties its lifecycle to the content
// script that actually emits events, so the two can never fall out of sync (a
// separately-injected MAIN-world script was observed running for some tabs but
// not others while this script kept emitting).
function injectAudioProbe(): void {
  try {
    const s = document.createElement("script");
    s.src = chrome.runtime.getURL("dist/youtube-audio.js");
    s.onload = () => s.remove();
    (document.head || document.documentElement).appendChild(s);
  } catch {
    /* best effort: audio language is optional metadata */
  }
}

async function main(): Promise<void> {
  if (!(await isEnabled())) return;

  const adapter = pickAdapter();
  if (youtubeAdapter.matches()) injectAudioProbe();
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
      language: d.language,
      meta: d.meta,
    });
    // After an extension reload/update, content scripts in already-open tabs
    // are orphaned: chrome.runtime is gone and sendMessage throws "Extension
    // context invalidated". Guard so a stale tab fails quietly instead of
    // spamming uncaught errors.
    if (!chrome.runtime?.id) return;
    try {
      void chrome.runtime.sendMessage({ kind: "watchtrail-event", event }).catch(() => {});
    } catch {
      /* context invalidated between the check and the call */
    }
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
    // The content script attaches at document_idle, but YouTube (and other
    // autoplay sites) often start playback before that — so the native "play"
    // already fired and was missed. Without a "play", the session never starts
    // and every later event is dropped. Bootstrap from the current state.
    if (!video.paused && !video.ended && video.readyState >= 2) {
      emit("play", video);
    }
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
    // Don't treat backgrounding as a stop: the video usually keeps playing, and
    // Chrome simply throttles the tab so timeupdate stops firing — tracking
    // stalls rather than ends. When the tab becomes visible again, rescan and
    // bootstrap any still-playing video so progress resumes. The long idle gap
    // while hidden exceeds the sessionizer's threshold, so it isn't counted as
    // watched time.
    if (document.visibilityState !== "visible") return;
    scan();
    const v = document.querySelector("video") as HTMLVideoElement | null;
    if (v && !v.paused && !v.ended && v.readyState >= 2) emit("play", v);
  });
}

void main();
