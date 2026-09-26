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

// clientsTruncate shortens s to n runes, ending in "...".
func clientsTruncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-3]) + "..."
}
