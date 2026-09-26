package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestBackup(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	path := filepath.Join(t.TempDir(), "RouterCfm.cfg")
	res := mustRun(t, r, "system", "backup", path)
	if res.Code != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(tendatest.ConfigBytes) {
		t.Fatalf("file content = %q", b)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v", info.Mode())
	}
}

func TestBackupRefusesExistingFile(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	path := filepath.Join(t.TempDir(), "RouterCfm.cfg")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	res := runCLI(t, r, "system", "backup", path)
	if res.Code != 1 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "existing" {
		t.Fatalf("file was overwritten: %q", b)
	}
}
