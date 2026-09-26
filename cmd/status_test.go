package cmd

import (
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestStatusCmd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "status")
	if want := readGolden(t, "status_legacy.golden"); res.Stdout != want {
		t.Fatalf("status output changed:\ngot\n%q\nwant\n%q", res.Stdout, want)
	}
	golden(t, "status_json", mustRun(t, r, "status", "-o", "json").Stdout)
}

func TestStatusCmdError(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetRouterStatus", 500, "boom")
	res := runCLI(t, r, "status")
	if res.Code != 1 || res.Stdout != "" || res.Stderr != "tendactl: GetRouterStatus: HTTP 500: boom\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", res.Code, res.Stdout, res.Stderr)
	}
}
