package tenda

import (
	"context"
	"io"
)

// DownloadBackup streams the full router configuration
// (cgi-bin/DownloadCfg/RouterCfm.cfg) to w. The file embeds every secret
// (WiFi keys, PPPoE/VPN credentials): treat it as sensitive.
func (c *Client) DownloadBackup(ctx context.Context, w io.Writer) (int64, error) {
	return c.download(ctx, "cgi-bin/DownloadCfg/RouterCfm.cfg", w)
}
