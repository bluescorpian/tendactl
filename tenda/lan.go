package tenda

import (
	"context"
	"net/url"
)

// LAN is "LAN Settings" (System Settings): AdvGetLanIp / AdvSetLanip.
type LAN struct {
	LANIP       string `json:"lanIp"`
	LANMask     string `json:"lanMask"`
	DHCPEnabled bool   `json:"dhcpEnabled"`
	DHCPStart   string `json:"dhcpStart"`
	DHCPEnd     string `json:"dhcpEnd"`
	LeaseTime   string `json:"leaseTime"` // seconds: 604800, 172800, 86400, 21600 or 3600
	DNSAuto     bool   `json:"dnsAuto"`
	DNS1        string `json:"dns1"`
	DNS2        string `json:"dns2"`
}

// lanWire has every key of AdvGetLanIp.json. Most of the extra fields
// (WanIp, GuestIp, PptpSvrIp, ...) are present only for the UI's own
// same-segment validation and are not surfaced in LAN.
type lanWire struct {
	LanIp       string `json:"lanIp"`
	LanMask     string `json:"lanMask"`
	Ip          string `json:"ip"`
	Mask        string `json:"mask"`
	StartIp     string `json:"startIp"`
	EndIp       string `json:"endIp"`
	LeaseTime   string `json:"leaseTime"`
	DhcpEn      Flag   `json:"dhcpEn"`
	LanDnsAuto  Flag   `json:"lanDnsAuto"`
	LanDns1     string `json:"lanDns1"`
	LanDns2     string `json:"lanDns2"`
	GuestIp     string `json:"guestIp"`
	GuestMask   string `json:"guestMask"`
	ServerIp    string `json:"serverIp"`
	Vlan2Ip     string `json:"vlan2Ip"`
	Vlan2Mask   string `json:"vlan2Mask"`
	WanIp       string `json:"wanIp"`
	WanMask     string `json:"wanMask"`
	PptpSvrIp   string `json:"pptpSvrIp"`
	PptpSvrMask string `json:"pptpSvrMask"`
	VpnCliIp    string `json:"vpnCliIp"`
	WlMode      string `json:"wl_mode"`
	RemoteIp    string `json:"remoteIp"`
}

// LAN reads AdvGetLanIp.
func (c *Client) LAN(ctx context.Context) (LAN, error) {
	var w lanWire
	if err := c.get(ctx, "AdvGetLanIp", nil, &w); err != nil {
		return LAN{}, err
	}
	return LAN{
		LANIP:       w.LanIp,
		LANMask:     w.LanMask,
		DHCPEnabled: bool(w.DhcpEn),
		DHCPStart:   w.StartIp,
		DHCPEnd:     w.EndIp,
		LeaseTime:   w.LeaseTime,
		DNSAuto:     bool(w.LanDnsAuto),
		DNS1:        w.LanDns1,
		DNS2:        w.LanDns2,
	}, nil
}

// SetLAN sends the full UI form (AdvSetLanip). It is on the hazard list
// unconditionally: changing the LAN address moves the router's own
// management address, and shrinking the DHCP range can strand leased
// clients, so every call needs --yes.
func (c *Client) SetLAN(ctx context.Context, l LAN) error {
	form := url.Values{
		"lanIp":      {l.LANIP},
		"lanMask":    {l.LANMask},
		"dhcpEn":     {flag(l.DHCPEnabled)},
		"startIp":    {l.DHCPStart},
		"endIp":      {l.DHCPEnd},
		"leaseTime":  {l.LeaseTime},
		"lanDnsAuto": {flag(l.DNSAuto)},
		"lanDns1":    {l.DNS1},
		"lanDns2":    {l.DNS2},
	}
	return c.set(ctx, "AdvSetLanip", form)
}
