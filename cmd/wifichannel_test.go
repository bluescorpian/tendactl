package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiChannelShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "wifichannel", mustRun(t, r, "wifi", "channel").Stdout)
	if got := mustRun(t, r, "wifi", "channel", "show").Stdout; got != mustRun(t, r, "wifi", "channel").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "wifichannel_json", mustRun(t, r, "wifi", "channel", "-o", "json").Stdout)
}

func TestWiFiChannelSetRequiresBand(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "channel", "set", "--channel", "6")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--band is required") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestWiFiChannelSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "channel", "set", "--band", "2.4")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestWiFiChannelSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "channel", "set", "--band", "2.4", "--channel", "6")
	got := r.LastCall(t, "WifiRadioSet").Form
	if got.Get("adv_channel") != "6" || got.Get("adv_band") != "20" || got.Get("adv_mode_5g") != "ac" {
		t.Fatalf("form = %v", got)
	}
}

func TestWiFiChannelSetInvalidChannel(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "channel", "set", "--band", "2.4", "--channel", "99")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --channel") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestWiFiChannelSetInvalidMode(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "channel", "set", "--band", "5", "--mode", "bgn")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --mode") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
