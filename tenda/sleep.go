package tenda

import (
	"context"
	"net/url"
)

// Sleep is "Sleeping Mode" (Advanced Settings): PowerSaveGet / PowerSaveSet.
// Enabling it turns WiFi (and LEDs, and the USB port on capable models) off
// during Window. Editing is blocked in the UI while WorkModeOK is false
// (Wireless Repeating enabled); tendactl does not enforce that client-side.
type Sleep struct {
	Enabled bool   `json:"enabled"`
	Window  string `json:"window"` // "HH:MM-HH:MM"
	LEDs    string `json:"leds"`   // "allClose" (all off) or "unpowerClose" (all but power)
	Delay   bool   `json:"delay"`  // delay enabling while a client is online

	WorkModeOK         bool   `json:"workModeOk"`         // read-only; editable only in AP mode
	LEDWindow          string `json:"ledWindow"`          // read-only; LED Control's schedule, for overlap warnings
	WiFiScheduleWindow string `json:"wifiScheduleWindow"` // read-only; WiFi Schedule's window, for overlap warnings
	TimeSynced         bool   `json:"timeSynced"`         // read-only; system clock synced with internet time
}

type sleepWire struct {
	PowerSavingEn  Number `json:"powerSavingEn"`
	PowerSaveDelay Number `json:"powerSaveDelay"`
	Time           string `json:"time"`
	WlMode         string `json:"wl_mode"`
	LedTime        string `json:"ledTime"`
	WifiTime       string `json:"wifiTime"`
	LedCloseType   string `json:"ledCloseType"`
	TimeUp         string `json:"timeUp"`
}

// Sleep reads PowerSaveGet.
func (c *Client) Sleep(ctx context.Context) (Sleep, error) {
	var w sleepWire
	if err := c.get(ctx, "PowerSaveGet", nil, &w); err != nil {
		return Sleep{}, err
	}
	return Sleep{
		Enabled:            w.PowerSavingEn != 0,
		Window:             w.Time,
		LEDs:               w.LedCloseType,
		Delay:              w.PowerSaveDelay != 0,
		WorkModeOK:         w.WlMode == "ap",
		LEDWindow:          w.LedTime,
		WiFiScheduleWindow: w.WifiTime,
		TimeSynced:         w.TimeUp == "1",
	}, nil
}

// SetSleep sends the full UI form. prev is the value most recently read,
// used to build the PowerSaveSet hazard rule's Before values: it fires only
// when enabling with values that differ from what's already running.
func (c *Client) SetSleep(ctx context.Context, prev, next Sleep) error {
	return c.setWith(ctx, "PowerSaveSet", sleepForm(next), sleepForm(prev))
}

func sleepForm(s Sleep) url.Values {
	return url.Values{
		"powerSavingEn":  {flag(s.Enabled)},
		"time":           {s.Window},
		"ledCloseType":   {s.LEDs},
		"powerSaveDelay": {flag(s.Delay)},
	}
}
