// Runs in YouTube's MAIN world (the isolated content script can't see page JS).
// Reads player state — the selected audio track and the playing video's id,
// title, and channel — from the player API and republishes it onto <html> data
// attributes for the content script to read synchronously.
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
      getVideoData?: () => { video_id?: string; title?: string; author?: string } | undefined;
      addEventListener?: (event: string, listener: () => void) => void;
    } | null;
    const ds = document.documentElement.dataset;

    // No player on the page => nothing is loaded; clear the id so a stale value
    // can't misattribute playback on a later page.
    if (!player) {
      delete ds.wtVideoId;
      delete ds.wtVideoTitle;
      delete ds.wtVideoAuthor;
      return;
    }

    // Publish the real playing video's id, title, and channel. These come from the
    // player, not the page DOM, so they stay correct in the miniplayer (where the
    // page shows a different watch page or the feed). Only write on change so the
    // content script's attribute observer fires for genuine id changes, not every
    // 2s poll; keep prior values across transient empties (e.g. ads).
    const data =
      typeof player.getVideoData === "function" ? player.getVideoData() : undefined;
    const vid = data?.video_id;
    if (vid && ds.wtVideoId !== vid) ds.wtVideoId = vid;
    const title = data?.title;
    if (title && ds.wtVideoTitle !== title) ds.wtVideoTitle = title;
    const author = data?.author;
    if (author && ds.wtVideoAuthor !== author) ds.wtVideoAuthor = author;

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
