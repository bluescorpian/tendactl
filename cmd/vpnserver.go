package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() {
	register("vpn", newVPNServerCmd)
	register("vpn", newVPNServerUsersCmd)
	register("vpn", newVPNServerOnlineCmd)
}

// newVPNServerCmd is "vpn server": the PPTP server's own settings (IP pool,
// MPPE). The user account table is "vpn users", and connected clients are
// "vpn online".
func newVPNServerCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		s, err := c.VPNServer(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, s, func(w io.Writer) error { return vpnServerText(w, s) })
	}
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the PPTP server (IP pool, MPPE)",
		Long: `Manage "PPTP Server" (VPN tab): on/off, the client IP address pool, and MPPE
encryption. User accounts are managed with "vpn users".`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the PPTP server configuration",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var startIP, endIP, mppe string
	var mppeBits int
	set := &cobra.Command{
		Use:   "set",
		Short: "Configure the PPTP server's IP pool or MPPE encryption",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "pool-start", "pool-end", "mppe", "mppe-bits") {
				return errNothingToChange
			}
			var mppeOn bool
			if changed(cmd, "mppe") {
				var err error
				if mppeOn, err = vpnServerParseOnOff(mppe, "--mppe"); err != nil {
					return err
				}
			}
			if changed(cmd, "mppe-bits") && mppeBits != 40 && mppeBits != 128 {
				return fmt.Errorf("invalid --mppe-bits %d: want 40 or 128", mppeBits)
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.VPNServer, c.SetVPNServer, func(s *tenda.VPNServer) error {
				if changed(cmd, "pool-start") {
					s.StartIP = startIP
				}
				if changed(cmd, "pool-end") {
					s.EndIP = endIP
				}
				if changed(cmd, "mppe") {
					s.MPPE = mppeOn
				}
				if changed(cmd, "mppe-bits") {
					s.MPPEBits = mppeBits
				}
				return nil
			}, "PPTP server settings updated")
		},
	}
	set.Flags().StringVar(&startIP, "pool-start", "", "first address of the client IP pool")
	set.Flags().StringVar(&endIP, "pool-end", "", "last address of the client IP pool")
	set.Flags().StringVar(&mppe, "mppe", "", "MPPE encryption: on or off")
	set.Flags().IntVar(&mppeBits, "mppe-bits", 0, "MPPE encryption bits: 40 or 128")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn the PPTP server on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.VPNServer, c.SetVPNServer, func(s *tenda.VPNServer) error { s.Enabled = true; return nil }, "PPTP server enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn the PPTP server off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.VPNServer, c.SetVPNServer, func(s *tenda.VPNServer) error { s.Enabled = false; return nil }, "PPTP server disabled")
		},
	})
	return cmd
}

func vpnServerParseOnOff(s, flagName string) (bool, error) {
	switch s {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid %s %q: want on or off", flagName, s)
}

func vpnServerText(w io.Writer, s tenda.VPNServer) error {
	return fields(w,
		"Enabled", onOff(s.Enabled),
		"Pool start", s.StartIP,
		"Pool end", s.EndIP,
		"MPPE", onOff(s.MPPE),
		"MPPE bits", strconv.Itoa(s.MPPEBits),
	)
}

// newVPNServerUsersCmd is "vpn users": the PPTP server's account table.
func newVPNServerUsersCmd(a *app) *cobra.Command {
	var showPassword bool
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		users, err := c.VPNServerUsers(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, vpnServerUsersJSON(users, showPassword), func(w io.Writer) error { return vpnServerUsersText(w, users, showPassword) })
	}
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage PPTP server user accounts",
		Long:  `Manage the PPTP server's user accounts (VPN tab, "PPTP Server").`,
		Args:  cobra.NoArgs,
		RunE:  list,
	}
	cmd.PersistentFlags().BoolVar(&showPassword, "show-password", false, "show account passwords instead of masking them")
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List PPTP server user accounts",
		Args:  cobra.NoArgs,
		RunE:  list,
	})

	var password string
	add := &cobra.Command{
		Use:   "add <user>",
		Short: "Add a PPTP server user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.AddVPNServerUser(cmd.Context(), args[0], password); err != nil {
				return err
			}
			return a.done(cmd, "Added PPTP server user %s", args[0])
		},
	}
	add.Flags().StringVar(&password, "password", "", "the account's password")
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <user>",
		Short: "Remove a PPTP server user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveVPNServerUser(cmd.Context(), args[0]); err != nil {
				return err
			}
			return a.done(cmd, "Removed PPTP server user %s", args[0])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable <user>",
		Short: "Turn on a PPTP server user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vpnServerSetUserEnabled(a, cmd, args[0], true, "Enabled PPTP server user %s")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable <user>",
		Short: "Turn off a PPTP server user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vpnServerSetUserEnabled(a, cmd, args[0], false, "Disabled PPTP server user %s")
		},
	})
	return cmd
}

func vpnServerSetUserEnabled(a *app, cmd *cobra.Command, user string, enabled bool, done string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	if err := c.SetVPNServerUserEnabled(cmd.Context(), user, enabled); err != nil {
		return err
	}
	return a.done(cmd, done, user)
}

// vpnServerUsersJSON masks passwords the same way the text output does,
// since the -o json contract carries no separate secret channel.
func vpnServerUsersJSON(users []tenda.VPNServerUser, showPassword bool) []tenda.VPNServerUser {
	out := make([]tenda.VPNServerUser, len(users))
	for i, u := range users {
		u.Password = maskSecret(u.Password, showPassword)
		out[i] = u
	}
	return out
}

func vpnServerUsersText(w io.Writer, users []tenda.VPNServerUser, showPassword bool) error {
	if len(users) == 0 {
		_, err := fmt.Fprintln(w, "No PPTP server user accounts configured")
		return err
	}
	rows := make([][]string, len(users))
	for i, u := range users {
		rows[i] = []string{u.Name, maskSecret(u.Password, showPassword), onOff(u.Enabled), onOff(u.Connected)}
	}
	return table(w, []col{{title: "USER NAME"}, {title: "PASSWORD"}, {title: "ENABLED"}, {title: "CONNECTED"}}, rows)
}

// newVPNServerOnlineCmd is "vpn online": a display-only list of clients
// currently connected to this router's PPTP server.
func newVPNServerOnlineCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "online",
		Short: "List clients connected to the PPTP server",
		Long:  `List "Online PPTP Users" (VPN tab): clients currently connected to this router's PPTP server.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			users, err := c.VPNServerOnlineUsers(cmd.Context())
			if err != nil {
				return err
			}
			return a.render(cmd, users, func(w io.Writer) error { return vpnServerOnlineText(w, users) })
		},
	}
}

func vpnServerOnlineText(w io.Writer, users []tenda.VPNServerOnlineUser) error {
	if len(users) == 0 {
		_, err := fmt.Fprintln(w, "No PPTP clients connected")
		return err
	}
	rows := make([][]string, len(users))
	for i, u := range users {
		rows[i] = []string{u.Name, u.DialIP, u.ClientIP, strconv.Itoa(u.OnlineMinutes)}
	}
	return table(w, []col{{title: "USER NAME"}, {title: "DIAL-IN IP"}, {title: "ASSIGNED IP"}, {title: "ONLINE (MIN)", right: true}}, rows)
}
