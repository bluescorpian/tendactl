package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestMaintenanceShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "maintenance", mustRun(t, r, "system", "maintenance").Stdout)
	if got := mustRun(t, r, "system", "maintenance", "show").Stdout; got != mustRun(t, r, "system", "maintenance").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "maintenance_json", mustRun(t, r, "system", "maintenance", "-o", "json").Stdout)
}

func TestMaintenanceSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "system", "maintenance", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestMaintenanceSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "system", "maintenance", "set", "--time", "04:30", "--delay", "on")
	want := url.Values{"autoRebootEn": {"1"}, "delayRebootEn": {"true"}, "rebootTime": {"04:30"}}.Encode()
	if got := r.LastCall(t, "SetSysAutoRebbotCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestMaintenanceSetInvalidTime(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "system", "maintenance", "set", "--time", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --time") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestMaintenanceEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "system", "maintenance", "disable")
	if got := r.LastCall(t, "SetSysAutoRebbotCfg").Form.Get("autoRebootEn"); got != "0" {
		t.Fatalf("autoRebootEn = %q", got)
	}
	mustRun(t, r, "system", "maintenance", "enable")
	if got := r.LastCall(t, "SetSysAutoRebbotCfg").Form.Get("autoRebootEn"); got != "1" {
		t.Fatalf("autoRebootEn = %q", got)
	}
}
