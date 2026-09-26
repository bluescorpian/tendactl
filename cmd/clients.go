package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newClientsCmd) }

func newClientsCmd(a *app) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		l, err := c.OnlineList(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, l, func(w io.Writer) error { return clientsText(w, l) })
	}
	cmd := &cobra.Command{
		Use:     "clients",
		Aliases: []string{"online"},
		Short:   "List connected devices (Manage Device)",
		Long: `List the devices the router has seen, with their current upload and
download speed in KB/s. Guest network clients are marked [Guest].`,
		Args: cobra.NoArgs,
		RunE: list,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List connected devices",
		Args:  cobra.NoArgs,
		RunE:  list,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "rename <mac> <name>",
		Short: "Rename a device (SetOnlineDevName)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RenameClient(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return a.done(cmd, "Renamed %s to %q", args[0], args[1])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "block <mac>",
		Short: "Add a device to the quick blacklist (setBlackRule)",
		Long: `Add a device to the "Manage Device" quick blacklist, cutting off its network
access immediately. This is a different list from macfilter. Blocking the
local host device is refused; use the api command to override.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.BlockClient(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Blocked %s", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "unblock <mac>",
		Short: "Remove a device from the quick blacklist (delBlackRule)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.UnblockClient(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Unblocked %s", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "blocked",
		Short: "List devices on the quick blacklist (getBlackRuleList)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			bl, err := c.BlockedClients(cmd.Context())
			if err != nil {
				return err
			}
			return a.render(cmd, bl, func(w io.Writer) error { return clientsBlockedText(w, bl) })
		},
	})
	return cmd
}

func clientsText(w io.Writer, l tenda.OnlineList) error {
	if _, err := fmt.Fprintf(w, "%s @ %s (%s)\n\n", l.Host.Name, l.Host.IP, l.Host.MAC); err != nil {
		return err
	}
	if len(l.Clients) == 0 {
		_, err := fmt.Fprintln(w, "No connected devices")
		return err
	}
	rows := make([][]string, len(l.Clients))
	for i, c := range l.Clients {
		typ := ""
		if c.Guest {
			typ = "[Guest]"
		}
		rows[i] = []string{clientsTruncate(c.Name, 18), c.IP, c.UpKBps, c.DownKBps, typ}
	}
	return table(w, []col{
		{title: "DEVICE NAME"}, {title: "IP ADDRESS"},
		{title: "↑KB/s", right: true}, {title: "↓KB/s", right: true},
		{title: "TYPE"},
	}, rows)
}

func clientsBlockedText(w io.Writer, bl []tenda.BlockedClient) error {
	if len(bl) == 0 {
		_, err := fmt.Fprintln(w, "No blocked devices")
		return err
	}
	rows := make([][]string, len(bl))
	for i, b := range bl {
		rows[i] = []string{b.MAC, b.Name}
	}
	return table(w, []col{{title: "MAC ADDRESS"}, {title: "DEVICE NAME"}}, rows)
}

// clientsTruncate shortens s to n runes, ending in "...".
func clientsTruncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-3]) + "..."
}
