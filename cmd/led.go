package cmd

import (
	"fmt"
	"io"
	"regexp"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newLEDCmd) }

var ledTimeRe = regexp.MustCompile(`^\d{2}:\d{2}-\d{2}:\d{2}$`)

func newLEDCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		l, err := c.LED(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, l, func(w io.Writer) error { return ledText(w, l) })
	}
	cmd := &cobra.Command{
		Use:   "led",
		Short: "Manage the status LEDs",
		Long:  `Manage the router's status LEDs (System Settings, "LED Control"). Cosmetic only.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the LED control mode",
		Args:  cobra.NoArgs,
		RunE:  show,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn the LEDs always on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.LED, c.SetLED, func(l *tenda.LED) error { l.Mode = "open"; return nil }, "LEDs set to always on")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn the LEDs always off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.LED, c.SetLED, func(l *tenda.LED) error { l.Mode = "close"; return nil }, "LEDs set to always off")
		},
	})

	var mode, timeWindow, closeType string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the LED control mode or schedule",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "mode", "time", "close-type") {
				return errNothingToChange
			}
			if changed(cmd, "mode") {
				switch mode {
				case "open", "close", "time":
				default:
					return fmt.Errorf("invalid --mode %q: want open, close or time", mode)
				}
			}
			if changed(cmd, "time") && !ledTimeRe.MatchString(timeWindow) {
				return fmt.Errorf("invalid --time %q: want HH:MM-HH:MM", timeWindow)
			}
			if changed(cmd, "close-type") {
				switch closeType {
				case "allClose", "unpowerClose":
				default:
					return fmt.Errorf("invalid --close-type %q: want allClose or unpowerClose", closeType)
				}
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.LED, c.SetLED, func(l *tenda.LED) error {
				if changed(cmd, "mode") {
					l.Mode = mode
				}
				if changed(cmd, "time") {
					l.Time = timeWindow
				}
				if changed(cmd, "close-type") {
					l.CloseType = closeType
				}
				return nil
			}, "LED settings updated")
		},
	}
	set.Flags().StringVar(&mode, "mode", "", "LED control mode: open, close or time")
	set.Flags().StringVar(&timeWindow, "time", "", "off window when --mode time, as HH:MM-HH:MM")
	set.Flags().StringVar(&closeType, "close-type", "", "which LEDs a scheduled/always-off window covers: allClose or unpowerClose")
	cmd.AddCommand(set)
	return cmd
}

func ledText(w io.Writer, l tenda.LED) error {
	return fields(w, "Mode", ledModeLabel(l.Mode), "Off window", l.Time, "Close type", ledCloseTypeLabel(l.CloseType))
}
