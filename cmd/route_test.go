package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestRouteList(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "route").Stdout
	golden(t, "route", text)
	if got := mustRun(t, r, "route", "list").Stdout; got != text {
		t.Fatalf("route list differs from route:\n%s", got)
	}
	golden(t, "route_json", mustRun(t, r, "route", "list", "-o", "json").Stdout)
}

func TestRouteAdd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "route", "add", "10.0.0.0", "255.255.255.0", "192.168.0.254")
	want := url.Values{"list": {"10.0.0.0,255.255.255.0,192.168.0.254,WAN1"}}.Encode()
	if got := r.LastCall(t, "SetStaticRouteCfg").RawBody; got != want {
		t.Fatalf("body = %q\nwant   %q", got, want)
	}
	if res.Stdout != "Added route to 10.0.0.0/255.255.255.0 via 192.168.0.254\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestRouteAddRefused(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "route", "add", "not-an-ip", "255.255.255.0", "0.0.0.0")
	if res.Code != 1 || res.Stderr == "" || len(r.CallsTo("SetStaticRouteCfg")) != 0 {
		t.Fatalf("exit %d, stderr %q, posts %d", res.Code, res.Stderr, len(r.CallsTo("SetStaticRouteCfg")))
	}
}

func TestRouteRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "route", "rm", "0.0.0.0", "0.0.0.0")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no user-added static route") || len(r.CallsTo("SetStaticRouteCfg")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
