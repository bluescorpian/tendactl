package tenda

import (
	"context"
	"net/url"
)

// WiFiPower is "Transmit Power" (WiFi Settings): WifiPowerGet / WifiPowerSet.
type WiFiPower struct {
	Power   string `json:"power"`   // "low", "middle" or "high"
	Power5g string `json:"power5g"` // this build's 5 GHz UI only offers "low" (labelled "Medium") and "high"
}

type wifiPowerWire struct {
	Power   string `json:"power"`
	Power5g string `json:"power_5g"`
}

// WiFiPower reads WifiPowerGet.
func (c *Client) WiFiPower(ctx context.Context) (WiFiPower, error) {
	var w wifiPowerWire
	if err := c.get(ctx, "WifiPowerGet", nil, &w); err != nil {
		return WiFiPower{}, err
	}
	return WiFiPower{Power: w.Power, Power5g: w.Power5g}, nil
}

// SetWiFiPower sends the full UI form.
func (c *Client) SetWiFiPower(ctx context.Context, p WiFiPower) error {
	return c.set(ctx, "WifiPowerSet", url.Values{"power": {p.Power}, "power_5g": {p.Power5g}})
}
