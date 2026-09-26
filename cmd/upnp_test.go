package cmd

import (
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestUPnPShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "upnp", mustRun(t, r, "upnp").Stdout)
	if got := mustRun(t, r, "upnp", "show").Stdout; got != mustRun(t, r, "upnp").Stdout {
		t.Fatalf("upnp show differs from upnp:\n%s", got)
	}
	golden(t, "upnp_json", mustRun(t, r, "upnp", "-o", "json").Stdout)
}

func TestUPnPEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "upnp", "disable")
	want := url.Values{"upnpEn": {"0"}}.Encode()
	if got := r.LastCall(t, "SetUpnpCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if res.Stdout != "UPnP disabled\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	mustRun(t, r, "upnp", "enable")
	want = url.Values{"upnpEn": {"1"}}.Encode()
	if got := r.LastCall(t, "SetUpnpCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
