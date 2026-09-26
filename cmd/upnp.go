package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newUPnPCmd) }

func newUPnPCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		u, err := c.UPnP(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, u, func(w io.Writer) error { return upnpText(w, u) })
	}
	cmd := &cobra.Command{
		Use:   "upnp",
		Short: "Manage UPnP",
		Long: `Manage UPnP (Advanced Settings). Enabling it lets LAN devices open inbound
WAN ports on their own, without further admin action.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show UPnP status and active mappings",
		Args:  cobra.NoArgs,
		RunE:  show,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Enable UPnP",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.UPnP, c.SetUPnP, func(u *tenda.UPnP) error { u.Enabled = true; return nil }, "UPnP enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Disable UPnP",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.UPnP, c.SetUPnP, func(u *tenda.UPnP) error { u.Enabled = false; return nil }, "UPnP disabled")
		},
	})
	return cmd
}

func upnpText(w io.Writer, u tenda.UPnP) error {
	if err := fields(w, "Enabled", onOff(u.Enabled)); err != nil {
		return err
	}
	if len(u.Mappings) == 0 {
		_, err := fmt.Fprintln(w, "No active UPnP mappings")
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	rows := make([][]string, len(u.Mappings))
	for i, m := range u.Mappings {
		rows[i] = []string{m.Host, m.InPort, m.OutPort, m.Protocol, m.RemoteHost}
	}
	return table(w, []col{{title: "HOST"}, {title: "IN PORT"}, {title: "OUT PORT"}, {title: "PROTOCOL"}, {title: "REMOTE HOST"}}, rows)
}
