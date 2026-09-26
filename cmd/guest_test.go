package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestGuestShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "guest", mustRun(t, r, "guest").Stdout)
	if got := mustRun(t, r, "guest", "show").Stdout; got != mustRun(t, r, "guest").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "guest_json", mustRun(t, r, "guest", "-o", "json").Stdout)

	if got := mustRun(t, r, "guest").Stdout; strings.Contains(got, "REDACTED") {
		t.Fatalf("password not masked: %q", got)
	}
	if got := mustRun(t, r, "guest", "--show-password").Stdout; !strings.Contains(got, "REDACTED") {
		t.Fatalf("password not shown: %q", got)
	}
}

func TestGuestSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "guest", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestGuestSetSSID(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "guest", "set", "--band", "2.4", "--ssid", "NewGuest")
	got := r.LastCall(t, "WifiGuestSet").Form
	if got.Get("guestSsid") != "NewGuest" || got.Get("guestSsid_5g") != "WiFi-18" {
		t.Fatalf("form = %v", got)
	}
}

func TestGuestSetInvalidEffectiveTime(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "guest", "set", "--effective-time", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --effective-time") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestGuestEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "guest", "disable")
	if got := r.LastCall(t, "WifiGuestSet").Form.Get("guestEn"); got != "0" {
		t.Fatalf("guestEn = %q", got)
	}
	mustRun(t, r, "guest", "enable")
	if got := r.LastCall(t, "WifiGuestSet").Form.Get("guestEn"); got != "1" {
		t.Fatalf("guestEn = %q", got)
	}
}
