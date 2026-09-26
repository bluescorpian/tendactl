package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestMaintenance(t *testing.T) {
	r := tendatest.New(t)
	m, err := newTestClient(t, r).Maintenance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !m.Enabled || m.RebootTime != "03:00" || m.DelayIfBusy || m.TimeSynced {
		t.Fatalf("m = %+v", m)
	}
}

func TestSetMaintenance(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	m := Maintenance{Enabled: true, RebootTime: "04:30", DelayIfBusy: true}
	if err := c.SetMaintenance(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"autoRebootEn": {"1"}, "delayRebootEn": {"true"}, "rebootTime": {"04:30"}}.Encode()
	if got := r.LastCall(t, "SetSysAutoRebbotCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}

	m.Enabled, m.DelayIfBusy = false, false
	if err := c.SetMaintenance(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	want = url.Values{"autoRebootEn": {"0"}, "delayRebootEn": {"false"}, "rebootTime": {"04:30"}}.Encode()
	if got := r.LastCall(t, "SetSysAutoRebbotCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
