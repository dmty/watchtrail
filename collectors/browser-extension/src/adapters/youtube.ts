import { youtubeIdentityFromState } from "../core/identity";
import { audioMeta, type SelectedAudio } from "../core/audiolang";
import type { Adapter, MediaDetails } from "./types";

// The page-world script (youtube-audio.ts) publishes the selected audio track
// here, since the isolated content script can't call YouTube's player API.
function selectedAudioFromDom(): SelectedAudio {
  const ds = document.documentElement.dataset;
  const out: SelectedAudio = {};
  if (ds.wtAudioLang && ds.wtAudioLang.toLowerCase() !== "und") out.language = ds.wtAudioLang;
  if (ds.wtAudioLangLabel) out.label = ds.wtAudioLangLabel;
  return out;
}

// Prefer the channel the probe read from the player API (data-wt-video-author):
// it names the actually-playing video's channel, which the page DOM gets wrong in
// the miniplayer. Fall back to the watch-page DOM.
function channelMeta(): Record<string, unknown> | undefined {
  const fromPlayer = document.documentElement.dataset.wtVideoAuthor?.trim();
  const el = document.querySelector(
    "ytd-channel-name a, #channel-name a, #owner #channel-name a",
  );
  const channel = fromPlayer || el?.textContent?.trim();
  return channel ? { channel } : undefined;
}

// Prefer the title the probe read from the player API (data-wt-video-title): it
// names the actually-playing video, so it stays correct in the miniplayer where
// the page DOM shows a different watch page. Fall back to the watch-page <h1>,
// then document.title — never reporting the bare "YouTube" placeholder (a
// transient SPA value), so the core stores no title rather than a wrong one.
function videoTitle(): string | undefined {
  const fromPlayer = document.documentElement.dataset.wtVideoTitle?.trim();
  if (fromPlayer) return fromPlayer;
  const el = document.querySelector(
    "h1.ytd-watch-metadata, #title h1 yt-formatted-string, h1.title yt-formatted-string",
  );
  const dom = el?.textContent?.trim();
  if (dom) return dom;
  const t = document.title.replace(/^\(\d+\)\s*/, "").replace(/\s*-\s*YouTube$/, "").trim();
  return t && t !== "YouTube" ? t : undefined;
}

export const youtubeAdapter: Adapter = {
  matches: () => location.hostname.endsWith("youtube.com"),
  identity: () => youtubeIdentityFromState(document.documentElement.dataset.wtVideoId, location.href),
  details: (video): MediaDetails => {
    const sel = selectedAudioFromDom();
    const meta = { ...channelMeta(), ...audioMeta(sel) };
    return {
      title: videoTitle(),
      duration_seconds: Number.isFinite(video.duration) ? Math.round(video.duration) : undefined,
      url_or_path: location.href,
      kind: "video",
      language: sel.language,
      meta: Object.keys(meta).length > 0 ? meta : undefined,
    };
  },
};
