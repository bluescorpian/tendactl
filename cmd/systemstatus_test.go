package cmd

import (
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSystemStatusShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "systemstatus", mustRun(t, r, "system", "status").Stdout)
	golden(t, "systemstatus_json", mustRun(t, r, "system", "status", "-o", "json").Stdout)
}

func TestSystemStatusError(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetSystemStatus", 500, "boom")
	res := runCLI(t, r, "system", "status")
	if res.Code != 1 || res.Stdout != "" || res.Stderr != "tendactl: GetSystemStatus: HTTP 500: boom\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", res.Code, res.Stdout, res.Stderr)
	}
}
