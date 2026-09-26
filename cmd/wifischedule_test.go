package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiScheduleShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "wifischedule", mustRun(t, r, "wifi", "schedule").Stdout)
	if got := mustRun(t, r, "wifi", "schedule", "show").Stdout; got != mustRun(t, r, "wifi", "schedule").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "wifischedule_json", mustRun(t, r, "wifi", "schedule", "-o", "json").Stdout)
}

func TestWiFiScheduleSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "schedule", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestWiFiScheduleSetInvalidTime(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "schedule", "set", "--time", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --time") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

// wifiScheduleEnableFixture makes the fake router report the schedule as
// enabled, like TestWiFiScheduleSetTimeAndDays and TestWiFiScheduleSetDaysAll
// need: the router UI only takes new time/day values from the submitted form
// when schedWifiEnable==1, so a --time/--days change is only observable on
// the wire while the schedule is on (js/wifi_time.js getSubmitData).
func wifiScheduleEnableFixture(r *tendatest.Router) {
	r.Handle("initSchedWifi", func(tendatest.Call) tendatest.Response {
		return tendatest.JSON(`{"wifiEn":1,"schedWifiEnable":1,"schedStartTime":"00:00","schedEndTime":"07:00","timeType":"1","day":"1,1,1,1,1,0,0","powerSaveTime":"","wl_mode":"ap","timeUp":"0"}`)
	})
}

func TestWiFiScheduleSetTimeAndDays(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	wifiScheduleEnableFixture(r)
	mustRun(t, r, "wifi", "schedule", "set", "--time", "22:00-06:00", "--days", "sat,sun", "--yes")
	got := r.LastCall(t, "openSchedWifi").Form
	if got.Get("schedStartTime") != "22:00" || got.Get("schedEndTime") != "06:00" {
		t.Fatalf("form = %v", got)
	}
	if got.Get("day") != "0,0,0,0,0,1,1" || got.Get("timeType") != "1" {
		t.Fatalf("day/timeType = %v", got)
	}
}

func TestWiFiScheduleSetDaysAll(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	wifiScheduleEnableFixture(r)
	mustRun(t, r, "wifi", "schedule", "set", "--days", "all", "--yes")
	if got := r.LastCall(t, "openSchedWifi").Form.Get("timeType"); got != "0" {
		t.Fatalf("timeType = %q", got)
	}
}

// TestWiFiScheduleSetWhileDisabledResendsPrevWindow covers the fix itself:
// the fixture's schedule starts disabled, so --time/--days must not reach
// the wire -- the router UI resends the previously-read window unchanged
// whenever schedWifiEnable isn't "1" (js/wifi_time.js getSubmitData).
func TestWiFiScheduleSetWhileDisabledResendsPrevWindow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "schedule", "set", "--time", "22:00-06:00", "--days", "sat,sun")
	got := r.LastCall(t, "openSchedWifi").Form
	if got.Get("schedWifiEnable") != "0" {
		t.Fatalf("schedWifiEnable = %v", got)
	}
	if got.Get("schedStartTime") != "00:00" || got.Get("schedEndTime") != "07:00" {
		t.Fatalf("form = %v", got)
	}
	if got.Get("day") != "1,1,1,1,1,0,0" || got.Get("timeType") != "1" {
		t.Fatalf("day/timeType = %v", got)
	}
}

// Enabling the schedule needs --yes, since its window (00:00-07:00 on the
// fixture) differs from the disabled state.
func TestWiFiScheduleEnableNeedsYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "schedule", "enable")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if n := len(r.CallsTo("openSchedWifi")); n != 0 {
		t.Fatalf("openSchedWifi calls = %d, want 0", n)
	}
	mustRun(t, r, "wifi", "schedule", "enable", "--yes")
	if got := r.LastCall(t, "openSchedWifi").Form.Get("schedWifiEnable"); got != "1" {
		t.Fatalf("schedWifiEnable = %q", got)
	}
}

func TestWiFiScheduleDisableNeedsNoYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "wifi", "schedule", "disable")
	if got := r.LastCall(t, "openSchedWifi").Form.Get("schedWifiEnable"); got != "0" {
		t.Fatalf("schedWifiEnable = %q", got)
	}
}
