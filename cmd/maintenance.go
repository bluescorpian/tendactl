package cmd

import (
	"fmt"
	"io"
	"regexp"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("system", newMaintenanceCmd) }

var maintenanceTimeRe = regexp.MustCompile(`^\d{2}:\d{2}$`)

func newMaintenanceCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		m, err := c.Maintenance(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, m, func(w io.Writer) error { return maintenanceText(w, m) })
	}
	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "Manage the scheduled reboot (Automatic Maintenance)",
		Long:  `Manage "Automatic Maintenance" (System Settings): a reboot at a fixed daily time.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the automatic maintenance schedule",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var timeOfDay, delay string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the reboot time or delay-if-busy option",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "time", "delay") {
				return errNothingToChange
			}
			if changed(cmd, "time") && !maintenanceTimeRe.MatchString(timeOfDay) {
				return fmt.Errorf("invalid --time %q: want HH:MM", timeOfDay)
			}
			var delayOn bool
			if changed(cmd, "delay") {
				switch delay {
				case "on":
					delayOn = true
				case "off":
				default:
					return fmt.Errorf("invalid --delay %q: want on or off", delay)
				}
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Maintenance, c.SetMaintenance, func(m *tenda.Maintenance) error {
				if changed(cmd, "time") {
					m.RebootTime = timeOfDay
				}
				if changed(cmd, "delay") {
					m.DelayIfBusy = delayOn
				}
				return nil
			}, "Automatic maintenance schedule updated")
		},
	}
	set.Flags().StringVar(&timeOfDay, "time", "", "daily reboot time, as HH:MM")
	set.Flags().StringVar(&delay, "delay", "", "delay the reboot while traffic is above ~3 KB/s: on or off")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn on the scheduled reboot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Maintenance, c.SetMaintenance, func(m *tenda.Maintenance) error { m.Enabled = true; return nil }, "Automatic maintenance enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn off the scheduled reboot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Maintenance, c.SetMaintenance, func(m *tenda.Maintenance) error { m.Enabled = false; return nil }, "Automatic maintenance disabled")
		},
	})
	return cmd
}

func maintenanceText(w io.Writer, m tenda.Maintenance) error {
	return fields(w,
		"Enabled", onOff(m.Enabled),
		"Reboot time", m.RebootTime,
		"Delay if busy", onOff(m.DelayIfBusy),
		"Clock synced", onOff(m.TimeSynced),
	)
}
