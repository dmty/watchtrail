// Package lang normalizes collector-supplied audio-track language codes to a
// consistent lower-cased BCP-47 primary subtag, so the store can group and
// filter across sources that report ISO 639-2 (VLC) or BCP-47 (browser).
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

// Normalize returns a lower-cased BCP-47 primary subtag for code, preserving any
// region subtag. Empty, whitespace, and "und" return "".
func Normalize(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	if c == "" || c == "und" {
		return ""
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
