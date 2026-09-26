package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestGuest(t *testing.T) {
	r := tendatest.New(t)
	g, err := newTestClient(t, r).Guest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !g.Enabled || g.SSID != "WiFi-18" || g.SSID5g != "WiFi-18" || g.ShareSpeedMbps != 0 {
		t.Fatalf("g = %+v", g)
	}
}

func TestGuestSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	g := Guest{Enabled: true, SSID: "Guest24", SSID5g: "Guest5", Password: "secretpw", EffectiveTime: "8", ShareSpeedMbps: 2}
	if err := c.SetGuest(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"guestEn": {"1"}, "guestEn_5g": {"1"},
		"guestSecurity": {"wpapsk"}, "guestSecurity_5g": {"wpapsk"},
		"guestSsid": {"Guest24"}, "guestSsid_5g": {"Guest5"},
		"guestWrlPwd": {"secretpw"}, "guestWrlPwd_5g": {"secretpw"},
		"effectiveTime": {"8"}, "shareSpeed": {"256"},
	}.Encode()
	if got := r.LastCall(t, "WifiGuestSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestGuestSetBlankPasswordStillSendsWpapsk(t *testing.T) {
	// The UI's own bug (docs/router-api.md): guestSecurity is unconditionally
	// "wpapsk", even with a blank password, because it reads a nonexistent
	// #wrlPwd element. This test pins that so tendactl matches the UI.
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetGuest(context.Background(), Guest{}); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "WifiGuestSet").Form.Get("guestSecurity"); got != "wpapsk" {
		t.Fatalf("guestSecurity = %q, want wpapsk", got)
	}
}
