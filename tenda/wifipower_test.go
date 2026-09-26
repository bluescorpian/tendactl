package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiPower(t *testing.T) {
	r := tendatest.New(t)
	p, err := newTestClient(t, r).WiFiPower(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if p.Power != "high" || p.Power5g != "high" {
		t.Fatalf("p = %+v", p)
	}
}

func TestWiFiPowerSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetWiFiPower(context.Background(), WiFiPower{Power: "low", Power5g: "middle"}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"power": {"low"}, "power_5g": {"middle"}}.Encode()
	if got := r.LastCall(t, "WifiPowerSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
