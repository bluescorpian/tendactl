package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newGuestCmd) }

func newGuestCmd(a *app) *cobra.Command {
	var showPassword bool
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		g, err := c.Guest(cmd.Context())
		if err != nil {
			return err
		}
		g.Password = maskSecret(g.Password, showPassword)
		return a.render(cmd, g, func(w io.Writer) error { return guestText(w, g) })
	}
	cmd := &cobra.Command{
		Use:   "guest",
		Short: "Manage the guest WiFi network",
		Long:  `Manage the "Guest Network". It needs the main radio in AP mode and enabled.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.Flags().BoolVar(&showPassword, "show-password", false, "show the guest password instead of masking it")
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the guest network",
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	showCmd.Flags().BoolVar(&showPassword, "show-password", false, "show the guest password instead of masking it")
	cmd.AddCommand(showCmd)

	var band tenda.Band
	var ssid, password, effectiveTime string
	var shareSpeed int
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the guest network's name, password, validity or bandwidth cap",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "ssid", "password", "effective-time", "share-speed") {
				return errNothingToChange
			}
			wireTime, err := guestEffectiveTimeWire(effectiveTime)
			if changed(cmd, "effective-time") && err != nil {
				return err
			}
			if changed(cmd, "share-speed") && shareSpeed < 0 {
				return fmt.Errorf("invalid --share-speed %d: want 0 (unlimited) or a positive Mbps value", shareSpeed)
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Guest, c.SetGuest, func(g *tenda.Guest) error {
				if changed(cmd, "ssid") {
					if band.Has24() {
						g.SSID = ssid
					}
					if band.Has5() {
						g.SSID5g = ssid
					}
				}
				if changed(cmd, "password") {
					g.Password = password
				}
				if changed(cmd, "effective-time") {
					g.EffectiveTime = wireTime
				}
				if changed(cmd, "share-speed") {
					g.ShareSpeedMbps = shareSpeed
				}
				return nil
			}, "Guest network settings updated")
		},
	}
	set.Flags().Var(&band, "band", "band the --ssid applies to: 2.4, 5 or all")
	set.Flags().StringVar(&ssid, "ssid", "", "guest WiFi network name")
	set.Flags().StringVar(&password, "password", "", "guest WiFi password (shared by both bands)")
	set.Flags().StringVar(&effectiveTime, "effective-time", "", "guest network validity: 4, 8 (hours) or always")
	set.Flags().IntVar(&shareSpeed, "share-speed", 0, "shared bandwidth cap for guests, in Mbps (0 = unlimited)")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn the guest network on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Guest, c.SetGuest, func(g *tenda.Guest) error { g.Enabled = true; return nil }, "Guest network enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn the guest network off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Guest, c.SetGuest, func(g *tenda.Guest) error { g.Enabled = false; return nil }, "Guest network disabled")
		},
	})
	return cmd
}

// guestEffectiveTimeWire accepts the doc's wire values plus "always" as a
// shorthand for "0".
func guestEffectiveTimeWire(s string) (string, error) {
	switch s {
	case "4", "8", "0":
		return s, nil
	case "always":
		return "0", nil
	default:
		return "", fmt.Errorf("invalid --effective-time %q: want 4, 8 or always", s)
	}
}

func guestEffectiveTimeLabel(s string) string {
	switch s {
	case "4":
		return "4 hours"
	case "8":
		return "8 hours"
	default:
		return "always"
	}
}

// guestShareSpeedLabel renders WifiGuestSet's shareSpeed (Mbps, 0 =
// unlimited) as the UI's "Shared Bandwidth for Guests" text.
func guestShareSpeedLabel(mbps int) string {
	if mbps == 0 {
		return "unlimited"
	}
	return strconv.Itoa(mbps) + " Mbps"
}

func guestText(w io.Writer, g tenda.Guest) error {
	return fields(w,
		"Enabled", onOff(g.Enabled),
		"2.4 GHz SSID", g.SSID,
		"5 GHz SSID", g.SSID5g,
		"Password", g.Password,
		"Validity", guestEffectiveTimeLabel(g.EffectiveTime),
		"Shared bandwidth", guestShareSpeedLabel(g.ShareSpeedMbps),
	)
}
