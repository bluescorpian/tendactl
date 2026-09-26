package cmd

import (
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newFirewallCmd) }

func newFirewallCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		f, err := c.Firewall(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, f, func(w io.Writer) error { return firewallText(w, f) })
	}
	cmd := &cobra.Command{
		Use:   "firewall",
		Short: "Manage flood defense and WAN ping",
		Long:  `Manage the firewall's flood-defense and WAN-ping toggles (Advanced Settings).`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show flood defense and WAN ping settings",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var icmp, tcp, udp, ignorePing bool
	set := &cobra.Command{
		Use:   "set",
		Short: "Set flood defense and WAN ping settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "icmp-flood", "tcp-flood", "udp-flood", "ignore-wan-ping") {
				return errNothingToChange
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Firewall, c.SetFirewall, func(f *tenda.Firewall) error {
				if changed(cmd, "icmp-flood") {
					f.ICMPFloodDefense = icmp
				}
				if changed(cmd, "tcp-flood") {
					f.TCPFloodDefense = tcp
				}
				if changed(cmd, "udp-flood") {
					f.UDPFloodDefense = udp
				}
				if changed(cmd, "ignore-wan-ping") {
					f.IgnoreWANPing = ignorePing
				}
				return nil
			}, "Firewall settings updated")
		},
	}
	set.Flags().BoolVar(&icmp, "icmp-flood", false, "ICMP flood attack defense")
	set.Flags().BoolVar(&tcp, "tcp-flood", false, "TCP flood attack defense")
	set.Flags().BoolVar(&udp, "udp-flood", false, "UDP flood attack defense")
	set.Flags().BoolVar(&ignorePing, "ignore-wan-ping", false, "ignore ping packets from the WAN port")
	cmd.AddCommand(set)
	return cmd
}

func firewallText(w io.Writer, f tenda.Firewall) error {
	return fields(w,
		"ICMP flood defense", onOff(f.ICMPFloodDefense),
		"TCP flood defense", onOff(f.TCPFloodDefense),
		"UDP flood defense", onOff(f.UDPFloodDefense),
		"Ignore WAN ping", onOff(f.IgnoreWANPing),
	)
}
