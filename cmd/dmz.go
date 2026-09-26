package cmd

import (
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newDMZCmd) }

func newDMZCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		d, err := c.DMZ(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, d, func(w io.Writer) error { return dmzText(w, d) })
	}
	cmd := &cobra.Command{
		Use:   "dmz",
		Short: "Manage the DMZ host",
		Long: `Manage the DMZ host (Advanced Settings). A DMZ host receives every inbound
WAN port, bypassing NAT and the firewall.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the DMZ host",
		Args:  cobra.NoArgs,
		RunE:  show,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set <ip>",
		Short: "Set and enable the DMZ host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.DMZ, c.SetDMZ, func(d *tenda.DMZ) error {
				d.Enabled = true
				d.HostIP = args[0]
				return nil
			}, "DMZ host set to "+args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Enable the DMZ",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.DMZ, c.SetDMZ, func(d *tenda.DMZ) error { d.Enabled = true; return nil }, "DMZ enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Disable the DMZ",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.DMZ, c.SetDMZ, func(d *tenda.DMZ) error { d.Enabled = false; return nil }, "DMZ disabled")
		},
	})
	return cmd
}

func dmzText(w io.Writer, d tenda.DMZ) error {
	return fields(w, "Enabled", onOff(d.Enabled), "Host IP", d.HostIP, "LAN IP", d.LANIP, "LAN mask", d.LANMask)
}
