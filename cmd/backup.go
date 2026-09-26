package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() { register("system", newBackupCmd) }

func newBackupCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "backup <file>",
		Short: "Download the router's full configuration backup",
		Long: `Download "Backup/Restore" (System Settings) as a .cfg file. The file embeds
every secret on the router (WiFi keys, PPPoE/VPN credentials): treat it as
sensitive. Refuses to overwrite an existing file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			f, err := os.OpenFile(args[0], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			n, err := c.DownloadBackup(cmd.Context(), f)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
			return a.done(cmd, "Wrote %d bytes to %s", n, args[0])
		},
	}
}
