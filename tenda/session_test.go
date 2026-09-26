package tenda

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSessionPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	a, b := DefaultSessionPath("192.168.0.1"), DefaultSessionPath("router.lan:8080")
	if a != filepath.Join(dir, "tendactl-session-192.168.0.1") || b != filepath.Join(dir, "tendactl-session-router.lan_8080") {
		t.Fatalf("paths = %s, %s", a, b)
	}
	if got := filepath.Base(DefaultSessionPath("../../etc/x")); got != "tendactl-session-.._.._etc_x" {
		t.Fatalf("unsafe host not sanitised: %s", got)
	}

	t.Setenv("XDG_RUNTIME_DIR", "")
	if got := DefaultSessionPath("h"); got != filepath.Join(os.TempDir(), "tendactl-session-h") {
		t.Fatalf("fallback = %s", got)
	}
}

func TestSaveSessionTightensMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveSession(path, "tok"); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", st.Mode().Perm())
	}
	if loadSession(path) != "tok" {
		t.Fatalf("load = %q", loadSession(path))
	}
	if loadSession(filepath.Join(t.TempDir(), "missing")) != "" {
		t.Fatal("missing file must load as empty")
	}
}
