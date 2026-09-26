package tenda

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestLAN(t *testing.T) {
	r := tendatest.New(t)
	l, err := newTestClient(t, r).LAN(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if l.LANIP != "192.168.0.1" || l.LANMask != "255.255.255.0" || !l.DHCPEnabled {
		t.Fatalf("l = %+v", l)
	}
	if l.DHCPStart != "192.168.0.100" || l.DHCPEnd != "192.168.0.200" || l.LeaseTime != "604800" {
		t.Fatalf("l = %+v", l)
	}
	if l.DNSAuto || l.DNS1 != "1.1.1.1" || l.DNS2 != "8.8.8.8" {
		t.Fatalf("l = %+v", l)
	}
}

func TestLANSet(t *testing.T) {
	r := tendatest.New(t)
	r.Allow("AdvSetLanip")
	c := newTestClient(t, r, WithConfirm(allowAll))
	l := LAN{LANIP: "192.168.0.1", LANMask: "255.255.255.0", DHCPEnabled: true, DHCPStart: "192.168.0.100", DHCPEnd: "192.168.0.200", LeaseTime: "604800", DNSAuto: false, DNS1: "1.1.1.1", DNS2: "8.8.8.8"}
	if err := c.SetLAN(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"lanIp": {"192.168.0.1"}, "lanMask": {"255.255.255.0"}, "dhcpEn": {"1"},
		"startIp": {"192.168.0.100"}, "endIp": {"192.168.0.200"}, "leaseTime": {"604800"},
		"lanDnsAuto": {"0"}, "lanDns1": {"1.1.1.1"}, "lanDns2": {"8.8.8.8"},
	}.Encode()
	if got := r.LastCall(t, "AdvSetLanip").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestLANSetNeedsConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r) // no WithConfirm: every hazard is refused
	l, err := c.LAN(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = c.SetLAN(context.Background(), l)
	var he *HazardError
	if !errors.As(err, &he) || !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("err = %v, want *HazardError", err)
	}
	if len(r.CallsTo("AdvSetLanip")) != 0 {
		t.Fatal("request sent")
	}
}
