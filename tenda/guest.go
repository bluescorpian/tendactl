package tenda

import (
	"context"
	"net/url"
	"strconv"
)

// Guest is the "Guest Network" tab: WifiGuestGet / WifiGuestSet. Both bands
// share one password and one security mode, matching the UI's single field.
type Guest struct {
	Enabled        bool   `json:"enabled"`
	SSID           string `json:"ssid"`
	SSID5g         string `json:"ssid5g"`
	Password       string `json:"password"`
	EffectiveTime  string `json:"effectiveTime"` // "4", "8" (hours) or "0" (Always)
	ShareSpeedMbps int    `json:"shareSpeedMbps"`
}

type guestGetWire struct {
	WlEn          Flag   `json:"wl_en"`   // main radio on/off; the form is disabled unless this and wl_mode == "ap"
	WlMode        string `json:"wl_mode"` // must be "ap" for the guest network to be editable
	GuestEn       Flag   `json:"guestEn"`
	GuestEn5g     Flag   `json:"guestEn_5g"` // always equal to GuestEn
	HideSsid      string `json:"hideSsid"`   // not read by main.js on this tab; unused
	GuestSsid     string `json:"guestSsid"`
	GuestWrlPwd   string `json:"guestWrlPwd"`
	HideSsid5g    string `json:"hideSsid_5g"` // unused
	GuestSsid5g   string `json:"guestSsid_5g"`
	GuestWrlPwd5g string `json:"guestWrlPwd_5g"`
	EffectiveTime string `json:"effectiveTime"`
	ShareSpeed    string `json:"shareSpeed"`
}

// Guest reads WifiGuestGet.
func (c *Client) Guest(ctx context.Context) (Guest, error) {
	var w guestGetWire
	if err := c.get(ctx, "WifiGuestGet", nil, &w); err != nil {
		return Guest{}, err
	}
	speed, err := strconv.Atoi(w.ShareSpeed)
	if err != nil {
		return Guest{}, &DecodeError{Endpoint: "WifiGuestGet", Err: err}
	}
	return Guest{
		Enabled:        bool(w.GuestEn),
		SSID:           w.GuestSsid,
		SSID5g:         w.GuestSsid5g,
		Password:       w.GuestWrlPwd,
		EffectiveTime:  w.EffectiveTime,
		ShareSpeedMbps: speed / 128,
	}, nil
}

// SetGuest sends the full UI form. guestSecurity/guestSecurity_5g are
// unconditionally "wpapsk", matching the UI's own bug: it reads a
// nonexistent #wrlPwd element, so its blank-password fallback to "none"
// never fires. Password is one field, sent for both bands, as the UI does.
func (c *Client) SetGuest(ctx context.Context, g Guest) error {
	form := url.Values{
		"guestEn":          {flag(g.Enabled)},
		"guestEn_5g":       {flag(g.Enabled)},
		"guestSecurity":    {"wpapsk"},
		"guestSecurity_5g": {"wpapsk"},
		"guestSsid":        {g.SSID},
		"guestSsid_5g":     {g.SSID5g},
		"guestWrlPwd":      {g.Password},
		"guestWrlPwd_5g":   {g.Password},
		"effectiveTime":    {g.EffectiveTime},
		"shareSpeed":       {strconv.Itoa(g.ShareSpeedMbps * 128)},
	}
	return c.set(ctx, "WifiGuestSet", form)
}
