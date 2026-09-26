package cmd

import (
	"fmt"
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("wifi", newAntijamCmd) }

func newAntijamCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		j, err := c.Antijam(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, j, func(w io.Writer) error { return fields(w, "Mode", j.Mode) })
	}
	cmd := &cobra.Command{
		Use:   "antijam",
		Short: "Manage anti-interference",
		Long: `Manage anti-interference (WiFi Settings), which lets the router switch WiFi
channel to avoid interference.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the anti-interference mode",
		Args:  cobra.NoArgs,
		RunE:  show,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set <auto|enable|disable>",
		Short: "Set the anti-interference mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "auto", "enable", "disable":
			default:
				return fmt.Errorf("invalid mode %q: want auto, enable or disable", args[0])
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Antijam, c.SetAntijam, func(j *tenda.Antijam) error { j.Mode = args[0]; return nil }, "Anti-interference set to "+args[0])
		},
	})
	return cmd
}
