package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newBandwidthCmd) }

func newBandwidthCmd(a *app) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		b, err := c.Bandwidth(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, b, func(w io.Writer) error { return bandwidthText(w, b) })
	}
	cmd := &cobra.Command{
		Use:   "bandwidth",
		Short: "Manage per-device bandwidth caps (Bandwidth Control)",
		Args:  cobra.NoArgs,
		RunE:  list,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List devices and their bandwidth caps",
		Args:  cobra.NoArgs,
		RunE:  list,
	})

	var up, down float64
	set := &cobra.Command{
		Use:   "set <mac>",
		Short: "Cap a device's upload and/or download speed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !changed(cmd, "up", "down") {
				return errNothingToChange
			}
			var upPtr, downPtr *float64
			if changed(cmd, "up") {
				upPtr = &up
			}
			if changed(cmd, "down") {
				downPtr = &down
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.SetBandwidthLimit(cmd.Context(), args[0], upPtr, downPtr); err != nil {
				return err
			}
			return a.done(cmd, "Bandwidth cap set for %s", args[0])
		},
	}
	set.Flags().Float64Var(&up, "up", 0, "upload cap in Mbps (0 = unlimited)")
	set.Flags().Float64Var(&down, "down", 0, "download cap in Mbps (0 = unlimited)")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <mac>",
		Short: "Remove a device's bandwidth cap (reset to unlimited)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveBandwidthLimit(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Bandwidth cap removed for %s", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn Bandwidth Control on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.SetBandwidthEnabled(cmd.Context(), true); err != nil {
				return err
			}
			return a.done(cmd, "Bandwidth Control enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn Bandwidth Control off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.SetBandwidthEnabled(cmd.Context(), false); err != nil {
				return err
			}
			return a.done(cmd, "Bandwidth Control disabled")
		},
	})
	return cmd
}

func bandwidthLimitText(mbps float64) string {
	if mbps == 0 {
		return "unlimited"
	}
	return strconv.FormatFloat(mbps, 'g', -1, 64) + " Mbps"
}

func bandwidthText(w io.Writer, b tenda.Bandwidth) error {
	if err := fields(w, "Bandwidth Control", onOff(b.Enabled)); err != nil {
		return err
	}
	if len(b.Devices) == 0 {
		_, err := fmt.Fprintln(w, "No known devices")
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	rows := make([][]string, len(b.Devices))
	for i, d := range b.Devices {
		status := "online"
		if d.Offline {
			status = "offline"
		}
		rows[i] = []string{d.Name, d.MAC, d.UpKBps, d.DownKBps, bandwidthLimitText(d.LimitUpMbps), bandwidthLimitText(d.LimitDownMbps), status}
	}
	return table(w, []col{
		{title: "DEVICE NAME"}, {title: "MAC ADDRESS"},
		{title: "↑KB/s", right: true}, {title: "↓KB/s", right: true},
		{title: "UP LIMIT"}, {title: "DOWN LIMIT"}, {title: "STATUS"},
	}, rows)
}
