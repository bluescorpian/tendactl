package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newDHCPCmd) }

func newDHCPCmd(a *app) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		d, err := c.DHCP(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, d, func(w io.Writer) error { return dhcpText(w, d) })
	}
	cmd := &cobra.Command{
		Use:   "dhcp",
		Short: "Manage DHCP reservations (DHCP Reservation)",
		Long: `Manage "DHCP Reservation" (System Settings). A reservation fixes the IP
address the router hands a device on its next DHCP lease.`,
		Args: cobra.NoArgs,
		RunE: list,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List DHCP reservations",
		Args:  cobra.NoArgs,
		RunE:  list,
	})

	var name string
	add := &cobra.Command{
		Use:   "add <mac> <ip>",
		Short: "Reserve an IP address for a device",
		Long: `Reserve ip for mac. Re-adding an existing MAC updates its reservation. The
binding takes effect the next time that device requests a DHCP lease.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			b := tenda.DHCPBinding{MAC: args[0], IP: args[1], Name: name}
			if err := c.AddDHCPBinding(cmd.Context(), b); err != nil {
				return err
			}
			return a.done(cmd, "Reserved %s for %s", args[1], args[0])
		},
	}
	add.Flags().StringVar(&name, "name", "", "device name")
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <mac>",
		Short: "Remove a DHCP reservation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveDHCPBinding(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Removed reservation for %s", args[0])
		},
	})
	return cmd
}

func dhcpText(w io.Writer, d tenda.DHCP) error {
	if err := fields(w, "LAN IP", d.LANIP, "LAN mask", d.LANMask); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if len(d.Bindings) == 0 {
		if _, err := fmt.Fprintln(w, "No DHCP reservations configured"); err != nil {
			return err
		}
	} else {
		rows := make([][]string, len(d.Bindings))
		for i, b := range d.Bindings {
			rows[i] = []string{b.Name, b.MAC, b.IP, dhcpStatus(b.Online)}
		}
		if err := table(w, dhcpCols, rows); err != nil {
			return err
		}
	}
	if len(d.Clients) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Other DHCP clients (not reserved):"); err != nil {
		return err
	}
	rows := make([][]string, len(d.Clients))
	for i, cl := range d.Clients {
		rows[i] = []string{cl.Name, cl.MAC, cl.IP, dhcpStatus(cl.Online)}
	}
	return table(w, dhcpCols, rows)
}

var dhcpCols = []col{{title: "DEVICE NAME"}, {title: "MAC ADDRESS"}, {title: "IP ADDRESS"}, {title: "STATUS"}}

func dhcpStatus(online bool) string {
	if online {
		return "online"
	}
	return "offline"
}
