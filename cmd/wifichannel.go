package cmd

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("wifi", newWiFiChannelCmd) }

var (
	wifiChannelModes24  = []string{"bgn", "bg", "n only"}
	wifiChannelModes5   = []string{"ac", "ac only"}
	wifiChannelWidths24 = []string{"20", "40", "auto"}
	wifiChannelWidths5  = []string{"20", "40", "80", "auto"}
)

func newWiFiChannelCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		ch, err := c.WiFiChannel(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, ch, func(w io.Writer) error { return wifiChannelText(w, ch) })
	}
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Manage WiFi channel, bandwidth and mode",
		Long:  `Manage "Channel & Bandwidth" (WiFi Settings). Changing a band's settings restarts that radio, disconnecting its clients.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show WiFi channel, bandwidth and mode",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var band tenda.Band
	var channel, width, mode string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set WiFi channel, bandwidth or mode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if band == tenda.BandAll {
				return fmt.Errorf("--band is required: 2.4 or 5")
			}
			if !changed(cmd, "channel", "width", "mode") {
				return errNothingToChange
			}
			modes, widths := wifiChannelModes24, wifiChannelWidths24
			if band == tenda.Band5 {
				modes, widths = wifiChannelModes5, wifiChannelWidths5
			}
			if changed(cmd, "mode") && !slices.Contains(modes, mode) {
				return fmt.Errorf("invalid --mode %q: want one of %v", mode, modes)
			}
			if changed(cmd, "width") && !slices.Contains(widths, width) {
				return fmt.Errorf("invalid --width %q: want one of %v", width, widths)
			}
			var ch int
			if changed(cmd, "channel") {
				var err error
				if ch, err = wifiChannelParse(channel); err != nil {
					return err
				}
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.WiFiChannel, c.SetWiFiChannel, func(v *tenda.WiFiChannel) error {
				r := &v.Band24
				if band == tenda.Band5 {
					r = &v.Band5
				}
				if changed(cmd, "channel") {
					if ch != 0 && !slices.Contains(r.Channels, ch) {
						return fmt.Errorf("invalid --channel %d: want auto or one of %v", ch, r.Channels)
					}
					r.Channel = ch
				}
				if changed(cmd, "width") {
					r.Width = width
				}
				if changed(cmd, "mode") {
					r.Mode = mode
				}
				return nil
			}, "WiFi channel settings updated")
		},
	}
	set.Flags().Var(&band, "band", "band to change: 2.4 or 5 (required)")
	set.Flags().StringVar(&channel, "channel", "", "channel number, or auto")
	set.Flags().StringVar(&width, "width", "", "bandwidth: 20, 40, 80 (5 GHz only) or auto")
	set.Flags().StringVar(&mode, "mode", "", `network mode: bgn, bg or "n only" (2.4 GHz); ac or "ac only" (5 GHz)`)
	cmd.AddCommand(set)
	return cmd
}

func wifiChannelParse(s string) (int, error) {
	if strings.EqualFold(s, "auto") {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --channel %q: want a channel number or auto", s)
	}
	return n, nil
}

func wifiChannelText(w io.Writer, ch tenda.WiFiChannel) error {
	band := func(name string, r tenda.RadioBand) []string {
		channel := "auto"
		if r.Channel != 0 {
			channel = strconv.Itoa(r.Channel)
		}
		return []string{name + " mode", r.Mode, name + " channel", channel, name + " width", r.Width, name + " country", r.Country}
	}
	kv := append(band("2.4 GHz", ch.Band24), band("5 GHz", ch.Band5)...)
	return fields(w, kv...)
}
