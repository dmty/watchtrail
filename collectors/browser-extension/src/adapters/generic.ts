import { genericIdentity } from "../core/identity";
import type { Adapter, MediaDetails } from "./types";

export const genericAdapter: Adapter = {
  matches: () => true,
  identity: () => genericIdentity(location.href),
  details: (video): MediaDetails => ({
    title: document.title || undefined,
    duration_seconds: Number.isFinite(video.duration) ? Math.round(video.duration) : undefined,
    url_or_path: location.href,
    kind: "video",
  }),
};
