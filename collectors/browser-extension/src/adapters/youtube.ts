import { youtubeIdentity } from "../core/identity";
import type { Adapter, MediaDetails } from "./types";

function channelMeta(): Record<string, unknown> | undefined {
  const el = document.querySelector(
    "ytd-channel-name a, #channel-name a, #owner #channel-name a",
  );
  const channel = el?.textContent?.trim();
  return channel ? { channel } : undefined;
}

// document.title is unreliable at playback start: YouTube's SPA leaves it as the
// bare "YouTube" (optionally prefixed with an unread count like "(3) ") for a
// moment before swapping in the real title. Prefer the player's metadata <h1>,
// which is populated earlier, and never report the "YouTube" placeholder so the
// core stores no title rather than a wrong one (a later event fills it in).
function videoTitle(): string | undefined {
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
  identity: () => youtubeIdentity(location.href),
  details: (video): MediaDetails => ({
    title: videoTitle(),
    duration_seconds: Number.isFinite(video.duration) ? Math.round(video.duration) : undefined,
    url_or_path: location.href,
    kind: "video",
    meta: channelMeta(),
  }),
};
