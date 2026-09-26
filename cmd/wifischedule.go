package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("wifi", newWiFiScheduleCmd) }

var wifiScheduleTimeRe = regexp.MustCompile(`^\d{2}:\d{2}-\d{2}:\d{2}$`)

func newWiFiScheduleCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		s, err := c.WiFiSchedule(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, s, func(w io.Writer) error { return wifiScheduleText(w, s) })
	}
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage the WiFi on/off schedule",
		Long: `Manage "WiFi Schedule" (WiFi Settings). Enabling it turns WiFi off during the
configured window; the only recovery inside that window is the router's
physical WiFi button. Requires the router in AP mode.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the WiFi schedule",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var timeWindow, days string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the WiFi schedule's off window or days",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "time", "days") {
				return errNothingToChange
			}
			var start, end string
			if changed(cmd, "time") {
				if !wifiScheduleTimeRe.MatchString(timeWindow) {
					return fmt.Errorf("invalid --time %q: want HH:MM-HH:MM", timeWindow)
				}
				parts := strings.SplitN(timeWindow, "-", 2)
				start, end = parts[0], parts[1]
				if start == end {
					return fmt.Errorf("invalid --time %q: start and end must differ", timeWindow)
				}
			}
			var everyDay bool
			var dayList []string
			if changed(cmd, "days") {
				var err error
				if everyDay, dayList, err = wifiScheduleParseDays(days); err != nil {
					return err
				}
			}
			return wifiScheduleApply(a, cmd, func(s *tenda.WiFiSchedule) {
				if changed(cmd, "time") {
					s.Start, s.End = start, end
				}
				if changed(cmd, "days") {
					s.EveryDay, s.Days = everyDay, dayList
				}
			}, "WiFi schedule updated")
		},
	}
	set.Flags().StringVar(&timeWindow, "time", "", "off window, as HH:MM-HH:MM")
	set.Flags().StringVar(&days, "days", "", "comma-separated days (mon,tue,...) the schedule runs on, or \"all\" for every day")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn the WiFi schedule on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wifiScheduleApply(a, cmd, func(s *tenda.WiFiSchedule) { s.Enabled = true }, "WiFi schedule enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn the WiFi schedule off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wifiScheduleApply(a, cmd, func(s *tenda.WiFiSchedule) { s.Enabled = false }, "WiFi schedule disabled")
		},
	})
	return cmd
}

// wifiScheduleApply is the Before-aware read-modify-write helper:
// openSchedWifi's hazard rule needs the previous value, so it cannot go
// through the generic update[T].
func wifiScheduleApply(a *app, cmd *cobra.Command, mutate func(*tenda.WiFiSchedule), done string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	prev, err := c.WiFiSchedule(ctx)
	if err != nil {
		return err
	}
	next := prev
	mutate(&next)
	if err := c.SetWiFiSchedule(ctx, prev, next); err != nil {
		return err
	}
	return a.done(cmd, "%s", done)
}

func wifiScheduleParseDays(s string) (everyDay bool, days []string, err error) {
	if strings.EqualFold(s, "all") {
		return true, nil, nil
	}
	names := map[string]bool{"mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true, "sun": true}
	for _, d := range strings.Split(s, ",") {
		d = strings.ToLower(strings.TrimSpace(d))
		if !names[d] {
			return false, nil, fmt.Errorf("invalid --days %q: want mon,tue,wed,thu,fri,sat,sun or all", s)
		}
		days = append(days, d)
	}
	return false, days, nil
}

func wifiScheduleText(w io.Writer, s tenda.WiFiSchedule) error {
	days := "every day"
	if !s.EveryDay {
		days = strings.Join(s.Days, ",")
		if days == "" {
			days = "(none selected)"
		}
	}
	return fields(w,
		"Enabled", onOff(s.Enabled),
		"Off window", s.Start+"-"+s.End,
		"Days", days,
		"Sleeping Mode window", s.PowerSaveWindow,
		"Router in AP mode", onOff(s.WorkModeOK),
		"Clock synced", onOff(s.TimeSynced),
	)
}
