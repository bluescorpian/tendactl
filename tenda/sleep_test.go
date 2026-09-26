package tenda

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSleep(t *testing.T) {
	r := tendatest.New(t)
	s, err := newTestClient(t, r).Sleep(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled || s.Window != "00:00-07:00" || s.LEDs != "allClose" || !s.Delay {
		t.Fatalf("s = %+v", s)
	}
	if !s.WorkModeOK || s.TimeSynced {
		t.Fatalf("s = %+v", s)
	}
}

func TestSleepSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r, WithConfirm(allowAll))
	prev := Sleep{Enabled: false, Window: "00:00-07:00", LEDs: "allClose", Delay: true}
	next := prev
	next.Enabled = true
	if err := c.SetSleep(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"powerSavingEn": {"1"}, "time": {"00:00-07:00"}, "ledCloseType": {"allClose"}, "powerSaveDelay": {"1"},
	}.Encode()
	if got := r.LastCall(t, "PowerSaveSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestSleepEnableNeedsConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r) // no WithConfirm: every hazard is refused
	prev := Sleep{Enabled: false, Window: "00:00-07:00"}
	next := prev
	next.Enabled = true
	err := c.SetSleep(context.Background(), prev, next)
	var he *HazardError
	if !errors.As(err, &he) || !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("err = %v, want *HazardError", err)
	}
	if len(r.CallsTo("PowerSaveSet")) != 0 {
		t.Fatal("request sent")
	}
}

func TestSleepResubmitNeedsNoConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	s := Sleep{Enabled: true, Window: "00:00-07:00", LEDs: "allClose"}
	if err := c.SetSleep(context.Background(), s, s); err != nil {
		t.Fatal(err)
	}
}

func TestSleepDisableNeedsNoConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	prev := Sleep{Enabled: true, Window: "00:00-07:00"}
	next := prev
	next.Enabled = false
	if err := c.SetSleep(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
}
