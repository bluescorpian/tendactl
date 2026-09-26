package cmd

import (
	"io"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("wifi", newBeamformingCmd) }

func newBeamformingCmd(a *app) *cobra.Command {
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		b, err := c.Beamforming(cmd.Context())
		if err != nil {
			return err
		}
		return a.render(cmd, b, func(w io.Writer) error { return fields(w, "Enabled", onOff(b.Enabled)) })
	}
	cmd := &cobra.Command{
		Use:   "beamforming",
		Short: "Manage Beamforming+",
		Long:  `Manage Beamforming+ (WiFi Settings), which focuses signal toward connected devices.`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show whether Beamforming+ is enabled",
		Args:  cobra.NoArgs,
		RunE:  show,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Enable Beamforming+",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Beamforming, c.SetBeamforming, func(b *tenda.Beamforming) error { b.Enabled = true; return nil }, "Beamforming+ enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Disable Beamforming+",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.Beamforming, c.SetBeamforming, func(b *tenda.Beamforming) error { b.Enabled = false; return nil }, "Beamforming+ disabled")
		},
	})
	return cmd
}
