package tenda

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// OnlineList is "Manage Device" (getOnlineList): the admin's own device, the
// MAC filter mode and every client the router has seen.
type OnlineList struct {
	Host          OnlineHost     `json:"host"`
	MACFilterMode string         `json:"macFilterMode"` // "black" or "white"
	Clients       []OnlineClient `json:"clients"`
}

// OnlineHost is the device the request came from ("Local Host" in the UI).
type OnlineHost struct {
	IP       string `json:"ip"`
	Name     string `json:"name"`
	MAC      string `json:"mac"`
	Wireless bool   `json:"wireless"`
}

// OnlineClient is one attached device.
type OnlineClient struct {
	MAC      string `json:"mac"`
	IP       string `json:"ip"`
	Name     string `json:"name"`
	UpKBps   string `json:"upKBps"`
	DownKBps string `json:"downKBps"`
	LinkType string `json:"linkType"`
	Line     string `json:"line"` // "wired", "2.4" or "5"
	Guest    bool   `json:"guest"`
	Blocked  bool   `json:"blocked"`
}

type onlineHeadWire struct {
	BlackNum          int    `json:"blackNum"` // unreliable (12492040 on an empty list); dropped
	MacFilterType     string `json:"macFilterType"`
	LocalhostIP       string `json:"localhostIP"`
	IsWirelessConnect Flag   `json:"isWirelessConnect"`
	LocalhostName     string `json:"localhostName"`
	LocalhostMac      string `json:"localhostMac"`
}

type onlineClientWire struct {
	DeviceId      string `json:"deviceId"`
	Ip            string `json:"ip"`
	DevName       string `json:"devName"`
	Line          string `json:"line"`
	UploadSpeed   string `json:"uploadSpeed"`
	DownloadSpeed string `json:"downloadSpeed"`
	LinkType      string `json:"linkType"`
	Black         Flag   `json:"black"`
	IsGuestClient Flag   `json:"isGuestClient"`
}

// OnlineList reads getOnlineList, a JSON array whose first element describes
// the requesting host and whose rest are clients.
func (c *Client) OnlineList(ctx context.Context) (OnlineList, error) {
	var raw []json.RawMessage
	if err := c.get(ctx, "getOnlineList", nil, &raw); err != nil {
		return OnlineList{}, err
	}
	if len(raw) == 0 {
		return OnlineList{}, &DecodeError{Endpoint: "getOnlineList", Body: []byte("[]"), Err: errors.New("empty array")}
	}
	var head onlineHeadWire
	if err := c.decodeJSON("getOnlineList", raw[0], &head); err != nil {
		return OnlineList{}, err
	}
	l := OnlineList{
		Host:          OnlineHost{IP: head.LocalhostIP, Name: head.LocalhostName, MAC: head.LocalhostMac, Wireless: bool(head.IsWirelessConnect)},
		MACFilterMode: head.MacFilterType,
		Clients:       make([]OnlineClient, 0, len(raw)-1),
	}
	for _, r := range raw[1:] {
		var w onlineClientWire
		if err := c.decodeJSON("getOnlineList", r, &w); err != nil {
			return OnlineList{}, err
		}
		l.Clients = append(l.Clients, OnlineClient{
			MAC: w.DeviceId, IP: w.Ip, Name: w.DevName,
			UpKBps: w.UploadSpeed, DownKBps: w.DownloadSpeed,
			LinkType: w.LinkType, Line: clientLine(w.Line),
			Guest: bool(w.IsGuestClient), Blocked: bool(w.Black),
		})
	}
	return l, nil
}

func clientLine(s string) string {
	switch s {
	case "0":
		return "wired"
	case "1":
		return Band24.String()
	case "2":
		return Band5.String()
	}
	return s
}

// BlockedClient is one entry in the "Manage Device" quick blacklist
// (getBlackRuleList), a different list from macfilter's block/allow table.
type BlockedClient struct {
	MAC  string `json:"mac"`
	Name string `json:"name"`
}

type blockedClientWire struct {
	DeviceId string `json:"deviceId"`
	DevName  string `json:"devName"`
}

var (
	// ErrClientIsLocalhost is BlockClient's refusal to blacklist the
	// browsing admin's own device; there is no doc'd errCode for this,
	// since the UI's "Add" button is simply hidden for that row.
	ErrClientIsLocalhost = errors.New("refusing to block the local host device; use the api command to override")
	// ErrClientBlocklistFull is setBlackRule errCode 1 (max 30 entries).
	ErrClientBlocklistFull = errors.New("the quick blacklist is full (max 30 entries)")
	// ErrClientNameTooLong is the UI's 20-character device name limit.
	ErrClientNameTooLong = errors.New("device name longer than 20 characters")
)

// RenameClient sets a device's name (SetOnlineDevName), identical across the
// four pages that call it.
func (c *Client) RenameClient(ctx context.Context, mac, name string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	if len([]rune(name)) > 20 {
		return ErrClientNameTooLong
	}
	return c.set(ctx, "SetOnlineDevName", url.Values{"mac": {strings.ToLower(m)}, "devName": {name}})
}

// BlockClient adds mac to the quick blacklist (setBlackRule), immediately
// cutting off its network access. It refuses the browsing session's own
// device outright; api can still be used to force it.
func (c *Client) BlockClient(ctx context.Context, mac string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	l, err := c.OnlineList(ctx)
	if err != nil {
		return err
	}
	if EqualMAC(m, l.Host.MAC) {
		return ErrClientIsLocalhost
	}
	err = c.set(ctx, "setBlackRule", url.Values{"mac": {strings.ToLower(m)}})
	return mapCode(err, 1, ErrClientBlocklistFull)
}

// UnblockClient removes mac from the quick blacklist (delBlackRule),
// restoring its network access.
func (c *Client) UnblockClient(ctx context.Context, mac string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	return c.set(ctx, "delBlackRule", url.Values{"mac": {strings.ToLower(m)}})
}

// BlockedClients reads the quick blacklist (getBlackRuleList).
func (c *Client) BlockedClients(ctx context.Context) ([]BlockedClient, error) {
	var raw []blockedClientWire
	if err := c.get(ctx, "getBlackRuleList", nil, &raw); err != nil {
		return nil, err
	}
	out := make([]BlockedClient, 0, len(raw))
	for _, w := range raw {
		out = append(out, BlockedClient{MAC: w.DeviceId, Name: w.DevName})
	}
	return out, nil
}
