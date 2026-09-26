package tenda

import (
	"context"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestRouterStatus(t *testing.T) {
	r := tendatest.New(t)
	s, err := newTestClient(t, r).RouterStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.DeviceName != "AC10" || s.ClientCount != 13 || !s.WiFi24.Enabled || s.WiFi24.SSID != "WiFi-16" || s.WiFi5.Enabled {
		t.Fatalf("status = %+v", s)
	}
	if len(s.WAN) != 1 || s.WAN[0].IP != "203.0.113.10" || s.WAN[0].DownKBps != "521.28" {
		t.Fatalf("wan = %+v", s.WAN)
	}
	if s.Firmware.Current != "V15.03.06.50_multi" || s.Firmware.UpdateAvailable {
		t.Fatalf("firmware = %+v", s.Firmware)
	}
}
