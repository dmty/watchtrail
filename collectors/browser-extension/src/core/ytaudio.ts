// Pure parser for YouTube's getAudioTrack() return value. Kept DOM-free so it
// is unit-testable; the page-world script (youtube-audio.ts) feeds it the raw
// object. The audio-track descriptor is nested under a MINIFIED key that
// changes between YouTube builds, so we locate it by shape, not key name. Its
// `id` is a language code with a trailing ".N" build suffix (e.g. "en-US.4");
// the descriptor's top-level sibling `.id` is an opaque blob and unusable.

export interface ParsedAudio {
  code?: string;
  label?: string;
}

interface TrackDescriptor {
  name?: string;
  id?: string;
  isDefault?: boolean;
  isAutoDubbed?: boolean;
}

function descriptor(track: unknown): TrackDescriptor | undefined {
  if (!track || typeof track !== "object") return undefined;
  for (const v of Object.values(track as Record<string, unknown>)) {
    if (
      v && typeof v === "object" && !Array.isArray(v) &&
      typeof (v as TrackDescriptor).id === "string" &&
      "isDefault" in v && "isAutoDubbed" in v
    ) {
      return v as TrackDescriptor;
    }
  }
  return undefined;
}

// parseAudioTrack returns the selected audio track's language code (suffix
// stripped) and display label. Single-audio videos report "und"/"Default",
// which yields {} (no meaningful audio-language to record).
export function parseAudioTrack(track: unknown): ParsedAudio {
  const d = descriptor(track);
  if (!d || typeof d.id !== "string") return {};
  const code = d.id.replace(/\.\d+$/, "");
  if (!code || code.toLowerCase() === "und") return {};
  const out: ParsedAudio = { code };
  if (d.name && d.name !== "Default") out.label = d.name;
  return out;
}

interface CaptionTrack {
  languageCode?: string;
  kind?: string;
}

function captionTracks(playerResponse: unknown): CaptionTrack[] {
  const tracks = (playerResponse as {
    captions?: { playerCaptionsTracklistRenderer?: { captionTracks?: unknown } };
  })?.captions?.playerCaptionsTracklistRenderer?.captionTracks;
  return Array.isArray(tracks) ? (tracks as CaptionTrack[]) : [];
}

// asrLanguage returns the spoken language of a single-audio video via its
// auto-generated (kind:"asr") caption track, which YouTube transcribes from the
// audio. Translation caption tracks (kind:"") are ignored — they are not the
// spoken language. No code (no asr track) yields {}.
export function asrLanguage(playerResponse: unknown): ParsedAudio {
  for (const t of captionTracks(playerResponse)) {
    if (t && t.kind === "asr" && typeof t.languageCode === "string" && t.languageCode) {
      return { code: t.languageCode };
    }
  }
  return {};
}

// selectedLanguage resolves the language the viewer is hearing: the selected
// audio track when the video is multi-audio, otherwise the asr-caption spoken
// language for single-audio videos.
export function selectedLanguage(audioTrack: unknown, playerResponse: unknown): ParsedAudio {
  const track = parseAudioTrack(audioTrack);
  if (track.code) return track;
  return asrLanguage(playerResponse);
}
