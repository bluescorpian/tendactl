package cmd

import (
	"fmt"
	"io"
	"regexp"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newSleepCmd) }

var sleepTimeRe = regexp.MustCompile(`^\d{2}:\d{2}-\d{2}:\d{2}$`)

func newSleepCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		s, err := c.Sleep(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, s, func(w io.Writer) error { return sleepText(w, s) })
	}
	cmd := &cobra.Command{
		Use:   "sleep",
		Short: "Manage Sleeping Mode",
		Long: `Manage "Sleeping Mode" (Advanced Settings). Enabling it turns WiFi (and the
LEDs) off during the configured window; the documented way to wake it early
is the router's WiFi button or the Tenda App. Needs the router in AP mode.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show Sleeping Mode",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var timeWindow, leds, delay string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set Sleeping Mode's window, LED behaviour or delay",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "time", "leds", "delay") {
				return errNothingToChange
			}
			if changed(cmd, "time") {
				if !sleepTimeRe.MatchString(timeWindow) {
					return fmt.Errorf("invalid --time %q: want HH:MM-HH:MM", timeWindow)
				}
			}
			var ledsWire string
			if changed(cmd, "leds") {
				var err error
				if ledsWire, err = sleepLEDsWire(leds); err != nil {
					return err
				}
			}
			var delayOn bool
			if changed(cmd, "delay") {
				var err error
				if delayOn, err = sleepParseOnOff(delay); err != nil {
					return err
				}
			}
			return sleepApply(a, cmd, func(s *tenda.Sleep) {
				if changed(cmd, "time") {
					s.Window = timeWindow
				}
				if changed(cmd, "leds") {
					s.LEDs = ledsWire
				}
				if changed(cmd, "delay") {
					s.Delay = delayOn
				}
			}, "Sleeping Mode settings updated")
		},
	}
	set.Flags().StringVar(&timeWindow, "time", "", "sleep window, as HH:MM-HH:MM")
	set.Flags().StringVar(&leds, "leds", "", "which indicators sleep during the window: all or except-power")
	set.Flags().StringVar(&delay, "delay", "", "delay enabling while a client is online: on or off")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn Sleeping Mode on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return sleepApply(a, cmd, func(s *tenda.Sleep) { s.Enabled = true }, "Sleeping Mode enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn Sleeping Mode off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return sleepApply(a, cmd, func(s *tenda.Sleep) { s.Enabled = false }, "Sleeping Mode disabled")
		},
	})
	return cmd
}

// sleepApply is the Before-aware read-modify-write helper: PowerSaveSet's
// hazard rule needs the previous value, so it cannot go through the generic
// update[T].
func sleepApply(a *app, cmd *cobra.Command, mutate func(*tenda.Sleep), done string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	prev, err := c.Sleep(ctx)
	if err != nil {
		return err
	}
	next := prev
	mutate(&next)
	if err := c.SetSleep(ctx, prev, next); err != nil {
		return err
	}
	return a.done(cmd, "%s", done)
}

func sleepLEDsWire(s string) (string, error) {
	switch s {
	case "all":
		return "allClose", nil
	case "except-power":
		return "unpowerClose", nil
	default:
		return "", fmt.Errorf("invalid --leds %q: want all or except-power", s)
	}
}

func sleepLEDsLabel(s string) string {
	if s == "unpowerClose" {
		return "all except power"
	}
	return "all"
}

func sleepParseOnOff(s string) (bool, error) {
	switch s {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q: want on or off", s)
	}
}

func sleepText(w io.Writer, s tenda.Sleep) error {
	return fields(w,
		"Enabled", onOff(s.Enabled),
		"Window", s.Window,
		"LEDs off", sleepLEDsLabel(s.LEDs),
		"Delay while online", onOff(s.Delay),
		"Router in AP mode", onOff(s.WorkModeOK),
		"LED Control window", s.LEDWindow,
		"WiFi Schedule window", s.WiFiScheduleWindow,
		"Clock synced", onOff(s.TimeSynced),
	)
}
