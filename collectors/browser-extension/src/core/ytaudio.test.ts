import { describe, it, expect } from "vitest";
import { parseAudioTrack, asrLanguage, selectedLanguage } from "./ytaudio";

// Fixtures captured from the live YouTube player API (getAudioTrack()). The
// audio-track descriptor is a nested object under a MINIFIED key that changes
// between YouTube builds (e.g. "mW"), so the parser must locate it by shape,
// not by key name. The language code is the descriptor's id with a trailing
// ".N" build-suffix; the top-level .id is an opaque blob.

describe("parseAudioTrack", () => {
  it("single-audio video reports no language (und/Default)", () => {
    const track = {
      id: "und",
      mW: { name: "Default", id: "und", isDefault: true, isAutoDubbed: false },
      captionTracks: [{ languageCode: "en" }],
    };
    expect(parseAudioTrack(track)).toEqual({});
  });

  it("multi-audio: extracts the selected track's code and label", () => {
    const track = {
      id: "251;ChEKBWFjb250EghvcmlnaW5hbA",
      mW: { name: "English (US) original", id: "en-US.4", isDefault: true, isAutoDubbed: false },
    };
    expect(parseAudioTrack(track)).toEqual({ code: "en-US", label: "English (US) original" });
  });

  it("strips the .N build suffix from the code", () => {
    const track = { id: "251;x", mW: { name: "Arabic", id: "ar.3", isDefault: false, isAutoDubbed: false } };
    expect(parseAudioTrack(track)).toEqual({ code: "ar", label: "Arabic" });
  });

  it("keeps script/region subtags (zh-Hans)", () => {
    const track = { id: "251;y", mW: { name: "Chinese (Simplified)", id: "zh-Hans.3", isDefault: false, isAutoDubbed: false } };
    expect(parseAudioTrack(track)).toEqual({ code: "zh-Hans", label: "Chinese (Simplified)" });
  });

  it("finds the descriptor regardless of the minified key name", () => {
    const track = { id: "251;z", xQ7: { name: "Spanish (Latin America)", id: "es-419.3", isDefault: false, isAutoDubbed: false } };
    expect(parseAudioTrack(track)).toEqual({ code: "es-419", label: "Spanish (Latin America)" });
  });

  it("returns {} for null / non-object / shapeless input", () => {
    expect(parseAudioTrack(null)).toEqual({});
    expect(parseAudioTrack(undefined)).toEqual({});
    expect(parseAudioTrack("x")).toEqual({});
    expect(parseAudioTrack({ id: "251;q" })).toEqual({});
  });
});

// Real player-response shape: captionTracks mixes the auto-speech-recognition
// track (kind:"asr", transcribed FROM the audio => spoken language) with
// translation options (kind:""). Only the asr track reflects the audio.
const playerResponse = {
  captions: {
    playerCaptionsTracklistRenderer: {
      captionTracks: [
        { languageCode: "en", languageName: "English", kind: "", vssId: ".en" },
        { languageCode: "en", languageName: "English (auto-generated)", kind: "asr", vssId: "a.en" },
        { languageCode: "de-DE", languageName: "German (Germany)", kind: "", vssId: ".de-DE" },
        { languageCode: "es-419", languageName: "Spanish (Latin America)", kind: "", vssId: ".es-419" },
      ],
    },
  },
};

describe("asrLanguage", () => {
  it("returns the auto-generated caption's code, ignoring translations", () => {
    expect(asrLanguage(playerResponse)).toEqual({ code: "en" });
  });

  it("returns {} when there is no asr track", () => {
    const noAsr = { captions: { playerCaptionsTracklistRenderer: { captionTracks: [
      { languageCode: "fr", kind: "" },
    ] } } };
    expect(asrLanguage(noAsr)).toEqual({});
  });

  it("returns {} for missing/shapeless input", () => {
    expect(asrLanguage(null)).toEqual({});
    expect(asrLanguage({})).toEqual({});
    expect(asrLanguage({ captions: {} })).toEqual({});
  });
});

describe("selectedLanguage", () => {
  it("prefers the selected audio track over asr", () => {
    const track = { id: "251;x", mW: { name: "Spanish (Latin America)", id: "es-419.3", isDefault: false, isAutoDubbed: false } };
    expect(selectedLanguage(track, playerResponse)).toEqual({ code: "es-419", label: "Spanish (Latin America)" });
  });

  it("falls back to asr when the audio track is und/single", () => {
    const track = { id: "und", mW: { name: "Default", id: "und", isDefault: true, isAutoDubbed: false } };
    expect(selectedLanguage(track, playerResponse)).toEqual({ code: "en" });
  });

  it("returns {} when neither yields a language", () => {
    expect(selectedLanguage(null, {})).toEqual({});
  });
});
