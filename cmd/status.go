package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newStatusCmd) }

func newStatusCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the router's internet status, WiFi, clients and firmware",
		Long: `Show the dashboard's "Internet Status": WAN IP and current speed, the
2.4/5 GHz networks, the number of online clients and the firmware version.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			s, err := c.RouterStatus(cmd.Context())
			if err != nil {
				return err
			}
			return a.render(cmd, s, func(w io.Writer) error { return statusText(w, s) })
		},
	}
}

// statusText is byte-identical to the pre-refactor `tendactl status`
// (cmd/testdata/status_legacy.golden).
func statusText(w io.Writer, d tenda.RouterStatus) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%s) @ %s\n", d.DeviceName, d.WorkMode, d.LANIP)

	wifi := "[WIFI]"
	if d.WiFi5.Enabled || d.WiFi24.Enabled {
		wifi = fmt.Sprintf("5G:%s 2.4G:%s", statusBand(d.WiFi5), statusBand(d.WiFi24))
	}
	wan := ""
	if len(d.WAN) > 0 {
		wan = fmt.Sprintf("WAN: %s (Down:%s KB/s Up:%s KB/s)", d.WAN[0].IP, d.WAN[0].DownKBps, d.WAN[0].UpKBps)
	}
	fmt.Fprintf(&sb, "%-40s %s\n", wifi, wan)
	fmt.Fprintf(&sb, "Clients: %-4d  MAC: %s\n", d.ClientCount, d.LANMAC)
	fmt.Fprintf(&sb, "Firmware: %s", d.Firmware.Current)
	if d.Firmware.UpdateAvailable {
		fmt.Fprintf(&sb, " → Update Available: %s", d.Firmware.Latest)
	}
	sb.WriteString("\n")
	_, err := io.WriteString(w, sb.String())
	return err
}

func statusBand(b tenda.WiFiBand) string {
	if b.Enabled {
		return "[ON]" + b.SSID
	}
	return "[OFF]" + b.SSID
}
