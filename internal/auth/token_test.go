package auth

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrCreateKey_NewFile(t *testing.T) {
	dir := t.TempDir()
	key, created, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateKey: %v", err)
	}
	if !created {
		t.Errorf("created = false, want true on first call")
	}
	if len(key) != 32 {
		t.Errorf("len(key) = %d, want 32", len(key))
	}
	info, err := os.Stat(filepath.Join(dir, "auth.key"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" {
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("mode = %o, want 0600", got)
		}
	}
}

func TestLoadOrCreateKey_Idempotent(t *testing.T) {
	dir := t.TempDir()
	k1, _, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	k2, created, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if created {
		t.Errorf("second call set created = true")
	}
	if !bytes.Equal(k1, k2) {
		t.Errorf("key changed between calls")
	}
}

func TestLoadOrCreateKey_RejectsShortFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "auth.key"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadOrCreateKey(dir); err == nil {
		t.Errorf("expected error on malformed key file")
	}
}

func TestHexKey_RoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0xab}, 32)
	got := HexKey(key)
	decoded, err := hex.DecodeString(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(decoded, key) {
		t.Errorf("decode mismatch")
	}
	if len(got) != 64 {
		t.Errorf("len = %d, want 64", len(got))
	}
}

func TestLoadOrCreateKey_DirMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	_, _, err := LoadOrCreateKey(dir)
	if err == nil {
		t.Errorf("want error when DataDir missing")
	}
	// Should be the std fs error, not a wrapped panic-style.
	var pe *fs.PathError
	if !errors.As(err, &pe) {
		t.Errorf("err = %v, want *fs.PathError", err)
	}
}
