package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("system", newSysLogCmd) }

func newSysLogCmd(a *app) *cobra.Command {
	var download string
	cmd := &cobra.Command{
		Use:   "log [--download FILE]",
		Short: "Show or download the system log",
		Long:  `Show "System Log" (System Settings), or save it as a .tar archive with --download.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if download != "" {
				f, err := os.OpenFile(download, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					return err
				}
				n, err := c.DownloadSysLog(ctx, f)
				if cerr := f.Close(); err == nil {
					err = cerr
				}
				if err != nil {
					return err
				}
				return a.done(cmd, "Wrote %d bytes to %s", n, download)
			}
			entries, err := c.SysLog(ctx)
			if err != nil {
				return err
			}
			return a.render(cmd, entries, func(w io.Writer) error { return sysLogText(w, entries) })
		},
	}
	cmd.Flags().StringVar(&download, "download", "", "save the log as a .tar archive to this file instead of listing it")
	return cmd
}

// sysLogText lists entries newest-first, matching js/system_log.js's re-sort
// of GetSySLogCfg's oldest-first response; the JSON output keeps the wire order.
func sysLogText(w io.Writer, entries []tenda.SysLogEntry) error {
	if len(entries) == 0 {
		_, err := fmt.Fprintln(w, "No log entries")
		return err
	}
	sorted := slices.Clone(entries)
	slices.Reverse(sorted)
	rows := make([][]string, len(sorted))
	for i, e := range sorted {
		rows[i] = []string{strconv.Itoa(e.Index), e.Time, e.Type, e.Log}
	}
	return table(w, []col{{title: "INDEX", right: true}, {title: "TIME"}, {title: "TYPE"}, {title: "LOG"}}, rows)
}
