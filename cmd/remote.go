package cmd

import (
	"fmt"
	"io"
	"net"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("system", newRemoteCmd) }

func newRemoteCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		r, err := c.Remote(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, r, func(w io.Writer) error { return remoteText(w, r) })
	}
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage remote (WAN-side) web admin access",
		Long: `Manage "Remote Management" (System Settings). Enabling this exposes the
router's admin UI to the WAN, optionally restricted to one source IP.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show remote management settings",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var port int
	var from string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the allowed source IP or port",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "port", "from") {
				return errNothingToChange
			}
			if changed(cmd, "from") && net.ParseIP(from) == nil {
				return fmt.Errorf("invalid --from %q: want an IPv4 address, or 0.0.0.0 for any", from)
			}
			if changed(cmd, "port") && (port < 1 || port > 65535) {
				return fmt.Errorf("invalid --port %d: want 1-65535", port)
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Remote, c.SetRemote, func(r *tenda.Remote) error {
				if changed(cmd, "port") {
					r.Port = port
				}
				if changed(cmd, "from") {
					r.FromIP = from
				}
				return nil
			}, "Remote management settings updated")
		},
	}
	set.Flags().IntVar(&port, "port", 0, "WAN port for the admin UI")
	set.Flags().StringVar(&from, "from", "", "allowed WAN source IP, or 0.0.0.0 for any")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn on remote management",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Remote, c.SetRemote, func(r *tenda.Remote) error { r.Enabled = true; return nil }, "Remote management enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn off remote management",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Remote, c.SetRemote, func(r *tenda.Remote) error { r.Enabled = false; return nil }, "Remote management disabled")
		},
	})
	return cmd
}

func remoteText(w io.Writer, r tenda.Remote) error {
	return fields(w,
		"Enabled", onOff(r.Enabled),
		"Allowed source", r.FromIP,
		"Port", fmt.Sprint(r.Port),
		"Admin password set", onOff(r.PasswordSet),
	)
}
