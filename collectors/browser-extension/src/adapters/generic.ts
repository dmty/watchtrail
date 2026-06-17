import { genericIdentity } from "../core/identity";
import { selectedAudioTrack, audioMeta } from "../core/audiolang";
import type { Adapter, MediaDetails } from "./types";

export const genericAdapter: Adapter = {
  matches: () => true,
  identity: () => genericIdentity(location.href),
  details: (video): MediaDetails => {
    // audioTracks is non-standard and disabled in most Chrome builds — usually
    // empty here, which is fine: selectedAudioTrack returns {} and language is
    // simply omitted.
    const sel = selectedAudioTrack((video as unknown as { audioTracks?: ArrayLike<unknown> }).audioTracks as never);
    return {
      title: document.title || undefined,
      duration_seconds: Number.isFinite(video.duration) ? Math.round(video.duration) : undefined,
      url_or_path: location.href,
      kind: "video",
      language: sel.language,
      meta: audioMeta(sel),
    };
  },
};
