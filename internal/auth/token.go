// Package auth owns the shared-secret cookie auth used by the dashboard and
// read API. The same 32-byte secret lives on disk as a hex string (auth.key,
// mode 0600), is sent in the "wt_auth" cookie, and is the value of the
// ?setup= magic-link query parameter. Rotating the file invalidates every
// issued cookie.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const keyBytes = 32

// KeyPath returns the on-disk location of the shared secret inside dataDir.
func KeyPath(dataDir string) string {
	return filepath.Join(dataDir, "auth.key")
}

// LoadOrCreateKey returns the 32-byte secret stored at KeyPath(dataDir),
// generating it on first call. created is true only when the file was just
// written. dataDir must already exist.
func LoadOrCreateKey(dataDir string) (key []byte, created bool, err error) {
	path := KeyPath(dataDir)
	raw, err := os.ReadFile(path)
	if err == nil {
		decoded, err := hex.DecodeString(strings.TrimSpace(string(raw)))
		if err != nil || len(decoded) != keyBytes {
			return nil, false, fmt.Errorf("auth.key: malformed (expected %d hex bytes)", keyBytes)
		}
		return decoded, false, nil
	}
	if !os.IsNotExist(err) {
		return nil, false, err
	}
	key = make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, false, fmt.Errorf("auth.key: rand: %w", err)
	}
	keyHex := hex.EncodeToString(key) + "\n"
	if err := os.WriteFile(path, []byte(keyHex), 0o600); err != nil {
		return nil, false, fmt.Errorf("auth.key: write: %w", err)
	}
	return key, true, nil
}

// HexKey returns the lowercase hex form used in cookies and URLs.
func HexKey(key []byte) string { return hex.EncodeToString(key) }
