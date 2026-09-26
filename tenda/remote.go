package tenda

import (
	"context"
	"net/url"
	"strconv"
)

// Remote is "Remote Management" (System Settings): GetRemoteWebCfg /
// SetRemoteWebCfg. It exposes the router's web admin UI to the WAN.
type Remote struct {
	Enabled     bool   `json:"enabled"`
	FromIP      string `json:"fromIp"` // "0.0.0.0" allows any WAN source
	Port        int    `json:"port"`
	PasswordSet bool   `json:"passwordSet"` // read-only; the UI blocks enabling without an admin password
	LANIP       string `json:"lanIp"`       // read-only
	LANMask     string `json:"lanMask"`     // read-only
}

type remoteWire struct {
	Syspwdflag  Flag   `json:"syspwdflag"`
	LanIp       string `json:"lanIp"`
	LanMask     string `json:"lanMask"`
	WlGuestIp   string `json:"wlGuestIp"`
	RemoteWebEn Flag   `json:"remoteWebEn"`
	RemoteIp    string `json:"remoteIp"`
	RemotePort  Number `json:"remotePort"`
}

// Remote reads GetRemoteWebCfg.
func (c *Client) Remote(ctx context.Context) (Remote, error) {
	var w remoteWire
	if err := c.get(ctx, "GetRemoteWebCfg", nil, &w); err != nil {
		return Remote{}, err
	}
	return Remote{
		Enabled:     bool(w.RemoteWebEn),
		FromIP:      w.RemoteIp,
		Port:        int(w.RemotePort),
		PasswordSet: bool(w.Syspwdflag),
		LANIP:       w.LanIp,
		LANMask:     w.LanMask,
	}, nil
}

// SetRemote sends the full UI form.
func (c *Client) SetRemote(ctx context.Context, r Remote) error {
	return c.set(ctx, "SetRemoteWebCfg", url.Values{
		"remoteWebEn": {flag(r.Enabled)},
		"remoteIp":    {r.FromIP},
		"remotePort":  {strconv.Itoa(r.Port)},
	})
}
