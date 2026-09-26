package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestFirewall(t *testing.T) {
	r := tendatest.New(t)
	f, err := newTestClient(t, r).Firewall(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := Firewall{ICMPFloodDefense: true, TCPFloodDefense: true, UDPFloodDefense: true, IgnoreWANPing: false}
	if f != want {
		t.Fatalf("f = %+v, want %+v", f, want)
	}
}

func TestFirewallSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetFirewall(context.Background(), Firewall{ICMPFloodDefense: true, TCPFloodDefense: false, UDPFloodDefense: true, IgnoreWANPing: true}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"firewallEn": {"1011"}}.Encode()
	if got := r.LastCall(t, "SetFirewallCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestFirewallDecodeError(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetFirewallCfg", 200, `{"firewallEn":"11"}`)
	if _, err := newTestClient(t, r).Firewall(context.Background()); err == nil {
		t.Fatal("want error")
	}
}
