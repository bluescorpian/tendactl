package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("vpn", newVPNClientCmd) }

// newVPNClientCmd is "vpn client": dialing out to an upstream PPTP/L2TP VPN
// server. This is unrelated to "vpn server", this router's own PPTP server.
func newVPNClientCmd(a *app) *cobra.Command {
	var showPassword bool
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		v, err := c.VPNClient(cmd.Context())
		if err != nil {
			return err
		}
		v.Password = maskSecret(v.Password, showPassword)
		return a.render(cmd, v, func(w io.Writer) error { return vpnClientText(w, v) })
	}
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Manage the PPTP/L2TP VPN client",
		Long: `Manage "PPTP/L2TP Client" (VPN tab): dial out to an upstream PPTP or L2TP VPN
server.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.Flags().BoolVar(&showPassword, "show-password", false, "show the client password instead of masking it")
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the VPN client configuration",
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	showCmd.Flags().BoolVar(&showPassword, "show-password", false, "show the client password instead of masking it")
	cmd.AddCommand(showCmd)

	var vpnType, domain, user, password, mppe string
	var mppeBits int
	set := &cobra.Command{
		Use:   "set",
		Short: "Configure the VPN client's server, type or credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "type", "domain", "user", "password", "mppe", "mppe-bits") {
				return errNothingToChange
			}
			if changed(cmd, "type") && vpnType != "pptp" && vpnType != "l2tp" {
				return fmt.Errorf("invalid --type %q: want pptp or l2tp", vpnType)
			}
			var mppeOn bool
			if changed(cmd, "mppe") {
				var err error
				if mppeOn, err = vpnClientParseOnOff(mppe); err != nil {
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
			return update(a, cmd, c.VPNClient, c.SetVPNClient, func(v *tenda.VPNClient) error {
				if changed(cmd, "type") {
					v.Type = vpnType
				}
				if changed(cmd, "domain") {
					v.Domain = domain
				}
				if changed(cmd, "user") {
					v.User = user
				}
				if changed(cmd, "password") {
					v.Password = password
				}
				if changed(cmd, "mppe") {
					v.MPPE = mppeOn
				}
				if changed(cmd, "mppe-bits") {
					v.MPPEBits = mppeBits
				}
				return nil
			}, "VPN client settings updated")
		},
	}
	set.Flags().StringVar(&vpnType, "type", "", "VPN protocol: pptp or l2tp")
	set.Flags().StringVar(&domain, "domain", "", "VPN server IP address or domain name")
	set.Flags().StringVar(&user, "user", "", "VPN account user name")
	set.Flags().StringVar(&password, "password", "", "VPN account password")
	set.Flags().StringVar(&mppe, "mppe", "", "MPPE encryption (PPTP only): on or off")
	set.Flags().IntVar(&mppeBits, "mppe-bits", 0, "MPPE encryption bits (PPTP only): 40 or 128")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn the VPN client on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.VPNClient, c.SetVPNClient, func(v *tenda.VPNClient) error { v.Enabled = true; return nil }, "VPN client enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn the VPN client off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.VPNClient, c.SetVPNClient, func(v *tenda.VPNClient) error { v.Enabled = false; return nil }, "VPN client disabled")
		},
	})
	return cmd
}

func vpnClientParseOnOff(s string) (bool, error) {
	switch s {
	case "on":
		return true, nil
	case "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid --mppe %q: want on or off", s)
}

func vpnClientText(w io.Writer, v tenda.VPNClient) error {
	return fields(w,
		"Enabled", onOff(v.Enabled),
		"Type", v.Type,
		"Domain", v.Domain,
		"User", v.User,
		"Password", v.Password,
		"MPPE", onOff(v.MPPE),
		"MPPE bits", strconv.Itoa(v.MPPEBits),
		"PPTP status", v.PPTPStatus,
		"PPTP IP", v.PPTPIP,
		"L2TP status", v.L2TPStatus,
		"L2TP IP", v.L2TPIP,
	)
}
