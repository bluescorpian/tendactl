package tenda

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultSessionPath is where a host's session token is kept: the per-user
// runtime dir (0700, tmpfs, cleared at logout/reboot) so the token stays
// session-scoped without sharing a world-writable directory; os.TempDir() is
// the fallback where XDG_RUNTIME_DIR is unset.
func DefaultSessionPath(host string) string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			return r
		}
		return '_'
	}, host)
	return filepath.Join(dir, "tendactl-session-"+safe)
}

// loadSession returns the stored token, or "" when there is none.
func loadSession(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func saveSession(path, token string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	// OpenFile's mode only applies on creation; tighten a pre-existing file too.
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.WriteString(token); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
