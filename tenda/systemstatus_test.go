package tenda

import (
	"context"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSystemStatus(t *testing.T) {
	r := tendatest.New(t)
	s, err := newTestClient(t, r).SystemStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Firmware != "V15.03.06.50_multi" || s.Hardware != "V1.0" || s.UptimeSeconds != 630448 {
		t.Fatalf("s = %+v", s)
	}
	if len(s.WAN) != 1 || s.WAN[0].IP != "203.0.113.10" || s.WAN[0].ConnectStatus != 3 {
		t.Fatalf("wan = %+v", s.WAN)
	}
	if s.WiFi24.SSID != "WiFi-16" || s.WiFi24.Security != "wpa2psk" {
		t.Fatalf("wifi24 = %+v", s.WiFi24)
	}
	if s.WiFi5Enabled || s.WiFi5.SSID != "" {
		t.Fatalf("wifi5 = %+v, enabled = %v", s.WiFi5, s.WiFi5Enabled)
	}
	if !s.AutoMaintenanceEnabled || s.RemoteManagementEnabled || !s.DHCPReservationConfigured || s.TimeSynced {
		t.Fatalf("s = %+v", s)
	}
}

func TestSystemWANConnectStatusText(t *testing.T) {
	if SystemWANConnectStatusText(3) != "Connected" || SystemWANConnectStatusText(0) != "Cable disconnected" {
		t.Fatal("wrong labels")
	}
	if SystemWANConnectStatusText(-1) != "" || SystemWANConnectStatusText(99) != "" {
		t.Fatal("want empty for out of range")
	}
}

func TestSystemWANConnectTypeText(t *testing.T) {
	if SystemWANConnectTypeText("2") != "PPPoE" || SystemWANConnectTypeText("0") != "Dynamic IP Address" {
		t.Fatal("wrong labels")
	}
	if SystemWANConnectTypeText("") != "" || SystemWANConnectTypeText("99") != "" || SystemWANConnectTypeText("x") != "" {
		t.Fatal("want empty for invalid input")
	}
}
