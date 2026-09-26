package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() {
	register("wifi", newWiFiBasicShowCmd)
	register("wifi", newWiFiBasicSetCmd)
	register("wifi", newWiFiBasicEnableCmd)
	register("wifi", newWiFiBasicDisableCmd)
	register("wifi", newWiFiBasicHideCmd)
	register("wifi", newWiFiBasicUnhideCmd)
}

// wifiBasicSecurities are WifiBasicSet's security enum.
var wifiBasicSecurities = []string{"none", "wpapsk", "wpa2psk", "wpawpa2psk"}

func newWiFiBasicShowCmd(a *app) *cobra.Command {
	var showPassword bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show WiFi name, password and status",
		Long:  `Show "WiFi Name & Password" for both bands, plus the WiFi Settings tab's tile status.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			b, err := c.WiFiBasic(cmd.Context())
			if err != nil {
				return err
			}
			b.Band24.Password = maskSecret(b.Band24.Password, showPassword)
			b.Band5.Password = maskSecret(b.Band5.Password, showPassword)
			return a.render(cmd, b, func(w io.Writer) error { return wifiBasicText(w, b) })
		},
	}
	cmd.Flags().BoolVar(&showPassword, "show-password", false, "show WiFi passwords instead of masking them")
	return cmd
}

func newWiFiBasicSetCmd(a *app) *cobra.Command {
	var band tenda.Band
	var ssid, password, security string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set WiFi name, password or encryption",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "ssid", "password", "security") {
				return errNothingToChange
			}
			if changed(cmd, "security") && !wifiBasicValidSecurity(security) {
				return fmt.Errorf("invalid --security %q: want one of %v", security, wifiBasicSecurities)
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			prev, err := c.WiFiBasic(ctx)
			if err != nil {
				return err
			}
			next := prev
			apply := func(r *tenda.WiFiRadio) {
				if changed(cmd, "ssid") {
					r.SSID = ssid
				}
				if changed(cmd, "password") {
					r.Password = password
				}
				if changed(cmd, "security") {
					r.Security = security
				}
			}
			if band.Has24() {
				apply(&next.Band24)
			}
			if band.Has5() {
				apply(&next.Band5)
			}
			if err := c.SetWiFiBasic(ctx, prev, next); err != nil {
				return err
			}
			return a.done(cmd, "WiFi settings updated")
		},
	}
	cmd.Flags().Var(&band, "band", "band to change: 2.4, 5 or all")
	cmd.Flags().StringVar(&ssid, "ssid", "", "WiFi network name")
	cmd.Flags().StringVar(&password, "password", "", "WiFi password")
	cmd.Flags().StringVar(&security, "security", "", "encryption: none, wpapsk, wpa2psk or wpawpa2psk")
	return cmd
}

func newWiFiBasicEnableCmd(a *app) *cobra.Command {
	var band tenda.Band
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Turn a WiFi band on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wifiBasicApply(a, cmd, band, func(r *tenda.WiFiRadio) { r.Enabled = true }, "WiFi enabled")
		},
	}
	cmd.Flags().Var(&band, "band", "band to enable: 2.4, 5 or all")
	return cmd
}

func newWiFiBasicDisableCmd(a *app) *cobra.Command {
	var band tenda.Band
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Turn a WiFi band off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wifiBasicApply(a, cmd, band, func(r *tenda.WiFiRadio) { r.Enabled = false }, "WiFi disabled")
		},
	}
	cmd.Flags().Var(&band, "band", "band to disable: 2.4, 5 or all")
	return cmd
}

func newWiFiBasicHideCmd(a *app) *cobra.Command {
	var band tenda.Band
	cmd := &cobra.Command{
		Use:   "hide",
		Short: "Hide a WiFi network's SSID broadcast",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wifiBasicApply(a, cmd, band, func(r *tenda.WiFiRadio) { r.Hidden = true }, "WiFi SSID hidden")
		},
	}
	cmd.Flags().Var(&band, "band", "band to hide: 2.4, 5 or all")
	return cmd
}

func newWiFiBasicUnhideCmd(a *app) *cobra.Command {
	var band tenda.Band
	cmd := &cobra.Command{
		Use:   "unhide",
		Short: "Broadcast a WiFi network's SSID again",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wifiBasicApply(a, cmd, band, func(r *tenda.WiFiRadio) { r.Hidden = false }, "WiFi SSID unhidden")
		},
	}
	cmd.Flags().Var(&band, "band", "band to unhide: 2.4, 5 or all")
	return cmd
}

// wifiBasicApply is the Before-aware read-modify-write helper for wifi
// enable/disable/hide/unhide: WifiBasicSet's hazard rule needs the previous
// value, so it cannot go through the generic update[T].
func wifiBasicApply(a *app, cmd *cobra.Command, band tenda.Band, mutate func(*tenda.WiFiRadio), done string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	prev, err := c.WiFiBasic(ctx)
	if err != nil {
		return err
	}
	next := prev
	if band.Has24() {
		mutate(&next.Band24)
	}
	if band.Has5() {
		mutate(&next.Band5)
	}
	if err := c.SetWiFiBasic(ctx, prev, next); err != nil {
		return err
	}
	return a.done(cmd, "%s", done)
}

func wifiBasicValidSecurity(s string) bool {
	for _, v := range wifiBasicSecurities {
		if s == v {
			return true
		}
	}
	return false
}

func wifiBasicText(w io.Writer, b tenda.WiFiBasic) error {
	return fields(w,
		"2.4 GHz", onOff(b.Band24.Enabled),
		"2.4 GHz SSID", b.Band24.SSID,
		"2.4 GHz hidden", onOff(b.Band24.Hidden),
		"2.4 GHz security", securityLabel(b.Band24.Security),
		"2.4 GHz password", b.Band24.Password,
		"5 GHz", onOff(b.Band5.Enabled),
		"5 GHz SSID", b.Band5.SSID,
		"5 GHz hidden", onOff(b.Band5.Hidden),
		"5 GHz security", securityLabel(b.Band5.Security),
		"5 GHz password", b.Band5.Password,
		"Schedule", onOff(b.ScheduleOn),
		"WPS", onOff(b.WPSOn),
		"Beamforming", onOff(b.BeamformingOn),
		"AP mode", onOff(b.APModeOn),
		"Wireless repeating", onOff(b.RepeatingOn),
		"Anti-interference", b.Antijam,
	)
}
