package tenda

import (
	"context"
	"net/url"
)

// Beamforming is "Beamforming+" (WiFi Settings): WifiBeamformingGet / Set.
type Beamforming struct {
	Enabled bool `json:"enabled"`
}

type beamformingWire struct {
	BeamformingEn Flag `json:"beamformingEn"`
}

// Beamforming reads WifiBeamformingGet.
func (c *Client) Beamforming(ctx context.Context) (Beamforming, error) {
	var w beamformingWire
	if err := c.get(ctx, "WifiBeamformingGet", nil, &w); err != nil {
		return Beamforming{}, err
	}
	return Beamforming{Enabled: bool(w.BeamformingEn)}, nil
}

// SetBeamforming toggles Beamforming+.
func (c *Client) SetBeamforming(ctx context.Context, b Beamforming) error {
	return c.set(ctx, "WifiBeamformingSet", url.Values{"beamformingEn": {flag(b.Enabled)}})
}
