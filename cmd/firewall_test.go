package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestFirewallShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "firewall", mustRun(t, r, "firewall").Stdout)
	if got := mustRun(t, r, "firewall", "show").Stdout; got != mustRun(t, r, "firewall").Stdout {
		t.Fatalf("firewall show differs from firewall:\n%s", got)
	}
	golden(t, "firewall_json", mustRun(t, r, "firewall", "-o", "json").Stdout)
}

func TestFirewallSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "firewall", "set", "--ignore-wan-ping", "--tcp-flood=false")
	want := url.Values{"firewallEn": {"1011"}}.Encode()
	if got := r.LastCall(t, "SetFirewallCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if res.Stdout != "Firewall settings updated\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestFirewallSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "firewall", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") || len(r.CallsTo("SetFirewallCfg")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
