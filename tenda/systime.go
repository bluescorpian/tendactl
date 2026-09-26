package tenda

import (
	"context"
	"net/url"
	"strconv"
)

// SysTime is "Time Settings" (System Settings): GetSysTimeCfg /
// SetSysTimeCfg. TimeZone is the GMT offset plus 12 hours, as "HH:MM": "12:00"
// is GMT, "14:00" is GMT+02:00.
type SysTime struct {
	TimeZone      string `json:"timeZone"`
	Time          string `json:"time"`          // read-only
	Synced        bool   `json:"synced"`        // read-only
	NTPServer     string `json:"ntpServer"`     // read-only in this UI; resent unchanged
	ResyncSeconds int    `json:"resyncSeconds"` // read-only in this UI; resent unchanged
}

type sysTimeWire struct {
	TimeType           string `json:"timeType"`
	TimeZone           string `json:"timeZone"`
	TimePeriod         Number `json:"timePeriod"`
	NtpServer          string `json:"ntpServer"`
	Time               string `json:"time"`
	IsSyncInternetTime Flag   `json:"isSyncInternetTime"`
}

// SysTime reads GetSysTimeCfg.
func (c *Client) SysTime(ctx context.Context) (SysTime, error) {
	var w sysTimeWire
	if err := c.get(ctx, "GetSysTimeCfg", nil, &w); err != nil {
		return SysTime{}, err
	}
	return SysTime{
		TimeZone:      w.TimeZone,
		Time:          w.Time,
		Synced:        bool(w.IsSyncInternetTime),
		NTPServer:     w.NtpServer,
		ResyncSeconds: int(w.TimePeriod),
	}, nil
}

// SetSysTime sends the full UI form. TimePeriod and NTPServer are not
// user-editable in this build (their controls are commented out); the UI
// always resends the values from the last GetSysTimeCfg, which SetSysTime
// does too.
func (c *Client) SetSysTime(ctx context.Context, t SysTime) error {
	return c.set(ctx, "SetSysTimeCfg", url.Values{
		"timeZone":   {t.TimeZone},
		"timePeriod": {strconv.Itoa(t.ResyncSeconds)},
		"ntpServer":  {t.NTPServer},
	})
}
