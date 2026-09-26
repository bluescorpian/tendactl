package tenda

import "context"

// RouterStatus is the dashboard's "Internet Status" tab (GetRouterStatus).
type RouterStatus struct {
	DeviceName  string    `json:"deviceName"`
	WorkMode    string    `json:"workMode"`
	LANIP       string    `json:"lanIp"`
	LANMAC      string    `json:"lanMac"`
	ClientCount int       `json:"clientCount"`
	WiFi24      WiFiBand  `json:"wifi24"`
	WiFi5       WiFiBand  `json:"wifi5"`
	WAN         []WANInfo `json:"wan"`
	Firmware    Firmware  `json:"firmware"`
}

// WiFiBand is one radio's main network as the dashboard shows it.
type WiFiBand struct {
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid"`
}

// WANInfo is one WAN port. Status is the 7-digit code described in the
// doc's Conventions -> Status codes.
type WANInfo struct {
	Status   string `json:"status"`
	IP       string `json:"ip"`
	UpKBps   string `json:"upKBps"`
	DownKBps string `json:"downKBps"`
}

// Firmware is the running version and any update the router knows about.
type Firmware struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
}

type routerStatusWire struct {
	Wl5gEn     Flag   `json:"wl5gEn"`
	Wl5gName   string `json:"wl5gName"`
	Wl24gEn    Flag   `json:"wl24gEn"`
	Wl24gName  string `json:"wl24gName"`
	Lineup     string `json:"lineup"`
	ClientNum  int    `json:"clientNum"`
	BlackNum   int    `json:"blackNum"`
	ListNum    int    `json:"listNum"`
	DeviceName string `json:"deviceName"`
	LanIP      string `json:"lanIP"`
	LanMAC     string `json:"lanMAC"`
	WorkMode   string `json:"workMode"`
	ApStatus   string `json:"apStatus"`
	WanInfo    []struct {
		WanStatus        string `json:"wanStatus"`
		WanIp            string `json:"wanIp"`
		WanUploadSpeed   string `json:"wanUploadSpeed"`
		WanDownloadSpeed string `json:"wanDownloadSpeed"`
	} `json:"wanInfo"`
	OnlineUpgradeInfo struct {
		NewVersionExist Flag   `json:"newVersionExist"`
		NewVersion      string `json:"newVersion"`
		CurVersion      string `json:"curVersion"`
	} `json:"onlineUpgradeInfo"`
}

// RouterStatus reads GetRouterStatus.
func (c *Client) RouterStatus(ctx context.Context) (RouterStatus, error) {
	var w routerStatusWire
	if err := c.get(ctx, "GetRouterStatus", nil, &w); err != nil {
		return RouterStatus{}, err
	}
	s := RouterStatus{
		DeviceName:  w.DeviceName,
		WorkMode:    w.WorkMode,
		LANIP:       w.LanIP,
		LANMAC:      w.LanMAC,
		ClientCount: w.ClientNum,
		WiFi24:      WiFiBand{Enabled: bool(w.Wl24gEn), SSID: w.Wl24gName},
		WiFi5:       WiFiBand{Enabled: bool(w.Wl5gEn), SSID: w.Wl5gName},
		WAN:         make([]WANInfo, 0, len(w.WanInfo)),
		Firmware: Firmware{
			Current:         w.OnlineUpgradeInfo.CurVersion,
			Latest:          w.OnlineUpgradeInfo.NewVersion,
			UpdateAvailable: bool(w.OnlineUpgradeInfo.NewVersionExist),
		},
	}
	for _, wan := range w.WanInfo {
		s.WAN = append(s.WAN, WANInfo{Status: wan.WanStatus, IP: wan.WanIp, UpKBps: wan.WanUploadSpeed, DownKBps: wan.WanDownloadSpeed})
	}
	return s, nil
}
