package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestDDNS(t *testing.T) {
	r := tendatest.New(t)
	d, err := newTestClient(t, r).DDNS(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Enabled || d.Provider != "no-ip.com" || d.Connected {
		t.Fatalf("d = %+v", d)
	}
}

func TestDDNSSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	d := DDNS{Enabled: true, Provider: "no-ip.com", User: "bob", Password: "s3cret", Domain: "bob.example.com"}
	if err := c.SetDDNS(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"ddnsEn": {"1"}, "serverName": {"no-ip.com"}, "ddnsUser": {"bob"}, "ddnsPwd": {"s3cret"}, "ddnsDomain": {"bob.example.com"},
	}.Encode()
	if got := r.LastCall(t, "SetDDNSCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
