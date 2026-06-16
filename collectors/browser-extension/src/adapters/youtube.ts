import { youtubeIdentity } from "../core/identity";
import type { Adapter, MediaDetails } from "./types";

function channelMeta(): Record<string, unknown> | undefined {
  const el = document.querySelector(
    "ytd-channel-name a, #channel-name a, #owner #channel-name a",
  );
  const channel = el?.textContent?.trim();
  return channel ? { channel } : undefined;
}

export const youtubeAdapter: Adapter = {
  matches: () => location.hostname.endsWith("youtube.com"),
  identity: () => youtubeIdentity(location.href),
  details: (video): MediaDetails => ({
    title: document.title.replace(/ - YouTube$/, "") || undefined,
    duration_seconds: Number.isFinite(video.duration) ? Math.round(video.duration) : undefined,
    url_or_path: location.href,
    kind: "video",
    meta: channelMeta(),
  }),
};
