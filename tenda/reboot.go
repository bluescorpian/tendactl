package tenda

import (
	"context"
	"net/http"
	"net/url"
)

// Reboot reboots the router (system_reboot.html's "Reboot" button,
// SysToolReboot). It is on the hazard list: the caller needs --yes.
func (c *Client) Reboot(ctx context.Context) error {
	_, err := c.submit(ctx, http.MethodPost, "SysToolReboot", url.Values{"action": {"0"}})
	return err
}
