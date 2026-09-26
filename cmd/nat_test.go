package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

const natFixtureList = "192.168.0.5,25565,25565,1~192.168.0.5,42069,42069,1~192.168.0.5,24454,24454,2~192.168.0.5,19284,19284,2~192.168.0.5,8100,8100,1~192.168.0.5,80,80,1~192.168.0.5,443,443,1"

func TestNATList(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "nat").Stdout
	golden(t, "nat", text)
	if got := mustRun(t, r, "nat", "list").Stdout; got != text {
		t.Fatalf("nat list differs from nat:\n%s", got)
	}
	golden(t, "nat_json", mustRun(t, r, "nat", "list", "-o", "json").Stdout)
}

func TestNATListEmpty(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetVirtualServerCfg", 200, `{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","virtualList":[]}`)
	if got := mustRun(t, r, "nat").Stdout; got != "No port forwarding rules configured\n" {
		t.Fatalf("got %q", got)
	}
	if got := mustRun(t, r, "nat", "-o", "json").Stdout; got != "[]\n" {
		t.Fatalf("got %q", got)
	}
}

func TestNATAdd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "nat", "add", "192.168.0.9", "8080")
	want := url.Values{"list": {natFixtureList + "~192.168.0.9,8080,8080,0"}}.Encode()
	if got := r.LastCall(t, "SetVirtualServerCfg").RawBody; got != want {
		t.Fatalf("body = %q\nwant   %q", got, want)
	}
	if res.Stdout != "Forwarding WAN port 8080 to 192.168.0.9:8080 (TCP&UDP)\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}

	mustRun(t, r, "nat", "add", "192.168.0.9", "22", "2222", "--proto", "udp")
	want = url.Values{"list": {natFixtureList + "~192.168.0.9,22,2222,2"}}.Encode()
	if got := r.LastCall(t, "SetVirtualServerCfg").RawBody; got != want {
		t.Fatalf("body = %q\nwant   %q", got, want)
	}
}

func TestNATAddRefused(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"nat", "add", "192.168.0.9", "443"},    // duplicate WAN port
		{"nat", "add", "192.168.0.9~x", "1234"}, // separator in the IP
		{"nat", "add", "192.168.0.9", "0"},
		{"nat", "add", "192.168.0.9", "80", "--proto", "icmp"},
	} {
		r := tendatest.New(t)
		res := runCLI(t, r, args...)
		if res.Code != 1 || res.Stderr == "" || len(r.CallsTo("SetVirtualServerCfg")) != 0 {
			t.Fatalf("%v: exit %d, stderr %q, posts %d", args, res.Code, res.Stderr, len(r.CallsTo("SetVirtualServerCfg")))
		}
	}
}

func TestNATRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "nat", "rm", "443")
	want := url.Values{"list": {strings.TrimSuffix(natFixtureList, "~192.168.0.5,443,443,1")}}.Encode()
	if got := r.LastCall(t, "SetVirtualServerCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}

	r = tendatest.New(t)
	res := runCLI(t, r, "nat", "rm", "9999")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no port forwarding rule") || len(r.CallsTo("SetVirtualServerCfg")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
