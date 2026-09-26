package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newParentalCmd) }

var parentalTimeRe = regexp.MustCompile(`^\d{2}:\d{2}-\d{2}:\d{2}$`)

func newParentalCmd(a *app) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		devices, err := c.ParentalDevices(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, devices, func(w io.Writer) error { return parentalListText(w, devices) })
	}
	cmd := &cobra.Command{
		Use:   "parental",
		Short: "Manage Parental Control",
		Long: `Manage "Parental Control": a per-device schedule and website filter, plus a
quick block/allow toggle independent of that schedule.`,
		Args: cobra.NoArgs,
		RunE: list,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List devices and their parental control status",
		Args:  cobra.NoArgs,
		RunE:  list,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show <mac>",
		Short: "Show a device's parental control rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			r, ok, err := c.ParentalRule(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return a.render(cmd, parentalRuleJSON(r, ok), func(w io.Writer) error { return parentalRuleText(w, r, ok) })
		},
	})

	var allow, days, urlFilter, limitMode, urls string
	set := &cobra.Command{
		Use:   "set <mac>",
		Short: "Configure a device's parental control schedule and website filter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !changed(cmd, "allow", "days", "url-filter", "limit-mode", "urls") {
				return errNothingToChange
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			r, ok, err := c.ParentalRule(ctx, args[0])
			if err != nil {
				return err
			}
			if !ok {
				if err := tenda.ValidateNewParentalDeviceMAC(args[0]); err != nil {
					return err
				}
				r = parentalDefaultRule(args[0])
			}
			r.Enabled = true
			if changed(cmd, "allow") {
				if !parentalTimeRe.MatchString(allow) {
					return fmt.Errorf("invalid --allow %q: want HH:MM-HH:MM", allow)
				}
				r.AllowedWindow = allow
			}
			if changed(cmd, "days") {
				everyDay, dayList, err := parentalParseDays(days)
				if err != nil {
					return err
				}
				r.EveryDay, r.Days = everyDay, dayList
			}
			if changed(cmd, "url-filter") {
				on, err := parentalParseOnOff(urlFilter, "--url-filter")
				if err != nil {
					return err
				}
				r.URLFilterOn = on
			}
			if changed(cmd, "limit-mode") {
				if limitMode != "blacklist" && limitMode != "whitelist" {
					return fmt.Errorf("invalid --limit-mode %q: want blacklist or whitelist", limitMode)
				}
				r.LimitType = limitMode
			}
			if changed(cmd, "urls") {
				r.URLs = parentalParseURLs(urls)
			}
			if err := c.SetParentalRule(ctx, r); err != nil {
				return err
			}
			return a.done(cmd, "Parental control rule set for %s", args[0])
		},
	}
	set.Flags().StringVar(&allow, "allow", "", "window internet access is allowed, as HH:MM-HH:MM")
	set.Flags().StringVar(&days, "days", "", "comma-separated days (sun,mon,...) the window applies on, or \"all\" for every day")
	set.Flags().StringVar(&urlFilter, "url-filter", "", "website keyword filter: on or off")
	set.Flags().StringVar(&limitMode, "limit-mode", "", "website filter mode: blacklist or whitelist")
	set.Flags().StringVar(&urls, "urls", "", "comma-separated website keywords (max 10)")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <mac>",
		Short: "Delete a device's parental control rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveParentalRule(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Removed parental control rule for %s", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable <mac>",
		Short: "Block a device's internet access now (isControled)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return parentalSetBlocked(a, cmd, args[0], true, "Blocked %s")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable <mac>",
		Short: "Restore a device's internet access (isControled)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return parentalSetBlocked(a, cmd, args[0], false, "Unblocked %s")
		},
	})
	return cmd
}

func parentalSetBlocked(a *app, cmd *cobra.Command, mac string, blocked bool, done string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	if err := c.SetParentalBlocked(cmd.Context(), mac, blocked); err != nil {
		return err
	}
	return a.done(cmd, done, mac)
}

func parentalDefaultRule(mac string) tenda.ParentalRule {
	return tenda.ParentalRule{
		MAC: mac, Enabled: true, AllowedWindow: "19:00-21:00", EveryDay: true,
		URLFilterOn: true, LimitType: "blacklist",
	}
}

func parentalParseDays(s string) (everyDay bool, days []string, err error) {
	if strings.EqualFold(s, "all") {
		return true, nil, nil
	}
	names := map[string]bool{"sun": true, "mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true}
	for _, d := range strings.Split(s, ",") {
		d = strings.ToLower(strings.TrimSpace(d))
		if !names[d] {
			return false, nil, fmt.Errorf("invalid --days %q: want sun,mon,tue,wed,thu,fri,sat or all", s)
		}
		days = append(days, d)
	}
	if len(days) == 0 {
		return false, nil, fmt.Errorf("invalid --days %q: at least one day is required", s)
	}
	return false, days, nil
}

func parentalParseOnOff(s, flagName string) (bool, error) {
	switch strings.ToLower(s) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid %s %q: want on or off", flagName, s)
}

func parentalParseURLs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, u := range strings.Split(s, ",") {
		out = append(out, strings.ToLower(strings.TrimSpace(u)))
	}
	return out
}

func parentalListText(w io.Writer, devices []tenda.ParentalDevice) error {
	if len(devices) == 0 {
		_, err := fmt.Fprintln(w, "No known devices")
		return err
	}
	rows := make([][]string, len(devices))
	for i, d := range devices {
		status := "offline"
		if d.Online {
			status = "online"
		}
		rule := "none"
		if d.RuleSet {
			rule = "set"
		}
		blocked := "no"
		if d.Blocked {
			blocked = "yes"
		}
		rows[i] = []string{d.Name, d.MAC, d.IP, status, rule, blocked}
	}
	return table(w, []col{
		{title: "DEVICE NAME"}, {title: "MAC ADDRESS"}, {title: "IP ADDRESS"},
		{title: "STATUS"}, {title: "RULE"}, {title: "BLOCKED"},
	}, rows)
}

// parentalRuleJSON is the -o json shape for `show`: the rule, or a bare
// {"mac": ..., "configured": false} when none exists.
func parentalRuleJSON(r tenda.ParentalRule, ok bool) any {
	if !ok {
		return struct {
			MAC        string `json:"mac"`
			Configured bool   `json:"configured"`
		}{MAC: r.MAC, Configured: false}
	}
	return r
}

func parentalRuleText(w io.Writer, r tenda.ParentalRule, ok bool) error {
	if !ok {
		_, err := fmt.Fprintf(w, "No parental control rule configured for %s\n", r.MAC)
		return err
	}
	days := "every day"
	if !r.EveryDay {
		days = strings.Join(r.Days, ",")
		if days == "" {
			days = "(none selected)"
		}
	}
	urls := strings.Join(r.URLs, ",")
	if urls == "" {
		urls = "(none)"
	}
	return fields(w,
		"Enabled", onOff(r.Enabled),
		"Allowed window", r.AllowedWindow,
		"Days", days,
		"URL filter", onOff(r.URLFilterOn),
		"Filter mode", r.LimitType,
		"Keywords ("+strconv.Itoa(len(r.URLs))+")", urls,
	)
}
