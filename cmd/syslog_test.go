package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSysLogShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "syslog", mustRun(t, r, "system", "log").Stdout)
	golden(t, "syslog_json", mustRun(t, r, "system", "log", "-o", "json").Stdout)
}

func TestSysLogShowEmpty(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetSySLogCfg", 200, `[]`)
	if got := mustRun(t, r, "system", "log").Stdout; got != "No log entries\n" {
		t.Fatalf("got %q", got)
	}
	if got := mustRun(t, r, "system", "log", "-o", "json").Stdout; got != "[]\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSysLogDownload(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	path := filepath.Join(t.TempDir(), "syslog.tar")
	res := mustRun(t, r, "system", "log", "--download", path)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(tendatest.SyslogBytes) {
		t.Fatalf("file content = %q", b)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v", info.Mode())
	}
	if res.Code != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
