import { describe, it, expect } from "vitest";
import { selectedAudioTrack, audioMeta } from "./audiolang";

describe("selectedAudioTrack", () => {
  it("picks the enabled track", () => {
    const tracks = [
      { enabled: false, language: "en", label: "English" },
      { enabled: true, language: "es-419", label: "Spanish (Latin America)" },
    ];
    expect(selectedAudioTrack(tracks)).toEqual({ language: "es-419", label: "Spanish (Latin America)" });
  });

  it("falls back to the first track when none is enabled", () => {
    expect(selectedAudioTrack([{ language: "ja", label: "Japanese" }])).toEqual({ language: "ja", label: "Japanese" });
  });

  it("drops und/empty language but keeps the label", () => {
    expect(selectedAudioTrack([{ enabled: true, language: "und", label: "Undetermined" }])).toEqual({ label: "Undetermined" });
  });

  it("returns {} for an empty or missing list", () => {
    expect(selectedAudioTrack([])).toEqual({});
    expect(selectedAudioTrack(null)).toEqual({});
    expect(selectedAudioTrack(undefined)).toEqual({});
  });
});

describe("audioMeta", () => {
  it("builds raw + label", () => {
    expect(audioMeta({ language: "es-419", label: "Spanish (Latin America)" })).toEqual({
      audio_language_raw: "es-419",
      audio_language_label: "Spanish (Latin America)",
    });
  });

  it("returns undefined when nothing is set", () => {
    expect(audioMeta({})).toBeUndefined();
  });
});
