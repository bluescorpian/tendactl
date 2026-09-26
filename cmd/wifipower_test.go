package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiPowerShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "wifipower", mustRun(t, r, "wifi", "power").Stdout)
	golden(t, "wifipower_json", mustRun(t, r, "wifi", "power", "-o", "json").Stdout)
}

func TestWiFiPowerSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "power", "set", "--band", "2.4", "mid")
	got := r.LastCall(t, "WifiPowerSet").Form
	if got.Get("power") != "middle" || got.Get("power_5g") != "high" {
		t.Fatalf("form = %v", got)
	}
}

func TestWiFiPowerSetInvalidLevel(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "power", "set", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid power level") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
