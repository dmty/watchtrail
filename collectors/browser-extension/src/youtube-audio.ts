// Runs in YouTube's MAIN world (the isolated content script can't see page JS).
// Reads the selected audio track from the player API and republishes it onto
// <html> data attributes for the content script to read synchronously.
// Best-effort: YouTube's player internals are undocumented; failures are silent.

interface YtAudioTrack {
  id?: string;
  name?: { simpleText?: string; runs?: Array<{ text?: string }> };
}

function trackLabel(t: YtAudioTrack | undefined): string | undefined {
  return t?.name?.simpleText ?? t?.name?.runs?.[0]?.text ?? undefined;
}

// YouTube encodes the language in the track id, e.g. "lang=es-419;..." or a
// name string. Pull the first BCP-47-looking token if present.
function trackCode(t: YtAudioTrack | undefined): string | undefined {
  const id = t?.id ?? "";
  const m = /lang=([A-Za-z]{2,3}(?:-[A-Za-z0-9]+)?)/.exec(id);
  return m ? m[1] : undefined;
}

function publish(): void {
  try {
    const player = document.getElementById("movie_player") as unknown as {
      getAudioTrack?: () => YtAudioTrack;
    } | null;
    const ds = document.documentElement.dataset;
    const track = player?.getAudioTrack?.();
    const code = trackCode(track);
    const label = trackLabel(track);
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
