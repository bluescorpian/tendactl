package tenda

import (
	"context"
	"net/url"
)

// Antijam is "Anti-interference" (WiFi Settings): WifiAntijamGet / Set. Mode
// is "auto", "enable" or "disable", since the wire value is a three-way
// enum rather than a plain toggle.
type Antijam struct {
	Mode string `json:"mode"`
}

type antijamWire struct {
	WifiAntijamEn string `json:"WifiAntijamEn"` // "auto", "true" or "false"
}

func antijamModeFromWire(s string) string {
	switch s {
	case "true":
		return "enable"
	case "false":
		return "disable"
	default:
		return "auto"
	}
}

func antijamModeToWire(mode string) string {
	switch mode {
	case "enable":
		return "true"
	case "disable":
		return "false"
	default:
		return "auto"
	}
}

// Antijam reads WifiAntijamGet.
func (c *Client) Antijam(ctx context.Context) (Antijam, error) {
	var w antijamWire
	if err := c.get(ctx, "WifiAntijamGet", nil, &w); err != nil {
		return Antijam{}, err
	}
	return Antijam{Mode: antijamModeFromWire(w.WifiAntijamEn)}, nil
}

// SetAntijam sets the anti-interference mode.
func (c *Client) SetAntijam(ctx context.Context, a Antijam) error {
	return c.set(ctx, "WifiAntijamSet", url.Values{"WifiAntijamEn": {antijamModeToWire(a.Mode)}})
}
