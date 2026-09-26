package tenda

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiSchedule(t *testing.T) {
	r := tendatest.New(t)
	s, err := newTestClient(t, r).WiFiSchedule(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled || s.Start != "00:00" || s.End != "07:00" || s.EveryDay {
		t.Fatalf("s = %+v", s)
	}
	want := []string{"mon", "tue", "wed", "thu", "fri"}
	if len(s.Days) != len(want) {
		t.Fatalf("days = %v", s.Days)
	}
	for i, d := range want {
		if s.Days[i] != d {
			t.Fatalf("days = %v", s.Days)
		}
	}
	if !s.WorkModeOK {
		t.Fatalf("workModeOk = %v", s.WorkModeOK)
	}
}

func TestWiFiScheduleSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r, WithConfirm(allowAll))
	prev := WiFiSchedule{Enabled: false, Start: "00:00", End: "07:00", EveryDay: false, Days: []string{"mon"}}
	next := prev
	next.Enabled = true
	if err := c.SetWiFiSchedule(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"schedWifiEnable": {"1"}, "schedStartTime": {"00:00"}, "schedEndTime": {"07:00"},
		"timeType": {"1"}, "day": {"1,0,0,0,0,0,0"},
	}.Encode()
	if got := r.LastCall(t, "openSchedWifi").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWiFiScheduleSetWhileDisabledResendsPrevWindow(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	prev := WiFiSchedule{Enabled: false, Start: "00:00", End: "07:00", EveryDay: false, Days: []string{"mon"}}
	next := prev
	next.Start, next.End = "22:00", "06:00"
	next.Days = []string{"tue"}
	if err := c.SetWiFiSchedule(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	// schedWifiEnable=0 must be paired with prev's start/end/day, exactly as
	// js/wifi_time.js's getSubmitData() sends when the toggle isn't "1" --
	// never with the new, unapplied values from next.
	want := url.Values{
		"schedWifiEnable": {"0"}, "schedStartTime": {"00:00"}, "schedEndTime": {"07:00"},
		"timeType": {"1"}, "day": {"1,0,0,0,0,0,0"},
	}.Encode()
	if got := r.LastCall(t, "openSchedWifi").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWiFiScheduleSetEnableChangedNeedsConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r) // no WithConfirm: every hazard is refused
	prev := WiFiSchedule{Enabled: false, Start: "00:00", End: "07:00"}
	next := prev
	next.Enabled = true
	err := c.SetWiFiSchedule(context.Background(), prev, next)
	var he *HazardError
	if !errors.As(err, &he) || !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("err = %v, want *HazardError", err)
	}
	if len(r.CallsTo("openSchedWifi")) != 0 {
		t.Fatal("request sent")
	}
}

func TestWiFiScheduleSetResubmitNeedsNoConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	s := WiFiSchedule{Enabled: true, Start: "00:00", End: "07:00", Days: []string{"mon"}}
	if err := c.SetWiFiSchedule(context.Background(), s, s); err != nil {
		t.Fatal(err)
	}
}

func TestWiFiScheduleSetDisableNeedsNoConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	prev := WiFiSchedule{Enabled: true, Start: "00:00", End: "07:00"}
	next := prev
	next.Enabled = false
	if err := c.SetWiFiSchedule(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
}
