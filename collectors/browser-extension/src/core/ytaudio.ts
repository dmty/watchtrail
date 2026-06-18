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
