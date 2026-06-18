// Runs in YouTube's MAIN world (the isolated content script can't see page JS).
// Reads the selected audio track from the player API and republishes it onto
// <html> data attributes for the content script to read synchronously.
// Best-effort: YouTube's player internals are undocumented; failures are silent.
// The shape parsing lives in core/ytaudio.ts (pure, unit-tested against real
// player payloads); this file is just the DOM/player glue.

import { selectedLanguage } from "./core/ytaudio";

function publish(): void {
  try {
    const player = document.getElementById("movie_player") as unknown as {
      getAudioTrack?: () => unknown;
      getPlayerResponse?: () => unknown;
    } | null;
    const ds = document.documentElement.dataset;
    if (!player || typeof player.getAudioTrack !== "function") return;
    const response = typeof player.getPlayerResponse === "function" ? player.getPlayerResponse() : undefined;
    const { code, label } = selectedLanguage(player.getAudioTrack(), response);
    if (code) ds.wtAudioLang = code;
    else delete ds.wtAudioLang;
    if (label) ds.wtAudioLangLabel = label;
    else delete ds.wtAudioLangLabel;
  } catch {
    /* best-effort: leave any previous value in place */
  }
}

publish();
setInterval(publish, 2000);
document.addEventListener("yt-navigate-finished", publish);
