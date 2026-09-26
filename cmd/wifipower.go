package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("wifi", newWiFiPowerCmd) }

func newWiFiPowerCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		p, err := c.WiFiPower(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, p, func(w io.Writer) error { return wifiPowerText(w, p) })
	}
	cmd := &cobra.Command{
		Use:   "power",
		Short: "Manage WiFi transmit power",
		Long:  `Manage "Transmit Power" (WiFi Settings). Lowering power reduces WiFi range and may drop marginal clients.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show WiFi transmit power",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var band tenda.Band
	set := &cobra.Command{
		Use:   "set <low|mid|high>",
		Short: "Set WiFi transmit power",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			level, err := wifiPowerLevel(args[0])
			if err != nil {
				return err
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.WiFiPower, c.SetWiFiPower, func(p *tenda.WiFiPower) error {
				if band.Has24() {
					p.Power = level
				}
				if band.Has5() {
					p.Power5g = level
				}
				return nil
			}, "WiFi transmit power set to "+level)
		},
	}
	set.Flags().Var(&band, "band", "band to change: 2.4, 5 or all")
	cmd.AddCommand(set)
	return cmd
}

// wifiPowerLevel accepts the doc's wire values plus "mid" as a shorthand for
// "middle".
func wifiPowerLevel(s string) (string, error) {
	switch s {
	case "low", "high":
		return s, nil
	case "mid", "middle":
		return "middle", nil
	default:
		return "", fmt.Errorf("invalid power level %q: want low, mid or high", s)
	}
}

func wifiPowerText(w io.Writer, p tenda.WiFiPower) error {
	return fields(w, "2.4 GHz power", p.Power, "5 GHz power", p.Power5g)
}
