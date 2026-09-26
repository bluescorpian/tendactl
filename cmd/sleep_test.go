package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSleepShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "sleep", mustRun(t, r, "sleep").Stdout)
	if got := mustRun(t, r, "sleep", "show").Stdout; got != mustRun(t, r, "sleep").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "sleep_json", mustRun(t, r, "sleep", "-o", "json").Stdout)
}

func TestSleepSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "sleep", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestSleepSetInvalidTime(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "sleep", "set", "--time", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --time") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestSleepSetLEDsAndDelay(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "sleep", "set", "--leds", "except-power", "--delay", "off")
	got := r.LastCall(t, "PowerSaveSet").Form
	if got.Get("ledCloseType") != "unpowerClose" || got.Get("powerSaveDelay") != "0" {
		t.Fatalf("form = %v", got)
	}
	// powerSavingEn is resent unchanged (0, from the fixture), so no --yes is needed.
	if got.Get("powerSavingEn") != "0" {
		t.Fatalf("powerSavingEn = %q", got.Get("powerSavingEn"))
	}
}

func TestSleepSetInvalidLEDs(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "sleep", "set", "--leds", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --leds") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

// Enabling Sleeping Mode needs --yes, since the fixture has it disabled.
func TestSleepEnableNeedsYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "sleep", "enable")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if n := len(r.CallsTo("PowerSaveSet")); n != 0 {
		t.Fatalf("PowerSaveSet calls = %d, want 0", n)
	}
	mustRun(t, r, "sleep", "enable", "--yes")
	if got := r.LastCall(t, "PowerSaveSet").Form.Get("powerSavingEn"); got != "1" {
		t.Fatalf("powerSavingEn = %q", got)
	}
}

func TestSleepDisableNeedsNoYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "sleep", "disable")
	if got := r.LastCall(t, "PowerSaveSet").Form.Get("powerSavingEn"); got != "0" {
		t.Fatalf("powerSavingEn = %q", got)
	}
}
