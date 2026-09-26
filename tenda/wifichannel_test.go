package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWiFiChannel(t *testing.T) {
	r := tendatest.New(t)
	ch, err := newTestClient(t, r).WiFiChannel(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ch.Band24.Mode != "bgn" || ch.Band24.Channel != 0 || ch.Band24.Width != "20" {
		t.Fatalf("band24 = %+v", ch.Band24)
	}
	if ch.Band5.Mode != "ac" || ch.Band5.Width != "auto" {
		t.Fatalf("band5 = %+v", ch.Band5)
	}
	if len(ch.Band24.Channels) != 11 || ch.Band24.Channels[0] != 1 {
		t.Fatalf("band24 channels = %v", ch.Band24.Channels)
	}
	// adv_band_5g is "auto", so the widest reported list (80) is used.
	want5 := []int{36, 40, 44, 48, 149, 153, 157, 161}
	if len(ch.Band5.Channels) != len(want5) {
		t.Fatalf("band5 channels = %v", ch.Band5.Channels)
	}
}

func TestWiFiChannelSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	ch := WiFiChannel{
		Band24: RadioBand{Mode: "bgn", Channel: 6, Width: "20"},
		Band5:  RadioBand{Mode: "ac", Channel: 0, Width: "auto"},
	}
	if err := c.SetWiFiChannel(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"adv_mode": {"bgn"}, "adv_channel": {"6"}, "adv_band": {"20"},
		"adv_mode_5g": {"ac"}, "adv_channel_5g": {"0"}, "adv_band_5g": {"auto"},
	}.Encode()
	if got := r.LastCall(t, "WifiRadioSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
