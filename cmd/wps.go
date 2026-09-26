package cmd

import (
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("wifi", newWPSCmd) }

func newWPSCmd(a *app) *cobra.Command {
	var showPassword bool
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		w, err := c.WPS(cmd.Context())
		if err != nil {
			return err
		}
		w.PIN = maskSecret(w.PIN, showPassword)
		return a.render(cmd, w, func(out io.Writer) error {
			return fields(out,
				"Enabled", onOff(w.Enabled),
				"PIN", w.PIN,
				"Wireless repeating", onOff(!w.APMode),
				"2.4 GHz radio on", onOff(w.RadioOn),
			)
		})
	}
	cmd := &cobra.Command{
		Use:   "wps",
		Short: "Manage WPS",
		Long: `Manage WPS (WiFi Settings). WPS needs the router in AP mode with the 2.4 GHz
radio on.`,
		Args: cobra.NoArgs,
		RunE: show,
	}
	cmd.Flags().BoolVar(&showPassword, "show-password", false, "show the WPS PIN instead of masking it")
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show WPS status",
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	showCmd.Flags().BoolVar(&showPassword, "show-password", false, "show the WPS PIN instead of masking it")
	cmd.AddCommand(showCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Enable WPS",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.WPS, c.SetWPS, func(w *tenda.WPS) error { w.Enabled = true; return nil }, "WPS enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Disable WPS",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.WPS, c.SetWPS, func(w *tenda.WPS) error { w.Enabled = false; return nil }, "WPS disabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start a WPS push-button pairing session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			if err := c.StartWPS(cmd.Context()); err != nil {
				return err
			}
			return a.done(cmd, "WPS pairing started; the window stays open for 2 minutes")
		},
	})
	return cmd
}
