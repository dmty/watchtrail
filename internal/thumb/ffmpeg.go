package thumb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const ffmpegTimeout = 10 * time.Second

// FFmpeg is the real Extractor. path is empty when ffmpeg is not installed.
type FFmpeg struct{ path string }

func NewFFmpeg() *FFmpeg {
	p, _ := exec.LookPath("ffmpeg")
	return &FFmpeg{path: p}
}

func (f *FFmpeg) Available() bool { return f.path != "" }

func frameArgs(path string, atSeconds int) []string {
	return []string{
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-ss", strconv.Itoa(atSeconds), "-i", path,
		"-frames:v", "1", "-vf", "thumbnail,scale=480:-1",
		"-f", "image2", "-vcodec", "mjpeg", "pipe:1",
	}
}

func coverArgs(path string) []string {
	return []string{
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-i", path, "-an", "-map", "0:v",
		"-frames:v", "1", "-f", "image2", "pipe:1",
	}
}

// EmbeddedCover is best-effort: any failure or empty output reports "no cover"
// so resolution falls through to frame extraction.
func (f *FFmpeg) EmbeddedCover(ctx context.Context, path string) ([]byte, bool, error) {
	if f.path == "" {
		return nil, false, nil
	}
	out, err := f.run(ctx, coverArgs(path))
	if err != nil || len(out) == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func (f *FFmpeg) ExtractFrame(ctx context.Context, path string, atSeconds int) ([]byte, error) {
	if f.path == "" {
		return nil, errors.New("ffmpeg not available")
	}
	out, err := f.run(ctx, frameArgs(path, atSeconds))
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("ffmpeg produced no frame")
	}
	return out, nil
}

func (f *FFmpeg) run(ctx context.Context, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, ffmpegTimeout)
	defer cancel()
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, f.path, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, bytes.TrimSpace(errBuf.Bytes()))
	}
	return buf.Bytes(), nil
}
