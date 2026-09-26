package tenda

import (
	"context"
	"net/url"
)

// Maintenance is "Automatic Maintenance" (System Settings):
// GetSysAutoRebbotCfg / SetSysAutoRebbotCfg. It schedules a future reboot;
// it is not itself on the hazard list.
type Maintenance struct {
	Enabled     bool   `json:"enabled"`
	RebootTime  string `json:"rebootTime"` // "HH:MM"
	DelayIfBusy bool   `json:"delayIfBusy"`
	TimeSynced  bool   `json:"timeSynced"` // read-only; false means the schedule can't yet take effect
}

type maintenanceWire struct {
	AutoRebootEn  Flag   `json:"autoRebootEn"`
	Time          string `json:"time"`
	RebootTime    string `json:"rebootTime"`
	DelayRebootEn Flag   `json:"delayRebootEn"`
	TimeUp        Flag   `json:"timeUp"`
	Speed         string `json:"speed"`
}

// Maintenance reads GetSysAutoRebbotCfg.
func (c *Client) Maintenance(ctx context.Context) (Maintenance, error) {
	var w maintenanceWire
	if err := c.get(ctx, "GetSysAutoRebbotCfg", nil, &w); err != nil {
		return Maintenance{}, err
	}
	return Maintenance{
		Enabled:     bool(w.AutoRebootEn),
		RebootTime:  w.RebootTime,
		DelayIfBusy: bool(w.DelayRebootEn),
		TimeSynced:  bool(w.TimeUp),
	}, nil
}

// delayFlag spells DelayIfBusy the way SetSysAutoRebbotCfg expects:
// "true"/"false", not "1"/"0".
func delayFlag(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// SetMaintenance sends the full UI form.
func (c *Client) SetMaintenance(ctx context.Context, m Maintenance) error {
	return c.set(ctx, "SetSysAutoRebbotCfg", url.Values{
		"autoRebootEn":  {flag(m.Enabled)},
		"delayRebootEn": {delayFlag(m.DelayIfBusy)},
		"rebootTime":    {m.RebootTime},
	})
}
