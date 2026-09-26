package tenda

import (
	"context"
	"net/url"
)

// WiFiBasic is "WiFi Name & Password" (WiFi Settings tab: WifiBasicGet /
// WifiBasicSet), plus that tab's tile-status summary (GetWrlStatus).
type WiFiBasic struct {
	Band24        WiFiRadio `json:"band24"`
	Band5         WiFiRadio `json:"band5"`
	ScheduleOn    bool      `json:"scheduleEnabled"`
	WPSOn         bool      `json:"wpsEnabled"`
	BeamformingOn bool      `json:"beamformingEnabled"`
	APModeOn      bool      `json:"apModeEnabled"`
	RepeatingOn   bool      `json:"repeatingEnabled"`
	Antijam       string    `json:"antijam"` // "on", "off" or "auto"
}

// WiFiRadio is one band's SSID, password and enable/hide state.
type WiFiRadio struct {
	Enabled  bool   `json:"enabled"`
	SSID     string `json:"ssid"`
	Hidden   bool   `json:"hidden"`
	Security string `json:"security"` // "none", "wpapsk", "wpa2psk" or "wpawpa2psk"
	Password string `json:"password"`
}

type wifiBasicWire struct {
	WrlEn          Flag   `json:"wrlEn"`
	WrlEn5g        Flag   `json:"wrlEn_5g"`
	Ssid           string `json:"ssid"`
	Ssid5g         string `json:"ssid_5g"`
	Security       string `json:"security"`
	Security5g     string `json:"security_5g"`
	WrlPwd         string `json:"wrlPwd"`
	WrlPwd5g       string `json:"wrlPwd_5g"`
	HideSsid       Flag   `json:"hideSsid"`
	HideSsid5g     Flag   `json:"hideSsid_5g"`
	WpapskType     string `json:"wpapsk_type"` // present but unused by the UI's JS
	WpapskType5g   string `json:"wpapsk_type_5g"`
	WpapskCrypto   string `json:"wpapsk_crypto"`
	WpapskCrypto5g string `json:"wpapsk_crypto_5g"`
	Uptime         string `json:"uptime"` // unused by the UI's JS
	Uptime5g       string `json:"uptime_5g"`
}

type wrlStatusWire struct {
	SchedWifiEn   string `json:"schedWifiEn"`
	WispEn        string `json:"wispEn"`
	WpsEn         string `json:"wpsEn"`
	NamePwd       string `json:"namePwd"` // duplicates wrlEn/wrlEn_5g's on/off; unused
	Signal        string `json:"signal"`  // duplicated precisely by WifiPower's power/power_5g; unused
	Beamforming   string `json:"beamforming"`
	ApMode        string `json:"apMode"`
	WifiAntijamEn string `json:"WifiAntijamEn"`
}

// WiFiBasic reads WifiBasicGet and GetWrlStatus.
func (c *Client) WiFiBasic(ctx context.Context) (WiFiBasic, error) {
	var w wifiBasicWire
	if err := c.get(ctx, "WifiBasicGet", nil, &w); err != nil {
		return WiFiBasic{}, err
	}
	var s wrlStatusWire
	if err := c.get(ctx, "GetWrlStatus", nil, &s); err != nil {
		return WiFiBasic{}, err
	}
	return WiFiBasic{
		Band24:        WiFiRadio{Enabled: bool(w.WrlEn), SSID: w.Ssid, Hidden: bool(w.HideSsid), Security: w.Security, Password: w.WrlPwd},
		Band5:         WiFiRadio{Enabled: bool(w.WrlEn5g), SSID: w.Ssid5g, Hidden: bool(w.HideSsid5g), Security: w.Security5g, Password: w.WrlPwd5g},
		ScheduleOn:    s.SchedWifiEn == "1",
		WPSOn:         s.WpsEn == "1",
		BeamformingOn: s.Beamforming == "1",
		APModeOn:      s.ApMode == "1",
		RepeatingOn:   s.WispEn != "0",
		Antijam:       antijamLabel(s.WifiAntijamEn),
	}, nil
}

// antijamLabel matches main.js's signalMsg-style catch-all: only the literal
// strings "true"/"false" are on/off, anything else (including the live
// value "auto") means Auto.
func antijamLabel(s string) string {
	switch s {
	case "true":
		return "on"
	case "false":
		return "off"
	default:
		return "auto"
	}
}

// SetWiFiBasic sends the full UI form. prev is the value most recently read,
// used to build the WifiBasicSet hazard rule's Before values so that only
// actually turning an on band off needs --yes.
func (c *Client) SetWiFiBasic(ctx context.Context, prev, next WiFiBasic) error {
	return c.setWith(ctx, "WifiBasicSet", wifiBasicForm(next), wifiBasicForm(prev))
}

func wifiBasicForm(b WiFiBasic) url.Values {
	return url.Values{
		"wrlEn":       {flag(b.Band24.Enabled)},
		"wrlEn_5g":    {flag(b.Band5.Enabled)},
		"security":    {b.Band24.Security},
		"security_5g": {b.Band5.Security},
		"ssid":        {b.Band24.SSID},
		"ssid_5g":     {b.Band5.SSID},
		"hideSsid":    {flag(b.Band24.Hidden)},
		"hideSsid_5g": {flag(b.Band5.Hidden)},
		"wrlPwd":      {b.Band24.Password},
		"wrlPwd_5g":   {b.Band5.Password},
	}
}
