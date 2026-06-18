// internal/thumb/ffmpeg_test.go
package thumb

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestFrameArgs(t *testing.T) {
	args := strings.Join(frameArgs("/m/v.mkv", 42), " ")
	for _, want := range []string{"-ss 42", "-i /m/v.mkv", "-frames:v 1", "thumbnail,scale=480:-1", "pipe:1"} {
		if !strings.Contains(args, want) {
			t.Errorf("frameArgs missing %q in %q", want, args)
		}
	}
}

func TestCoverArgs(t *testing.T) {
	args := strings.Join(coverArgs("/m/v.mp4"), " ")
	for _, want := range []string{"-i /m/v.mp4", "-frames:v 1", "pipe:1"} {
		if !strings.Contains(args, want) {
			t.Errorf("coverArgs missing %q in %q", want, args)
		}
	}
}

func TestFFmpegAvailableMatchesLookPath(t *testing.T) {
	_, lookErr := exec.LookPath("ffmpeg")
	if got := NewFFmpeg().Available(); got != (lookErr == nil) {
		t.Fatalf("Available()=%v, lookPath err=%v", got, lookErr)
	}
}

// Real extraction, skipped when ffmpeg is absent (CI safe).
func TestFFmpegExtractFrameReal(t *testing.T) {
	f := NewFFmpeg()
	if !f.Available() {
		t.Skip("ffmpeg not installed")
	}
	// Generate a 1s test pattern to a temp file, then extract a frame.
	dir := t.TempDir()
	src := dir + "/src.mp4"
	gen := exec.Command(f.path, "-nostdin", "-y", "-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=5", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Skipf("could not synthesize test video: %v (%s)", err, out)
	}
	data, err := f.ExtractFrame(context.Background(), src, 0)
	if err != nil || len(data) == 0 {
		t.Fatalf("ExtractFrame: err=%v len=%d", err, len(data))
	}
}
