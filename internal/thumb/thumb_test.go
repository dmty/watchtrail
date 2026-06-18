// internal/thumb/thumb_test.go
package thumb

import "testing"

func TestLocalPath(t *testing.T) {
	cases := []struct {
		in       string
		wantPath string
		wantOK   bool
	}{
		{"file:///Users/me/Movies/The%20Film.mkv", "/Users/me/Movies/The Film.mkv", true},
		{"file:///tmp/a.mp4", "/tmp/a.mp4", true},
		{"/tmp/bare.mp4", "/tmp/bare.mp4", true},
		{"url:abc", "", false},
		{"https://youtu.be/x", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := LocalPath(c.in)
		if ok != c.wantOK || got != c.wantPath {
			t.Errorf("LocalPath(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.wantPath, c.wantOK)
		}
	}
}

func TestContentTypeByExt(t *testing.T) {
	for in, want := range map[string]string{
		"poster.jpg": "image/jpeg", "P.JPEG": "image/jpeg",
		"folder.png": "image/png", "x.webp": "image/webp", "y.txt": "",
	} {
		if got := contentTypeByExt(in); got != want {
			t.Errorf("contentTypeByExt(%q) = %q, want %q", in, got, want)
		}
	}
}
