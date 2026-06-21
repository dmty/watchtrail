import { youtubeAdapter } from "./adapters/youtube";
import { genericAdapter } from "./adapters/generic";
import { newSession, step, switchMedia, type NativeKind, type SessionState } from "./core/session";
import { buildEvent, type EventType, type WatchEvent } from "./core/event";
import type { Identity } from "./core/identity";
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
  let current: Identity | null = adapter.identity();
  let lastPosition = 0;

  // After an extension reload/update, content scripts in already-open tabs are
  // orphaned: chrome.runtime is gone and sendMessage throws. Guard so a stale
  // tab fails quietly instead of spamming uncaught errors.
  function dispatch(event: WatchEvent): void {
    if (!chrome.runtime?.id) return;
    try {
      void chrome.runtime.sendMessage({ kind: "watchtrail-event", event }).catch(() => {});
    } catch {
      /* context invalidated between the check and the call */
    }
  }

  function emitClose(idy: Identity, position: number): void {
    dispatch(buildEvent({ type: "stop", identity: idy, position_seconds: position }));
  }

  function emitFull(type: EventType, idy: Identity, video: HTMLVideoElement): void {
    const d = adapter.details(video);
    dispatch(
      buildEvent({
        type,
        identity: idy,
        position_seconds: Math.round(video.currentTime),
        title: d.title,
        duration_seconds: d.duration_seconds,
        url_or_path: d.url_or_path,
        kind: d.kind,
        language: d.language,
        meta: d.meta,
      }),
    );
  }

  function emit(native: NativeKind, video: HTMLVideoElement): void {
    const idy = adapter.identity();
    if (!idy) return;
    if (!current || idy.external_id !== current.external_id) {
      const t = switchMedia(state, current?.external_id ?? null, idy.external_id, Date.now());
      if (t.close && current) emitClose(current, lastPosition);
      state = t.state;
      current = idy;
    }
    lastPosition = Math.round(video.currentTime);
    const r = step(state, native, Date.now());
    state = r.state;
    if (!r.type) return;
    emitFull(r.type, idy, video);
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
    // The content script attaches at document_idle, but YouTube often starts
    // playback before that — the native "play" already fired and was missed.
    // Bootstrap from the current state so the session still starts.
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

  // Emit a synthetic "play" for the currently-playing element when no native
  // "play" reaches us — the tab regained focus, or YouTube swapped the playing
  // video id without a fresh play (miniplayer start / expand).
  function bootstrapPlaying(): void {
    const v = document.querySelector("video") as HTMLVideoElement | null;
    if (v && !v.paused && !v.ended && v.readyState >= 2) emit("play", v);
  }

  scan();
  new MutationObserver(scan).observe(document.documentElement, { childList: true, subtree: true });
  window.addEventListener("pagehide", stopActive);
  // YouTube reuses one <video> element across watch/miniplayer/fullscreen. When a
  // video starts in the miniplayer (or is expanded back to the watch page) the
  // element keeps playing, so no native "play" fires and the URL no longer
  // identifies it. The probe publishes the real id; when it changes, bootstrap a
  // play for the still-playing element so the session starts/switches.
  if (youtubeAdapter.matches()) {
    new MutationObserver(bootstrapPlaying).observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-wt-video-id"],
    });
  }
  document.addEventListener("visibilitychange", () => {
    // Don't treat backgrounding as a stop: the video usually keeps playing, and
    // Chrome simply throttles the tab so timeupdate stops firing — tracking
    // stalls rather than ends. When the tab becomes visible again, rescan and
    // bootstrap any still-playing video so progress resumes. The long idle gap
    // while hidden exceeds the sessionizer's threshold, so it isn't counted as
    // watched time.
    if (document.visibilityState !== "visible") return;
    scan();
    bootstrapPlaying();
  });
}

void main();
