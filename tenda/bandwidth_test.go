package tenda

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestBandwidth(t *testing.T) {
	r := tendatest.New(t)
	b, err := newTestClient(t, r).Bandwidth(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if b.Enabled {
		t.Fatalf("enabled = %v, want false", b.Enabled)
	}
	if len(b.Devices) != 13 {
		t.Fatalf("devices = %d", len(b.Devices))
	}
	first := b.Devices[0]
	if first.MAC != "02:00:00:00:00:04" || first.Name != "Device-3" || first.UpKBps != "2" || first.LimitUpMbps != 0 || first.Limited || first.Offline {
		t.Fatalf("first = %+v", first)
	}
}

func TestBandwidthSetLimit(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	up, down := 2.0, 0.5
	if err := c.SetBandwidthLimit(context.Background(), "02:00:00:00:00:04", &up, &down); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetNetControlList")
	// js/net_control.js's getSubmitData() never includes netControlEn in
	// the outgoing body for any SetNetControlList call -- the field is
	// commented out for every case, including a plain per-device limit
	// edit like this one.
	if _, present := call.Form["netControlEn"]; present {
		t.Fatalf("form has netControlEn: %v", call.Form)
	}
	rows := LineCR.Decode(call.Form.Get("list"))
	if len(rows) != 13 {
		t.Fatalf("rows = %d", len(rows))
	}
	if got := rows[0]; got[0] != "Device-3" || got[1] != "02:00:00:00:00:04" || got[2] != "256" || got[3] != "64" {
		t.Fatalf("row 0 = %v", got)
	}
	// Other rows are unchanged.
	if got := rows[1]; got[2] != "0" || got[3] != "0" {
		t.Fatalf("row 1 = %v", got)
	}
	if !strings.HasPrefix(call.RawBody, "list=") {
		t.Fatalf("body = %q, want it to start with %q", call.RawBody, "list=")
	}
}

func TestBandwidthSetLimitPartial(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	up := 1.0
	if err := c.SetBandwidthLimit(context.Background(), "02:00:00:00:00:04", &up, nil); err != nil {
		t.Fatal(err)
	}
	rows := LineCR.Decode(r.LastCall(t, "SetNetControlList").Form.Get("list"))
	if rows[0][2] != "128" || rows[0][3] != "0" {
		t.Fatalf("row 0 = %v", rows[0])
	}
}

func TestBandwidthSetLimitNoDevice(t *testing.T) {
	r := tendatest.New(t)
	err := newTestClient(t, r).SetBandwidthLimit(context.Background(), "aa:bb:cc:dd:ee:ff", nil, nil)
	if !errors.Is(err, ErrBandwidthNoDevice) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("SetNetControlList")); n != 0 {
		t.Fatalf("posts = %d, want 0", n)
	}
}

func TestBandwidthRemoveLimit(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.RemoveBandwidthLimit(context.Background(), "02:00:00:00:00:04"); err != nil {
		t.Fatal(err)
	}
	rows := LineCR.Decode(r.LastCall(t, "SetNetControlList").Form.Get("list"))
	if rows[0][2] != "0" || rows[0][3] != "0" {
		t.Fatalf("row 0 = %v", rows[0])
	}
}

func TestBandwidthSetEnabled(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetBandwidthEnabled(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetNetControlList")
	if _, present := call.Form["netControlEn"]; present {
		t.Fatalf("form has netControlEn: %v", call.Form)
	}
	rows := LineCR.Decode(call.Form.Get("list"))
	if len(rows) != 13 {
		t.Fatalf("rows = %d, want unchanged device list", len(rows))
	}
}
