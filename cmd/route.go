package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newRouteCmd) }

func newRouteCmd(a *app) *cobra.Command {
	list := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		cfg, err := c.Routes(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, cfg, func(w io.Writer) error { return routeText(w, cfg) })
	}
	cmd := &cobra.Command{
		Use:   "route",
		Short: "Manage static routes (Static Route)",
		Long: `Manage "Static Route" (Advanced Settings). System routes are added by the
router itself and are read-only; only user-added routes can be added or
removed.`,
		Args: cobra.NoArgs,
		RunE: list,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List static routes",
		Args:  cobra.NoArgs,
		RunE:  list,
	})

	var ifname string
	add := &cobra.Command{
		Use:     "add <network> <mask> <gateway>",
		Short:   "Add a static route",
		Long:    `Add a user-defined static route. gateway may be 0.0.0.0 for none.`,
		Example: `  tendactl route add 10.0.0.0 255.255.255.0 192.168.0.254`,
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			rt := tenda.Route{Network: args[0], Mask: args[1], Gateway: args[2], Interface: ifname}
			if err := c.AddRoute(cmd.Context(), rt); err != nil {
				return err
			}
			return a.done(cmd, "Added route to %s/%s via %s", args[0], args[1], args[2])
		},
	}
	add.Flags().StringVar(&ifname, "if", "WAN1", "WAN interface the route uses")
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use:   "rm <network> <mask>",
		Short: "Remove a user-added static route",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.RemoveRoute(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			return a.done(cmd, "Removed route to %s/%s", args[0], args[1])
		},
	})
	return cmd
}

var routeCols = []col{{title: "NETWORK"}, {title: "MASK"}, {title: "GATEWAY"}, {title: "INTERFACE"}, {title: "ACTIVE"}}

func routeText(w io.Writer, cfg tenda.RouteConfig) error {
	if err := fields(w, "LAN IP", cfg.LANIP, "LAN mask", cfg.LANMask, "WAN mask", cfg.WANMask, "WAN gateway", cfg.WANGateway); err != nil {
		return err
	}
	var user, system []tenda.Route
	for _, rt := range cfg.Routes {
		if rt.System {
			system = append(system, rt)
		} else {
			user = append(user, rt)
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if len(user) == 0 {
		if _, err := fmt.Fprintln(w, "No user-added static routes"); err != nil {
			return err
		}
	} else if err := table(w, routeCols, routeRows(user)); err != nil {
		return err
	}
	if len(system) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "System routes (read-only):"); err != nil {
		return err
	}
	return table(w, routeCols, routeRows(system))
}

func routeRows(routes []tenda.Route) [][]string {
	rows := make([][]string, len(routes))
	for i, rt := range routes {
		active := "no"
		if rt.Active {
			active = "yes"
		}
		rows[i] = []string{rt.Network, rt.Mask, rt.Gateway, rt.Interface, active}
	}
	return rows
}
