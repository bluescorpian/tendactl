package tenda

import (
	"context"
	"net/url"
)

// LED is "LED Control" (System Settings): GetLEDCfg / SetLEDCfg.
type LED struct {
	Mode          string `json:"mode"`          // "open" (always on), "close" (always off) or "time" (scheduled)
	Time          string `json:"time"`          // "HH:MM-HH:MM"; meaningful when Mode == "time"
	CloseType     string `json:"closeType"`     // "allClose" or "unpowerClose"
	PowerSaveTime string `json:"powerSaveTime"` // read-only; Sleeping Mode's window, for overlap warnings
}

type ledWire struct {
	LedType       string `json:"ledType"`
	Time          string `json:"time"`
	LedCloseType  string `json:"ledCloseType"`
	PowerSaveTime string `json:"powerSaveTime"`
	TimeUp        string `json:"timeUp"` // undocumented beyond gating a UI tip; unused
}

// LED reads GetLEDCfg.
func (c *Client) LED(ctx context.Context) (LED, error) {
	var w ledWire
	if err := c.get(ctx, "GetLEDCfg", nil, &w); err != nil {
		return LED{}, err
	}
	return LED{Mode: w.LedType, Time: w.Time, CloseType: w.LedCloseType, PowerSaveTime: w.PowerSaveTime}, nil
}

// SetLED sends the full UI form. When Mode != "time", the UI resends the
// previously-fetched Time unchanged rather than clearing it; callers using
// update[T] get that for free, since Time is only changed when Mode ==
// "time".
func (c *Client) SetLED(ctx context.Context, l LED) error {
	return c.set(ctx, "SetLEDCfg", url.Values{"ledType": {l.Mode}, "time": {l.Time}, "ledCloseType": {l.CloseType}})
}
