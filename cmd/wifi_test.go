package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiBasicShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "wifi", mustRun(t, r, "wifi", "show").Stdout)
	golden(t, "wifi_json", mustRun(t, r, "wifi", "show", "-o", "json").Stdout)

	if got := mustRun(t, r, "wifi", "show").Stdout; strings.Contains(got, "REDACTED") {
		t.Fatalf("password not masked: %q", got)
	}
	if got := mustRun(t, r, "wifi", "show", "--show-password").Stdout; !strings.Contains(got, "REDACTED") {
		t.Fatalf("password not shown: %q", got)
	}
}

func TestWiFiBasicSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestWiFiBasicSetSSID(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "set", "--band", "2.4", "--ssid", "NewName")
	got := r.LastCall(t, "WifiBasicSet").Form
	if got.Get("ssid") != "NewName" || got.Get("ssid_5g") != "WiFi-17" {
		t.Fatalf("form = %v", got)
	}
}

func TestWiFiBasicSetInvalidSecurity(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "set", "--security", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --security") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestWiFiBasicEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "enable", "--band", "5", "--yes")
	got := r.LastCall(t, "WifiBasicSet").Form
	if got.Get("wrlEn_5g") != "1" {
		t.Fatalf("form = %v", got)
	}

	// Disabling an on band needs --yes and sends nothing without it.
	res := runCLI(t, r, "wifi", "disable", "--band", "2.4")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if n := len(r.CallsTo("WifiBasicSet")); n != 1 { // only the earlier enable call
		t.Fatalf("WifiBasicSet calls = %d, want 1", n)
	}

	mustRun(t, r, "wifi", "disable", "--band", "2.4", "--yes")
	want := url.Values{"wrlEn": {"1"}}.Encode()
	if got := r.LastCall(t, "WifiBasicSet").Form.Get("wrlEn"); got != "0" {
		t.Fatalf("wrlEn = %q, want 0 (sanity: %s)", got, want)
	}
}

func TestWiFiBasicHideUnhide(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "hide", "--band", "5")
	if got := r.LastCall(t, "WifiBasicSet").Form.Get("hideSsid_5g"); got != "1" {
		t.Fatalf("hideSsid_5g = %q", got)
	}
	mustRun(t, r, "wifi", "unhide", "--band", "5")
	if got := r.LastCall(t, "WifiBasicSet").Form.Get("hideSsid_5g"); got != "0" {
		t.Fatalf("hideSsid_5g = %q", got)
	}
}
