// Runs in YouTube's MAIN world (the isolated content script can't see page JS).
// Reads player state — the selected audio track and the real playing video id —
// from the player API and republishes it onto <html> data attributes for the
// content script to read synchronously.
// Best-effort: YouTube's player internals are undocumented; failures are silent.
// The audio-shape parsing lives in core/ytaudio.ts (pure, unit-tested against
// real player payloads); this file is just the DOM/player glue.

import { selectedLanguage } from "./core/ytaudio";

let stateHooked = false;

function publish(): void {
  try {
    const player = document.getElementById("movie_player") as unknown as {
      getAudioTrack?: () => unknown;
      getPlayerResponse?: () => unknown;
      getVideoData?: () => { video_id?: string } | undefined;
      addEventListener?: (event: string, listener: () => void) => void;
    } | null;
    const ds = document.documentElement.dataset;

    // No player on the page => nothing is loaded; clear the id so a stale value
    // can't misattribute playback on a later page.
    if (!player) {
      delete ds.wtVideoId;
      return;
    }

    // Publish the real playing id. Only write on change so the content script's
    // attribute observer fires for genuine id changes, not every 2s poll. Keep a
    // prior id across transient empties (e.g. ads) rather than clearing it.
    const vid =
      typeof player.getVideoData === "function" ? player.getVideoData()?.video_id : undefined;
    if (vid && ds.wtVideoId !== vid) ds.wtVideoId = vid;

    // Refresh promptly on state changes instead of waiting for the 2s poll.
    if (!stateHooked && typeof player.addEventListener === "function") {
      player.addEventListener("onStateChange", publish);
      stateHooked = true;
    }

    if (typeof player.getAudioTrack !== "function") return;
    const response =
      typeof player.getPlayerResponse === "function" ? player.getPlayerResponse() : undefined;
    const { code, label } = selectedLanguage(player.getAudioTrack(), response);
    if (code) ds.wtAudioLang = code;
    else delete ds.wtAudioLang;
    if (label) ds.wtAudioLangLabel = label;
    else delete ds.wtAudioLangLabel;
  } catch {
    /* best-effort: leave any previous values in place */
  }
}

publish();
setInterval(publish, 2000);
document.addEventListener("yt-navigate-finished", publish);
