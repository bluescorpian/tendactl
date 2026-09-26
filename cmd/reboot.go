package cmd

import "github.com/spf13/cobra"

func init() { register("system", newRebootCmd) }

func newRebootCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "reboot",
		Short: "Reboot the router",
		Long: `Reboot the router. WiFi, LAN and WAN drop for about 45 seconds. Requires
--yes.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.Reboot(cmd.Context()); err != nil {
				return err
			}
			return a.done(cmd, "Router rebooting")
		},
	}
}
