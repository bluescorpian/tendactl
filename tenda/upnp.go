package tenda

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
)

// UPnP is the "UPnP" page (Advanced Settings): GetUpnpCfg / SetUpnpCfg.
type UPnP struct {
	Enabled  bool          `json:"enabled"`
	Mappings []UPnPMapping `json:"mappings"` // read-only; live, UPnP-negotiated port mappings
}

// UPnPMapping is one active, LAN-device-negotiated port mapping.
type UPnPMapping struct {
	RemoteHost string `json:"remoteHost"`
	OutPort    string `json:"outPort"`
	Host       string `json:"host"`
	InPort     string `json:"inPort"`
	Protocol   string `json:"protocol"`
}

type upnpHeadWire struct {
	UpnpEn Flag `json:"upnpEn"`
}

type upnpMappingWire struct {
	RemoteHost string `json:"remoteHost"`
	OutPort    string `json:"outPort"`
	Host       string `json:"host"`
	InPort     string `json:"inPort"`
	Protocol   string `json:"protocol"`
}

// UPnP reads GetUpnpCfg, a JSON array whose first element is the enable
// state and whose rest, if any, are live mappings.
func (c *Client) UPnP(ctx context.Context) (UPnP, error) {
	var raw []json.RawMessage
	if err := c.get(ctx, "GetUpnpCfg", nil, &raw); err != nil {
		return UPnP{}, err
	}
	if len(raw) == 0 {
		return UPnP{}, &DecodeError{Endpoint: "GetUpnpCfg", Body: []byte("[]"), Err: errors.New("empty array")}
	}
	var head upnpHeadWire
	if err := c.decodeJSON("GetUpnpCfg", raw[0], &head); err != nil {
		return UPnP{}, err
	}
	u := UPnP{Enabled: bool(head.UpnpEn), Mappings: make([]UPnPMapping, 0, len(raw)-1)}
	for _, r := range raw[1:] {
		var w upnpMappingWire
		if err := c.decodeJSON("GetUpnpCfg", r, &w); err != nil {
			return UPnP{}, err
		}
		u.Mappings = append(u.Mappings, UPnPMapping(w))
	}
	return u, nil
}

// SetUPnP enables or disables UPnP. Mappings are read-only and not sent.
func (c *Client) SetUPnP(ctx context.Context, u UPnP) error {
	return c.set(ctx, "SetUpnpCfg", url.Values{"upnpEn": {flag(u.Enabled)}})
}
