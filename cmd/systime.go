package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("system", newSysTimeCmd) }

var sysTimeZoneRe = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)

// sysTimeValidZone checks the doc's range: "0:00" (GMT-12) to "25:00"
// (GMT+13), plus the ":10" Russia/Ukraine duplicates.
func sysTimeValidZone(z string) bool {
	m := sysTimeZoneRe.FindStringSubmatch(z)
	if m == nil {
		return false
	}
	h, _ := strconv.Atoi(m[1])
	min := m[2]
	if h < 0 || h > 25 {
		return false
	}
	return min == "00" || min == "30" || min == "10"
}

func newSysTimeCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		s, err := c.SysTime(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, s, func(w io.Writer) error { return sysTimeText(w, s) })
	}
	cmd := &cobra.Command{
		Use:   "time",
		Short: "Manage the time zone (Time Settings)",
		Long:  `Manage "Time Settings" (System Settings). The router only syncs via NTP.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the time zone and sync status",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var zone string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the time zone",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "zone") {
				return errNothingToChange
			}
			if !sysTimeValidZone(zone) {
				return fmt.Errorf("invalid --zone %q: want HH:MM, GMT offset plus 12h, 0:00 to 25:00", zone)
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.SysTime, c.SetSysTime, func(s *tenda.SysTime) error {
				s.TimeZone = zone
				return nil
			}, "Time zone updated")
		},
	}
	set.Flags().StringVar(&zone, "zone", "", `time zone, as the GMT offset plus 12h (e.g. "14:00" = GMT+02:00)`)
	cmd.AddCommand(set)
	return cmd
}

// sysTimeZoneLabel renders SetSysTimeCfg's timeZone (the GMT offset plus
// 12h, e.g. "14:00" = GMT+02:00; the ":10" variants duplicate the ":00"
// offset) as "GMT±HH:MM" alongside the raw wire value.
func sysTimeZoneLabel(raw string) string {
	m := sysTimeZoneRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	if min == 10 {
		min = 0
	}
	total := h*60 + min - 12*60
	gmt := "GMT"
	if total != 0 {
		sign := "+"
		if total < 0 {
			sign = "-"
			total = -total
		}
		gmt += fmt.Sprintf("%s%02d:%02d", sign, total/60, total%60)
	}
	return fmt.Sprintf("%s (%s)", gmt, raw)
}

func sysTimeText(w io.Writer, s tenda.SysTime) error {
	synced := "unsynchronized"
	if s.Synced {
		synced = "synchronized with internet time"
	}
	return fields(w,
		"Time zone", sysTimeZoneLabel(s.TimeZone),
		"Current time", s.Time+" ("+synced+")",
		"NTP server", s.NTPServer,
	)
}
