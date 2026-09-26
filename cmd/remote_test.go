package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestRemoteShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "remote", mustRun(t, r, "system", "remote").Stdout)
	if got := mustRun(t, r, "system", "remote", "show").Stdout; got != mustRun(t, r, "system", "remote").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "remote_json", mustRun(t, r, "system", "remote", "-o", "json").Stdout)
}

func TestRemoteSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "system", "remote", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestRemoteSetInvalid(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"system", "remote", "set", "--from", "bogus"},
		{"system", "remote", "set", "--port", "0"},
		{"system", "remote", "set", "--port", "70000"},
	} {
		r := tendatest.New(t)
		res := runCLI(t, r, args...)
		if res.Code != 1 || len(r.CallsTo("SetRemoteWebCfg")) != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, res.Code, res.Stderr)
		}
	}
}

func TestRemoteSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "system", "remote", "set", "--port", "9443", "--from", "203.0.113.5")
	want := url.Values{"remoteWebEn": {"0"}, "remoteIp": {"203.0.113.5"}, "remotePort": {"9443"}}.Encode()
	if got := r.LastCall(t, "SetRemoteWebCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestRemoteEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "system", "remote", "enable")
	if got := r.LastCall(t, "SetRemoteWebCfg").Form.Get("remoteWebEn"); got != "1" {
		t.Fatalf("remoteWebEn = %q", got)
	}
	mustRun(t, r, "system", "remote", "disable")
	if got := r.LastCall(t, "SetRemoteWebCfg").Form.Get("remoteWebEn"); got != "0" {
		t.Fatalf("remoteWebEn = %q", got)
	}
}
