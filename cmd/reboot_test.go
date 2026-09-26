package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestRebootNeedsYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "system", "reboot")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if n := len(r.CallsTo("SysToolReboot")); n != 0 {
		t.Fatalf("calls = %d, want 0", n)
	}
}

func TestReboot(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Allow("SysToolReboot")
	res := mustRun(t, r, "system", "reboot", "--yes")
	if res.Stdout != "Router rebooting\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if got := r.LastCall(t, "SysToolReboot").Form.Get("action"); got != "0" {
		t.Fatalf("action = %q", got)
	}
}
