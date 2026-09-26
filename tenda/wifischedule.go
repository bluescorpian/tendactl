package tenda

import (
	"context"
	"net/url"
	"strings"
)

// wifiScheduleDays is the wire day order, Monday first (docs/router-api.md
// UI names).
var wifiScheduleDays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

// WiFiSchedule is "WiFi Schedule" (WiFi Settings): initSchedWifi /
// openSchedWifi. Days is meaningful only when EveryDay is false.
type WiFiSchedule struct {
	Enabled         bool     `json:"enabled"`
	Start           string   `json:"start"` // "HH:MM"; WiFi turns off from Start to End
	End             string   `json:"end"`
	EveryDay        bool     `json:"everyDay"`
	Days            []string `json:"days"`            // e.g. ["mon","tue"]
	PowerSaveWindow string   `json:"powerSaveWindow"` // read-only; Sleeping Mode's window, "" if off
	WorkModeOK      bool     `json:"workModeOk"`      // read-only; the schedule is only editable in AP mode
	TimeSynced      bool     `json:"timeSynced"`      // read-only; false means the schedule can't yet take effect
}

type wifiScheduleWire struct {
	WifiEn          Number `json:"wifiEn"` // present but unused by this page's JS
	SchedWifiEnable Number `json:"schedWifiEnable"`
	SchedStartTime  string `json:"schedStartTime"`
	SchedEndTime    string `json:"schedEndTime"`
	TimeType        string `json:"timeType"`
	Day             string `json:"day"`
	PowerSaveTime   string `json:"powerSaveTime"`
	WlMode          string `json:"wl_mode"`
	TimeUp          string `json:"timeUp"`
}

// WiFiSchedule reads initSchedWifi.
func (c *Client) WiFiSchedule(ctx context.Context) (WiFiSchedule, error) {
	var w wifiScheduleWire
	if err := c.get(ctx, "initSchedWifi", nil, &w); err != nil {
		return WiFiSchedule{}, err
	}
	return WiFiSchedule{
		Enabled:         w.SchedWifiEnable != 0,
		Start:           w.SchedStartTime,
		End:             w.SchedEndTime,
		EveryDay:        w.TimeType == "0",
		Days:            decodeScheduleDays(w.Day),
		PowerSaveWindow: w.PowerSaveTime,
		WorkModeOK:      w.WlMode == "ap",
		TimeSynced:      w.TimeUp == "1",
	}, nil
}

func decodeScheduleDays(raw string) []string {
	flags := strings.Split(raw, ",")
	var days []string
	for i, f := range flags {
		if i < len(wifiScheduleDays) && f == "1" {
			days = append(days, wifiScheduleDays[i])
		}
	}
	return days
}

func encodeScheduleDays(days []string) string {
	flags := make([]string, len(wifiScheduleDays))
	for i := range flags {
		flags[i] = "0"
	}
	for _, d := range days {
		for i, name := range wifiScheduleDays {
			if d == name {
				flags[i] = "1"
			}
		}
	}
	return strings.Join(flags, ",")
}

// SetWiFiSchedule sends the full UI form. prev is the value most recently
// read, used to build the openSchedWifi hazard rule's Before values: it
// fires only when enabling with values that differ from what's already
// running.
func (c *Client) SetWiFiSchedule(ctx context.Context, prev, next WiFiSchedule) error {
	return c.setWith(ctx, "openSchedWifi", wifiScheduleForm(prev, next), wifiScheduleForm(prev, prev))
}

// wifiScheduleForm builds the openSchedWifi form. The UI's getSubmitData()
// only takes schedStartTime/schedEndTime/timeType/day from the live form
// when schedWifiEnable == 1; otherwise it resends the last-read (prev)
// values unchanged, regardless of what next asks to change
// (js/wifi_time.js moduleModel.getSubmitData).
func wifiScheduleForm(prev, next WiFiSchedule) url.Values {
	start, end, everyDay, days := next.Start, next.End, next.EveryDay, next.Days
	if !next.Enabled {
		start, end, everyDay, days = prev.Start, prev.End, prev.EveryDay, prev.Days
	}
	timeType := "1"
	if everyDay {
		timeType = "0"
	}
	return url.Values{
		"schedWifiEnable": {flag(next.Enabled)},
		"schedStartTime":  {start},
		"schedEndTime":    {end},
		"timeType":        {timeType},
		"day":             {encodeScheduleDays(days)},
	}
}
