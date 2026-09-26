package tenda

import (
	"context"
	"encoding/json"
	"errors"
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
