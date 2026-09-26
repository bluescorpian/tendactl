package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestMACFilterShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "macfilter").Stdout
	golden(t, "macfilter", text)
	if got := mustRun(t, r, "macfilter", "show").Stdout; got != text {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "macfilter_json", mustRun(t, r, "macfilter", "-o", "json").Stdout)
}

func TestMACFilterAdd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "macfilter", "add", "6c:4c:bc:83:95:2b", "--name", "Device6")
	if got := r.LastCall(t, "setMacFilterCfg").Form.Get("deviceList"); got != "Device6\r6C:4C:BC:83:95:2B" {
		t.Fatalf("deviceList = %q", got)
	}
	if res.Stdout != "Added 6c:4c:bc:83:95:2b\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestMACFilterRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("getMacFilterCfg", 200, `{"localhostIP":"1","localhostName":"h","localhostMac":"m","macFilterType":"black","blackList":[{"devName":"D","devMac":"AA:BB:CC:DD:EE:FF"}],"whiteList":[],"onlineList":[]}`)
	mustRun(t, r, "macfilter", "rm", "aa:bb:cc:dd:ee:ff")
	if got := r.LastCall(t, "setMacFilterCfg").Form.Get("deviceList"); got != "" {
		t.Fatalf("deviceList = %q", got)
	}

	res := runCLI(t, r, "macfilter", "rm", "11:22:33:44:55:66")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no entry") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestMACFilterModeBlackNoYesNeeded(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "macfilter", "mode", "black")
	if got := r.LastCall(t, "setMacFilterCfg").Form.Get("macFilterType"); got != "black" {
		t.Fatalf("macFilterType = %q", got)
	}
}

func TestMACFilterModeWhiteNeedsYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "macfilter", "mode", "white")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") || len(r.CallsTo("setMacFilterCfg")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	mustRun(t, r, "macfilter", "mode", "white", "--yes")
	if got := r.LastCall(t, "setMacFilterCfg").Form.Get("macFilterType"); got != "white" {
		t.Fatalf("macFilterType = %q", got)
	}
}
