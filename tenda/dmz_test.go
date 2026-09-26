package tenda

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestDMZ(t *testing.T) {
	r := tendatest.New(t)
	d, err := newTestClient(t, r).DMZ(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Enabled || d.HostIP != "192.168.0.100" || d.LANIP != "192.168.0.1" {
		t.Fatalf("d = %+v", d)
	}
}

func TestSetDMZ(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetDMZ(context.Background(), DMZ{Enabled: true, HostIP: "192.168.0.100"}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"dmzEn": {"1"}, "dmzIp": {"192.168.0.100"}}.Encode()
	if got := r.LastCall(t, "SetDMZCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestSetDMZHostIsLAN(t *testing.T) {
	r := tendatest.New(t)
	r.ErrCode("SetDMZCfg", 2)
	err := newTestClient(t, r).SetDMZ(context.Background(), DMZ{Enabled: true, HostIP: "192.168.0.1"})
	if !errors.Is(err, ErrDMZHostIsLAN) {
		t.Fatalf("err = %v", err)
	}
}
