package tenda

import (
	"context"
	"net/url"
)

// WPS is "WPS" (WiFi Settings): WifiWpsGet / WifiWpsSet / WifiWpsStart.
type WPS struct {
	Enabled bool   `json:"enabled"`
	PIN     string `json:"pin"`
	APMode  bool   `json:"apMode"`  // read-only; WPS needs this true
	RadioOn bool   `json:"radioOn"` // read-only; 2.4 GHz radio on, which WPS needs
}

type wpsWire struct {
	WpsEn   Flag   `json:"wpsEn"`
	PinCode string `json:"pinCode"`
	WlMode  string `json:"wl_mode"`
	WlEn    Flag   `json:"wl_en"`
}

// WPS reads WifiWpsGet.
func (c *Client) WPS(ctx context.Context) (WPS, error) {
	var w wpsWire
	if err := c.get(ctx, "WifiWpsGet", nil, &w); err != nil {
		return WPS{}, err
	}
	return WPS{Enabled: bool(w.WpsEn), PIN: w.PinCode, APMode: w.WlMode == "ap", RadioOn: bool(w.WlEn)}, nil
}

// SetWPS toggles WPS.
func (c *Client) SetWPS(ctx context.Context, w WPS) error {
	return c.set(ctx, "WifiWpsSet", url.Values{"wpsEn": {flag(w.Enabled)}})
}

// StartWPS starts a 2-minute WPS push-button pairing session.
func (c *Client) StartWPS(ctx context.Context) error {
	return c.set(ctx, "WifiWpsStart", url.Values{"action": {"wps"}})
}
