package tenda

import (
	"context"
	"errors"
	"net/url"
)

// DMZ is "DMZ Host" (Advanced Settings): GetDMZCfg / SetDMZCfg.
type DMZ struct {
	Enabled bool   `json:"enabled"`
	HostIP  string `json:"hostIp"`
	LANIP   string `json:"lanIp"`   // read-only
	LANMask string `json:"lanMask"` // read-only
}

type dmzWire struct {
	LanIp   string `json:"lanIp"`
	LanMask string `json:"lanMask"`
	DmzEn   Flag   `json:"dmzEn"`
	DmzIp   string `json:"dmzIp"`
}

// ErrDMZHostIsLAN is SetDMZCfg errCode 2.
var ErrDMZHostIsLAN = errors.New("DMZ host IP equals the router's LAN IP")

// DMZ reads GetDMZCfg.
func (c *Client) DMZ(ctx context.Context) (DMZ, error) {
	var w dmzWire
	if err := c.get(ctx, "GetDMZCfg", nil, &w); err != nil {
		return DMZ{}, err
	}
	return DMZ{Enabled: bool(w.DmzEn), HostIP: w.DmzIp, LANIP: w.LanIp, LANMask: w.LanMask}, nil
}

// SetDMZ sends the full UI form. Disabling resends the last HostIP, as the
// UI does.
func (c *Client) SetDMZ(ctx context.Context, d DMZ) error {
	err := c.set(ctx, "SetDMZCfg", url.Values{"dmzEn": {flag(d.Enabled)}, "dmzIp": {d.HostIP}})
	return mapCode(err, 2, ErrDMZHostIsLAN)
}
