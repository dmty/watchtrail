// Package lang normalizes collector-supplied audio-track language values to a
// consistent lower-cased BCP-47 primary subtag, so the store can group and
// filter across sources that report ISO 639-2 codes (VLC), BCP-47 (browser),
// or full English language names (VLC's item:info() humanizes the code).
package lang

import "strings"

// iso6392to1 maps the common ISO 639-2 (B and T) codes collectors emit to their
// ISO 639-1 equivalents. Unlisted 3-letter codes are kept verbatim.
var iso6392to1 = map[string]string{
	"eng": "en", "jpn": "ja", "fre": "fr", "fra": "fr", "spa": "es",
	"ger": "de", "deu": "de", "ita": "it", "por": "pt", "rus": "ru",
	"chi": "zh", "zho": "zh", "kor": "ko", "ara": "ar", "hin": "hi",
	"nld": "nl", "dut": "nl", "pol": "pl", "tur": "tr", "swe": "sv",
	"vie": "vi", "tha": "th", "ind": "id", "ukr": "uk", "ces": "cs",
	"cze": "cs", "ron": "ro", "rum": "ro", "ell": "el", "gre": "el",
	"heb": "he", "dan": "da", "fin": "fi", "nor": "no", "hun": "hu",
}

// englishName maps the full English language names VLC surfaces (e.g. "Japanese")
// to their BCP-47 code. Unlisted names fall through and are kept verbatim.
var englishName = map[string]string{
	"japanese": "ja", "english": "en", "spanish": "es", "french": "fr",
	"german": "de", "italian": "it", "portuguese": "pt", "russian": "ru",
	"chinese": "zh", "korean": "ko", "arabic": "ar", "hindi": "hi",
	"dutch": "nl", "polish": "pl", "swedish": "sv", "turkish": "tr",
	"vietnamese": "vi", "thai": "th", "indonesian": "id", "ukrainian": "uk",
	"czech": "cs", "romanian": "ro", "greek": "el", "hebrew": "he",
	"danish": "da", "finnish": "fi", "norwegian": "no", "hungarian": "hu",
}

// Normalize returns a lower-cased BCP-47 primary subtag for value, preserving any
// region subtag. value may be an ISO 639-2/639-1 code or a full English language
// name. Empty, whitespace, "und", and "undetermined" return "".
func Normalize(value string) string {
	c := strings.ToLower(strings.TrimSpace(value))
	if c == "" || c == "und" || c == "undetermined" {
		return ""
	}
	if mapped, ok := englishName[c]; ok {
		return mapped
	}
	primary, rest, hasRest := strings.Cut(c, "-")
	if mapped, ok := iso6392to1[primary]; ok {
		primary = mapped
	}
	if hasRest {
		return primary + "-" + rest
	}
	return primary
}
