package tenda

import "context"

// systemWiFiConnectTypes is the UI's index into adv_connect_type (System
// Status field labels).
var systemWiFiConnectTypes = []string{"Dynamic IP Address", "Static IP Address", "PPPoE", "Russia PPTP", "Russia L2TP", "Russia PPPoE"}

// systemWANConnectStatuses is the UI's index into wanInfo[].adv_connect_status.
var systemWANConnectStatuses = []string{"Cable disconnected", "Disconnected", "Connecting", "Connected"}

// SystemStatus is "System Status" (System Settings tab), with the same
// tab's own tile overview folded in (GetSystemStatus, GetSysStatus).
type SystemStatus struct {
	Time          string          `json:"time"`
	UptimeSeconds int             `json:"uptimeSeconds"`
	Firmware      string          `json:"firmware"`
	Hardware      string          `json:"hardware"`
	WAN           []SystemWANInfo `json:"wan"`
	LANIP         string          `json:"lanIp"`
	LANMask       string          `json:"lanMask"`
	LANMAC        string          `json:"lanMac"`
	WiFi24        SystemWiFiInfo  `json:"wifi24"`
	WiFi5         SystemWiFiInfo  `json:"wifi5"`
	// WiFi5Enabled is the 5 GHz radio's on/off state. There is no reliable
	// equivalent for 2.4 GHz: the doc's System Status field labels map no UI
	// on/off control to adv_wrl_en, which is the hidden-SSID flag instead.
	WiFi5Enabled bool `json:"wifi5Enabled"`

	AutoMaintenanceEnabled    bool `json:"autoMaintenanceEnabled"`
	RemoteManagementEnabled   bool `json:"remoteManagementEnabled"`
	DHCPReservationConfigured bool `json:"dhcpReservationConfigured"`
	TimeSynced                bool `json:"timeSynced"`
}

// SystemWANInfo is one WAN port on the System Status page. ConnectStatus
// indexes systemWANConnectStatuses; ConnectType indexes
// systemWiFiConnectTypes as its numeral string.
type SystemWANInfo struct {
	ConnectStatus int    `json:"connectStatus"`
	ConnectType   string `json:"connectType"`
	UptimeSeconds string `json:"uptimeSeconds"`
	IP            string `json:"ip"`
	Mask          string `json:"mask"`
	Gateway       string `json:"gateway"`
	DNS1          string `json:"dns1"`
	DNS2          string `json:"dns2"`
	MAC           string `json:"mac"`
	UpKBps        string `json:"upKBps"`
	DownKBps      string `json:"downKBps"`
}

// SystemWiFiInfo is one radio's status on the System Status page. Hidden is
// the SSID-hidden flag (adv_wrl_en's inverted naming), not radio-on/off.
type SystemWiFiInfo struct {
	Hidden   bool   `json:"hidden"`
	SSID     string `json:"ssid"`
	Security string `json:"security"`
	Channel  string `json:"channel"`
	Band     string `json:"band"`
	MAC      string `json:"mac"`
}

type systemStatusWire struct {
	AdvSysTime string `json:"adv_sys_time"`
	AdvRunTime int    `json:"adv_run_time"`
	AdvFirmVer string `json:"adv_firm_ver"`
	AdvHardVer string `json:"adv_hard_ver"`
	WanInfo    []struct {
		AdvConnectStatus int    `json:"adv_connect_status"`
		AdvConnectTime   string `json:"adv_connect_time"`
		AdvIp            string `json:"adv_ip"`
		AdvMask          string `json:"adv_mask"`
		AdvGateway       string `json:"adv_gateway"`
		WanUploadSpeed   string `json:"wanUploadSpeed"`
		WanDownloadSpeed string `json:"wanDownloadSpeed"`
		AdvConnectType   string `json:"adv_connect_type"`
		AdvDns1          string `json:"adv_dns1"`
		AdvDns2          string `json:"adv_dns2"`
		AdvMac           string `json:"adv_mac"`
	} `json:"wanInfo"`
	AdvLanIp   string `json:"adv_lan_ip"`
	AdvLanMask string `json:"adv_lan_mask"`
	AdvLanMac  string `json:"adv_lan_mac"`

	AdvWrlEn      Flag   `json:"adv_wrl_en"`
	AdvWrlSsid    string `json:"adv_wrl_ssid"`
	AdvWrlSec     string `json:"adv_wrl_sec"`
	AdvWrlChannel string `json:"adv_wrl_channel"`
	AdvWrlBand    string `json:"adv_wrl_band"`
	AdvWrlMac     string `json:"adv_wrl_mac"`

	// The 5 GHz fields below are absent from the JSON entirely while that
	// radio is off, which is the only state the fixture captures; they
	// decode to their zero value in that case.
	WifiEnable5g    Flag   `json:"wifi_enable_5g"`
	AdvWrlEn5g      Flag   `json:"adv_wrl_en_5g"`
	AdvWrlSsid5g    string `json:"adv_wrl_ssid_5g"`
	AdvWrlSec5g     string `json:"adv_wrl_sec_5g"`
	AdvWrlChannel5g string `json:"adv_wrl_channel_5g"`
	AdvWrlBand5g    string `json:"adv_wrl_band_5g"`
	AdvWrlMac5g     string `json:"adv_wrl_mac_5g"`
}

type sysStatusWire struct {
	Firmware         string `json:"firmware"`
	RebootEn         Flag   `json:"rebootEn"`
	RemoteWeb        Flag   `json:"remoteWeb"`
	WlMode           string `json:"wl_mode"`
	Lan              string `json:"lan"`
	SyncInternetTime Flag   `json:"syncInternetTime"`
	ApClientConnect  Flag   `json:"apClientConnect"`
	IpMacBindEn      Flag   `json:"ipMacBindEn"`
}

// SystemStatus reads GetSystemStatus and GetSysStatus.
func (c *Client) SystemStatus(ctx context.Context) (SystemStatus, error) {
	var w systemStatusWire
	if err := c.get(ctx, "GetSystemStatus", nil, &w); err != nil {
		return SystemStatus{}, err
	}
	var sw sysStatusWire
	if err := c.get(ctx, "GetSysStatus", nil, &sw); err != nil {
		return SystemStatus{}, err
	}
	s := SystemStatus{
		Time:          w.AdvSysTime,
		UptimeSeconds: w.AdvRunTime,
		Firmware:      w.AdvFirmVer,
		Hardware:      w.AdvHardVer,
		WAN:           make([]SystemWANInfo, 0, len(w.WanInfo)),
		LANIP:         w.AdvLanIp,
		LANMask:       w.AdvLanMask,
		LANMAC:        w.AdvLanMac,
		WiFi24: SystemWiFiInfo{
			Hidden: bool(w.AdvWrlEn), SSID: w.AdvWrlSsid, Security: w.AdvWrlSec,
			Channel: w.AdvWrlChannel, Band: w.AdvWrlBand, MAC: w.AdvWrlMac,
		},
		WiFi5: SystemWiFiInfo{
			Hidden: bool(w.AdvWrlEn5g), SSID: w.AdvWrlSsid5g, Security: w.AdvWrlSec5g,
			Channel: w.AdvWrlChannel5g, Band: w.AdvWrlBand5g, MAC: w.AdvWrlMac5g,
		},
		WiFi5Enabled:              bool(w.WifiEnable5g),
		AutoMaintenanceEnabled:    bool(sw.RebootEn),
		RemoteManagementEnabled:   bool(sw.RemoteWeb),
		DHCPReservationConfigured: bool(sw.IpMacBindEn),
		TimeSynced:                bool(sw.SyncInternetTime),
	}
	for _, wan := range w.WanInfo {
		s.WAN = append(s.WAN, SystemWANInfo{
			ConnectStatus: wan.AdvConnectStatus, ConnectType: wan.AdvConnectType, UptimeSeconds: wan.AdvConnectTime,
			IP: wan.AdvIp, Mask: wan.AdvMask, Gateway: wan.AdvGateway, DNS1: wan.AdvDns1, DNS2: wan.AdvDns2, MAC: wan.AdvMac,
			UpKBps: wan.WanUploadSpeed, DownKBps: wan.WanDownloadSpeed,
		})
	}
	return s, nil
}

// SystemWANConnectStatusText labels a SystemWANInfo.ConnectStatus value, or
// "" if it is out of range.
func SystemWANConnectStatusText(status int) string {
	if status < 0 || status >= len(systemWANConnectStatuses) {
		return ""
	}
	return systemWANConnectStatuses[status]
}

// SystemWANConnectTypeText labels a SystemWANInfo.ConnectType numeral, or ""
// if it does not parse or is out of range.
func SystemWANConnectTypeText(connectType string) string {
	if connectType == "" {
		return ""
	}
	i := 0
	for _, r := range connectType {
		if r < '0' || r > '9' {
			return ""
		}
		i = i*10 + int(r-'0')
	}
	if i >= len(systemWiFiConnectTypes) {
		return ""
	}
	return systemWiFiConnectTypes[i]
}
