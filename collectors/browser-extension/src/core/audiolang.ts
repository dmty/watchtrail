export interface AudioTrackLike {
  enabled?: boolean;
  language?: string;
  label?: string;
}

export interface SelectedAudio {
  language?: string;
  label?: string;
}

/** The enabled audio track (else the first), with und/empty language dropped. */
export function selectedAudioTrack(
  tracks: ArrayLike<AudioTrackLike> | null | undefined,
): SelectedAudio {
  if (!tracks || tracks.length === 0) return {};
  let chosen: AudioTrackLike | undefined;
  for (let i = 0; i < tracks.length; i++) {
    if (tracks[i].enabled) {
      chosen = tracks[i];
      break;
    }
  }
  if (!chosen) chosen = tracks[0];
  const out: SelectedAudio = {};
  if (chosen.language && chosen.language.toLowerCase() !== "und") out.language = chosen.language;
  if (chosen.label) out.label = chosen.label;
  return out;
}

/** Meta fields carrying the raw code + display label, or undefined if empty. */
export function audioMeta(sel: SelectedAudio): Record<string, unknown> | undefined {
  const meta: Record<string, unknown> = {};
  if (sel.label) meta.audio_language_label = sel.label;
  if (sel.language) meta.audio_language_raw = sel.language;
  return Object.keys(meta).length > 0 ? meta : undefined;
}
