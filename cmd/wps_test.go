package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWPSShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "wps", mustRun(t, r, "wifi", "wps").Stdout)
	if got := mustRun(t, r, "wifi", "wps", "show").Stdout; got != mustRun(t, r, "wifi", "wps").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "wps_json", mustRun(t, r, "wifi", "wps", "-o", "json").Stdout)

	if got := mustRun(t, r, "wifi", "wps").Stdout; strings.Contains(got, "REDACTED") {
		t.Fatalf("PIN not masked: %q", got)
	}
	if got := mustRun(t, r, "wifi", "wps", "--show-password").Stdout; !strings.Contains(got, "REDACTED") {
		t.Fatalf("PIN not shown: %q", got)
	}
	if got := mustRun(t, r, "wifi", "wps", "-o", "json").Stdout; strings.Contains(got, "REDACTED") {
		t.Fatalf("PIN not masked in JSON: %q", got)
	}
	if got := mustRun(t, r, "wifi", "wps", "-o", "json", "--show-password").Stdout; !strings.Contains(got, "REDACTED") {
		t.Fatalf("PIN not shown in JSON: %q", got)
	}
}

func TestWPSEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "wps", "enable")
	want := url.Values{"wpsEn": {"1"}}.Encode()
	if got := r.LastCall(t, "WifiWpsSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	mustRun(t, r, "wifi", "wps", "disable")
	want = url.Values{"wpsEn": {"0"}}.Encode()
	if got := r.LastCall(t, "WifiWpsSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWPSStart(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "wifi", "wps", "start")
	want := url.Values{"action": {"wps"}}.Encode()
	if got := r.LastCall(t, "WifiWpsStart").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if !strings.Contains(res.Stdout, "started") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}
