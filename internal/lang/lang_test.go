package lang

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"eng":    "en",
		"ENG":    "en",
		"jpn":    "ja",
		"fre":    "fr",
		"fra":    "fr",
		"spa":    "es",
		"en":     "en",
		"EN":     "en",
		"es-419": "es-419",
		"pt-BR":  "pt-br",
		"und":          "",
		"undetermined": "",
		"":             "",
		"   ":          "",
		"xyz":          "xyz",
		"Japanese":     "ja",
		"english":      "en",
		"  Spanish  ":  "es",
		"Klingon":      "klingon",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPrimary(t *testing.T) {
	cases := map[string]string{
		"en-US": "en", "es-419": "es", "zh-Hans": "zh", "pt-BR": "pt",
		"ja": "ja", "jpn": "ja", "Japanese": "ja", "und": "", "": "", "xyz": "xyz",
	}
	for in, want := range cases {
		if got := Primary(in); got != want {
			t.Errorf("Primary(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	cases := map[string]string{
		"en-us": "English", "es-419": "Spanish", "ja": "Japanese",
		"zh-Hans": "Chinese", "jpn": "Japanese", "": "", "und": "",
		"xyz": "XYZ", // unknown primary falls back to upper-cased code
	}
	for in, want := range cases {
		if got := DisplayName(in); got != want {
			t.Errorf("DisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}
