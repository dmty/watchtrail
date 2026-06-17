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
		"und":    "",
		"":       "",
		"   ":    "",
		"xyz":    "xyz",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
