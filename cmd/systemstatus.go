package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("system", newSystemStatusCmd) }

func newSystemStatusCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the System Status page",
		Long:  `Show "System Status" (System Settings): uptime, firmware, WAN and WiFi status.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			s, err := c.SystemStatus(cmd.Context())
			if err != nil {
				return err
			}
			return a.render(cmd, s, func(w io.Writer) error { return systemStatusText(w, s) })
		},
	}
}

func systemStatusText(w io.Writer, s tenda.SystemStatus) error {
	kv := []string{
		"System time", s.Time,
		"Uptime (s)", fmt.Sprint(s.UptimeSeconds),
		"Firmware", s.Firmware,
		"Hardware", s.Hardware,
		"LAN IP", s.LANIP,
		"LAN mask", s.LANMask,
		"LAN MAC", s.LANMAC,
		"2.4 GHz SSID", s.WiFi24.SSID,
		"2.4 GHz hidden", onOff(s.WiFi24.Hidden),
		"5 GHz", onOff(s.WiFi5Enabled),
	}
	if s.WiFi5Enabled {
		kv = append(kv, "5 GHz SSID", s.WiFi5.SSID, "5 GHz hidden", onOff(s.WiFi5.Hidden))
	}
	for i, wan := range s.WAN {
		status := tenda.SystemWANConnectStatusText(wan.ConnectStatus)
		connType := tenda.SystemWANConnectTypeText(wan.ConnectType)
		kv = append(kv,
			fmt.Sprintf("WAN%d status", i+1), status,
			fmt.Sprintf("WAN%d type", i+1), connType,
			fmt.Sprintf("WAN%d IP", i+1), wan.IP,
			fmt.Sprintf("WAN%d speed", i+1), fmt.Sprintf("↓%s ↑%s KB/s", wan.DownKBps, wan.UpKBps),
		)
	}
	kv = append(kv,
		"Automatic maintenance", onOff(s.AutoMaintenanceEnabled),
		"Remote management", onOff(s.RemoteManagementEnabled),
		"DHCP reservation configured", onOff(s.DHCPReservationConfigured),
		"Clock synced", onOff(s.TimeSynced),
	)
	return fields(w, kv...)
}
