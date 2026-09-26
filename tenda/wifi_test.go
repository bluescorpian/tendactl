package tenda

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiBasic(t *testing.T) {
	r := tendatest.New(t)
	b, err := newTestClient(t, r).WiFiBasic(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !b.Band24.Enabled || b.Band5.Enabled || b.Band24.SSID != "WiFi-16" || b.Band5.SSID != "WiFi-17" {
		t.Fatalf("b = %+v", b)
	}
	if b.Antijam != "auto" {
		t.Fatalf("antijam = %q, want auto", b.Antijam)
	}
}

func TestWiFiBasicSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	prev := WiFiBasic{Band24: WiFiRadio{Enabled: true, SSID: "old", Security: "wpa2psk"}, Band5: WiFiRadio{Enabled: true, SSID: "old5", Security: "wpa2psk"}}
	next := prev
	next.Band24.SSID = "new"
	if err := c.SetWiFiBasic(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"wrlEn": {"1"}, "wrlEn_5g": {"1"},
		"security": {"wpa2psk"}, "security_5g": {"wpa2psk"},
		"ssid": {"new"}, "ssid_5g": {"old5"},
		"hideSsid": {"0"}, "hideSsid_5g": {"0"},
		"wrlPwd": {""}, "wrlPwd_5g": {""},
	}.Encode()
	if got := r.LastCall(t, "WifiBasicSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWiFiBasicSetDisableNeedsConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r) // no WithConfirm: every hazard is refused
	prev := WiFiBasic{Band24: WiFiRadio{Enabled: true}, Band5: WiFiRadio{Enabled: false}}
	next := prev
	next.Band24.Enabled = false
	err := c.SetWiFiBasic(context.Background(), prev, next)
	var he *HazardError
	if !errors.As(err, &he) || !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("err = %v, want *HazardError", err)
	}
	if len(r.CallsTo("WifiBasicSet")) != 0 {
		t.Fatal("request sent")
	}
}

func TestWiFiBasicSetDisableAlreadyOffNeedsNoConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	prev := WiFiBasic{Band24: WiFiRadio{Enabled: false}, Band5: WiFiRadio{Enabled: false}}
	next := prev // resubmitting the same off state
	if err := c.SetWiFiBasic(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
}
