package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newMACFilterCmd) }

func newMACFilterCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		f, err := c.MACFilter(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, f, func(w io.Writer) error { return macFilterText(w, f) })
	}
	cmd := &cobra.Command{
		Use:   "macfilter",
		Short: "Manage the MAC address allow/deny list (Filter MAC Address)",
		Long: `Manage "Filter MAC Address" (Advanced Settings). In blacklist mode, listed
devices are denied; in whitelist mode, only listed devices are allowed and
every other device loses access.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the active list and its mode",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var name string
	add := &cobra.Command{
		Use:   "add <mac>",
		Short: "Add a device to the active list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.AddMACFilterEntry(cmd.Context(), args[0], name); err != nil {
				return err
			}
			return a.done(cmd, "Added %s", args[0])
		},
	}
	add.Flags().StringVar(&name, "name", "", "device name")
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <mac>",
		Short: "Remove a device from the active list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveMACFilterEntry(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Removed %s", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "mode <black|white>",
		Short: "Switch between blacklist and whitelist mode",
		Long: `Switch mode. Whitelist mode needs --yes: only listed devices keep access,
and every other device is cut off immediately.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.SetMACFilterMode(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "MAC filter mode set to %s", args[0])
		},
	})
	return cmd
}

func macFilterText(w io.Writer, f tenda.MACFilter) error {
	if err := fields(w, "Mode", f.Mode); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if len(f.Devices) == 0 {
		_, err := fmt.Fprintf(w, "No entries in the %s list\n", f.Mode)
		return err
	}
	rows := make([][]string, len(f.Devices))
	for i, d := range f.Devices {
		rows[i] = []string{d.MAC, d.Name}
	}
	return table(w, []col{{title: "MAC ADDRESS"}, {title: "DEVICE NAME"}}, rows)
}
