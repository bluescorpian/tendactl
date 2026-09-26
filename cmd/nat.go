package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newNATCmd) }

func newNATCmd(a *app) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		cfg, err := c.NAT(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, cfg.Rules, func(w io.Writer) error { return natText(w, cfg.Rules) })
	}
	cmd := &cobra.Command{
		Use:   "nat",
		Short: "Manage port forwarding rules (Virtual Server)",
		Long: `Manage port forwarding (the UI's "Virtual Server"). Each rule forwards a WAN
port to a port on a LAN device; the WAN port identifies the rule.`,
		Args: cobra.NoArgs,
		RunE: list,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List port forwarding rules",
		Args:  cobra.NoArgs,
		RunE:  list,
	})

	var proto tenda.Protocol
	add := &cobra.Command{
		Use:   "add <ip> <inPort> [outPort]",
		Short: "Forward a WAN port to a LAN device",
		Long: `Forward WAN port outPort to inPort on the LAN device at ip. outPort defaults
to inPort.`,
		Example: `  tendactl nat add 192.168.0.100 80 8080 --proto tcp`,
		Args:    cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			r := tenda.PortForward{IP: args[0], Protocol: proto}
			var err error
			if r.InPort, err = natPort(args[1]); err != nil {
				return err
			}
			r.OutPort = r.InPort
			if len(args) == 3 {
				if r.OutPort, err = natPort(args[2]); err != nil {
					return err
				}
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.AddNATRule(cmd.Context(), r); err != nil {
				return err
			}
			return a.done(cmd, "Forwarding WAN port %d to %s:%d (%s)", r.OutPort, r.IP, r.InPort, r.Protocol)
		},
	}
	add.Flags().Var(&proto, "proto", "protocol: both, tcp or udp")
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <outPort>",
		Short: "Remove the rule for a WAN port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := natPort(args[0])
			if err != nil {
				return err
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveNATRule(cmd.Context(), port); err != nil {
				return err
			}
			return a.done(cmd, "Removed port forwarding rule for WAN port %d", port)
		},
	})
	return cmd
}

func natPort(s string) (int, error) {
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("invalid port %q: want 1-65535", s)
	}
	return p, nil
}

func natText(w io.Writer, rules []tenda.PortForward) error {
	if len(rules) == 0 {
		_, err := fmt.Fprintln(w, "No port forwarding rules configured")
		return err
	}
	rows := make([][]string, len(rules))
	for i, r := range rules {
		rows[i] = []string{r.IP, strconv.Itoa(r.InPort), strconv.Itoa(r.OutPort), r.Protocol.String()}
	}
	return table(w, []col{{title: "IP"}, {title: "INTERNAL PORT"}, {title: "EXTERNAL PORT"}, {title: "PROTOCOL"}}, rows)
}
