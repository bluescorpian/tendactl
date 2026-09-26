package cmd

import (
	"fmt"
	"io"
	"net"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newLANCmd) }

func newLANCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		l, err := c.LAN(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, l, func(w io.Writer) error { return lanText(w, l) })
	}
	cmd := &cobra.Command{
		Use:   "lan",
		Short: "Manage the router's LAN address and DHCP server (LAN Settings)",
		Long: `Manage "LAN Settings" (System Settings). Changing the LAN address moves the
router's own management address; clients (and this tool) lose it until they
pick up the change. Requires --yes.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show LAN and DHCP settings",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var ip, mask, dhcp, dhcpStart, dhcpEnd, leaseTime, dnsAuto, dns1, dns2 string
	set := &cobra.Command{
		Use:   "set",
		Short: "Change the LAN address or DHCP server settings",
		Long:  `Change LAN Settings. Requires --yes.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "ip", "mask", "dhcp", "dhcp-start", "dhcp-end", "lease-time", "dns-auto", "dns1", "dns2") {
				return errNothingToChange
			}
			for _, f := range []struct{ name, val string }{
				{"ip", ip}, {"mask", mask}, {"dhcp-start", dhcpStart}, {"dhcp-end", dhcpEnd}, {"dns1", dns1}, {"dns2", dns2},
			} {
				if changed(cmd, f.name) && net.ParseIP(f.val).To4() == nil {
					return fmt.Errorf("invalid --%s %q: want an IPv4 address", f.name, f.val)
				}
			}
			var dhcpOn, dnsOn bool
			var err error
			if changed(cmd, "dhcp") {
				if dhcpOn, err = lanParseOnOff(dhcp); err != nil {
					return err
				}
			}
			if changed(cmd, "dns-auto") {
				if dnsOn, err = lanParseOnOff(dnsAuto); err != nil {
					return err
				}
			}
			if changed(cmd, "lease-time") {
				switch leaseTime {
				case "604800", "172800", "86400", "21600", "3600":
				default:
					return fmt.Errorf("invalid --lease-time %q: want 604800, 172800, 86400, 21600 or 3600 (seconds)", leaseTime)
				}
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.LAN, c.SetLAN, func(l *tenda.LAN) error {
				if changed(cmd, "ip") {
					l.LANIP = ip
				}
				if changed(cmd, "mask") {
					l.LANMask = mask
				}
				if changed(cmd, "dhcp") {
					l.DHCPEnabled = dhcpOn
				}
				if changed(cmd, "dhcp-start") {
					l.DHCPStart = dhcpStart
				}
				if changed(cmd, "dhcp-end") {
					l.DHCPEnd = dhcpEnd
				}
				if changed(cmd, "lease-time") {
					l.LeaseTime = leaseTime
				}
				if changed(cmd, "dns-auto") {
					l.DNSAuto = dnsOn
				}
				if changed(cmd, "dns1") {
					l.DNS1 = dns1
				}
				if changed(cmd, "dns2") {
					l.DNS2 = dns2
				}
				return nil
			}, "LAN settings updated")
		},
	}
	set.Flags().StringVar(&ip, "ip", "", "router's LAN IP address")
	set.Flags().StringVar(&mask, "mask", "", "LAN subnet mask")
	set.Flags().StringVar(&dhcp, "dhcp", "", "DHCP server: on or off")
	set.Flags().StringVar(&dhcpStart, "dhcp-start", "", "DHCP range start address")
	set.Flags().StringVar(&dhcpEnd, "dhcp-end", "", "DHCP range end address")
	set.Flags().StringVar(&leaseTime, "lease-time", "", "DHCP lease time in seconds: 604800, 172800, 86400, 21600 or 3600")
	set.Flags().StringVar(&dnsAuto, "dns-auto", "", "LAN-side DNS: auto or manual")
	set.Flags().StringVar(&dns1, "dns1", "", "primary LAN DNS server")
	set.Flags().StringVar(&dns2, "dns2", "", "secondary LAN DNS server")
	cmd.AddCommand(set)
	return cmd
}

func lanParseOnOff(s string) (bool, error) {
	switch s {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q: want on or off", s)
	}
}

func lanText(w io.Writer, l tenda.LAN) error {
	dns := l.DNS1 + ", " + l.DNS2
	if l.DNSAuto {
		dns = "auto"
	}
	return fields(w,
		"LAN IP", l.LANIP,
		"LAN mask", l.LANMask,
		"DHCP server", onOff(l.DHCPEnabled),
		"DHCP range", l.DHCPStart+" - "+l.DHCPEnd,
		"Lease time", l.LeaseTime+"s",
		"DNS", dns,
	)
}
