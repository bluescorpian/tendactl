package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestLANShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "lan", mustRun(t, r, "lan").Stdout)
	if got := mustRun(t, r, "lan", "show").Stdout; got != mustRun(t, r, "lan").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "lan_json", mustRun(t, r, "lan", "-o", "json").Stdout)
}

func TestLANSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "lan", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestLANSetInvalidIP(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "lan", "set", "--ip", "bogus", "--yes")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --ip") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestLANSetInvalidLeaseTime(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "lan", "set", "--lease-time", "60", "--yes")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --lease-time") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

// Any lan set needs --yes: AdvSetLanip is unconditionally hazardous.
func TestLANSetNeedsYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Allow("AdvSetLanip")
	res := runCLI(t, r, "lan", "set", "--dns1", "9.9.9.9")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if n := len(r.CallsTo("AdvSetLanip")); n != 0 {
		t.Fatalf("AdvSetLanip calls = %d, want 0", n)
	}
	mustRun(t, r, "lan", "set", "--dns1", "9.9.9.9", "--yes")
	if got := r.LastCall(t, "AdvSetLanip").Form.Get("lanDns1"); got != "9.9.9.9" {
		t.Fatalf("lanDns1 = %q", got)
	}
}

func TestLANSetDHCPRange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Allow("AdvSetLanip")
	mustRun(t, r, "lan", "set", "--dhcp-start", "192.168.0.150", "--dhcp-end", "192.168.0.250", "--yes")
	got := r.LastCall(t, "AdvSetLanip").Form
	if got.Get("startIp") != "192.168.0.150" || got.Get("endIp") != "192.168.0.250" {
		t.Fatalf("form = %v", got)
	}
	// Unchanged fields are resent from the current config.
	if got.Get("lanIp") != "192.168.0.1" || got.Get("leaseTime") != "604800" {
		t.Fatalf("form = %v", got)
	}
}
