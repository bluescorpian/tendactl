package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestDMZShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "dmz", mustRun(t, r, "dmz").Stdout)
	if got := mustRun(t, r, "dmz", "show").Stdout; got != mustRun(t, r, "dmz").Stdout {
		t.Fatalf("dmz show differs from dmz:\n%s", got)
	}
	golden(t, "dmz_json", mustRun(t, r, "dmz", "-o", "json").Stdout)
}

func TestDMZSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "dmz", "set", "192.168.0.150")
	want := url.Values{"dmzEn": {"1"}, "dmzIp": {"192.168.0.150"}}.Encode()
	if got := r.LastCall(t, "SetDMZCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if res.Stdout != "DMZ host set to 192.168.0.150\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestDMZEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "dmz", "enable")
	want := url.Values{"dmzEn": {"1"}, "dmzIp": {"192.168.0.100"}}.Encode()
	if got := r.LastCall(t, "SetDMZCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	mustRun(t, r, "dmz", "disable")
	want = url.Values{"dmzEn": {"0"}, "dmzIp": {"192.168.0.100"}}.Encode()
	if got := r.LastCall(t, "SetDMZCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestDMZSetHostIsLAN(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.ErrCode("SetDMZCfg", 2)
	res := runCLI(t, r, "dmz", "set", "192.168.0.1")
	if res.Code != 1 || !strings.Contains(res.Stderr, "equals the router's LAN IP") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
