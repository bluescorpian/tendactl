package tenda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// Bandwidth is "Bandwidth Control" (Advanced Settings): GetNetControlList /
// SetNetControlList. Devices is always every device the router knows; there
// is no add/remove, only editing a device's limits.
type Bandwidth struct {
	Enabled bool              `json:"enabled"`
	Devices []BandwidthDevice `json:"devices"`
}

// BandwidthDevice is one device's live throughput and configured caps.
// LimitUpMbps/LimitDownMbps are 0 for unlimited.
type BandwidthDevice struct {
	MAC           string  `json:"mac"`
	IP            string  `json:"ip"`
	Name          string  `json:"name"`
	UpKBps        string  `json:"upKBps"`
	DownKBps      string  `json:"downKBps"`
	LimitUpMbps   float64 `json:"limitUpMbps"`
	LimitDownMbps float64 `json:"limitDownMbps"`
	Limited       bool    `json:"limited"` // isControled
	Offline       bool    `json:"offline"`
}

type bandwidthHeadWire struct {
	NetControlEn string `json:"netControlEn"`
}

type bandwidthDeviceWire struct {
	UpSpeed     string `json:"upSpeed"`
	DownSpeed   string `json:"downSpeed"`
	DevType     string `json:"devType"`
	HostName    string `json:"hostName"`
	Ip          string `json:"ip"`
	Mac         string `json:"mac"`
	LimitUp     string `json:"limitUp"`
	LimitDown   string `json:"limitDown"`
	IsControled Flag   `json:"isControled"`
	Offline     Flag   `json:"offline"`
	IsSet       Flag   `json:"isSet"`
}

// bandwidthKBpsPerMbps is the wire unit conversion (docs: UI names,
// Bandwidth Control).
const bandwidthKBpsPerMbps = 128

var ErrBandwidthNoDevice = errors.New("no known device with that MAC address")

// Bandwidth reads GetNetControlList, a JSON array whose first element is the
// global enable flag and whose rest are devices.
func (c *Client) Bandwidth(ctx context.Context) (Bandwidth, error) {
	var raw []json.RawMessage
	if err := c.get(ctx, "GetNetControlList", nil, &raw); err != nil {
		return Bandwidth{}, err
	}
	if len(raw) == 0 {
		return Bandwidth{}, &DecodeError{Endpoint: "GetNetControlList", Body: []byte("[]"), Err: errors.New("empty array")}
	}
	var head bandwidthHeadWire
	if err := c.decodeJSON("GetNetControlList", raw[0], &head); err != nil {
		return Bandwidth{}, err
	}
	b := Bandwidth{Enabled: head.NetControlEn == "1", Devices: make([]BandwidthDevice, 0, len(raw)-1)}
	for _, r := range raw[1:] {
		var w bandwidthDeviceWire
		if err := c.decodeJSON("GetNetControlList", r, &w); err != nil {
			return Bandwidth{}, err
		}
		up, err1 := strconv.Atoi(w.LimitUp)
		down, err2 := strconv.Atoi(w.LimitDown)
		if err := errors.Join(err1, err2); err != nil {
			return Bandwidth{}, &DecodeError{Endpoint: "GetNetControlList", Err: err}
		}
		b.Devices = append(b.Devices, BandwidthDevice{
			MAC: w.Mac, IP: w.Ip, Name: w.HostName,
			UpKBps: w.UpSpeed, DownKBps: w.DownSpeed,
			LimitUpMbps: float64(up) / bandwidthKBpsPerMbps, LimitDownMbps: float64(down) / bandwidthKBpsPerMbps,
			Limited: bool(w.IsControled), Offline: bool(w.Offline),
		})
	}
	return b, nil
}

// setBandwidth posts the full device list. getSubmitData() never includes
// netControlEn in the outgoing body for any SetNetControlList call: the
// field, and the conditional around it, are commented out in
// js/net_control.js, and the on-page #netControlEn control is a purely
// local UI-state toggle (it only starts/stops the page's 5s polling
// interval) whose value is never read into the submitted string, for any
// of enable, disable or a plain per-device limit edit.
func (c *Client) setBandwidth(ctx context.Context, devices []BandwidthDevice) error {
	rows := make([][]string, len(devices))
	for i, d := range devices {
		rows[i] = []string{d.Name, d.MAC, strconv.Itoa(int(d.LimitUpMbps * bandwidthKBpsPerMbps)), strconv.Itoa(int(d.LimitDownMbps * bandwidthKBpsPerMbps))}
	}
	list, err := LineCR.Encode(rows)
	if err != nil {
		return err
	}
	return c.set(ctx, "SetNetControlList", url.Values{"list": {list}})
}

// SetBandwidthEnabled resends every device's current limits unchanged. It
// cannot actually change the global Bandwidth Control switch: the router's
// own web UI has no way to either, in this firmware build (see
// setBandwidth); the field is read-only via GetNetControlList.
func (c *Client) SetBandwidthEnabled(ctx context.Context, enabled bool) error {
	b, err := c.Bandwidth(ctx)
	if err != nil {
		return err
	}
	return c.setBandwidth(ctx, b.Devices)
}

// SetBandwidthLimit sets one device's upload and/or download cap in Mbps (0
// = unlimited). A nil pointer leaves that direction unchanged.
func (c *Client) SetBandwidthLimit(ctx context.Context, mac string, upMbps, downMbps *float64) error {
	b, err := c.Bandwidth(ctx)
	if err != nil {
		return err
	}
	idx, err := bandwidthFind(b, mac)
	if err != nil {
		return err
	}
	if upMbps != nil {
		b.Devices[idx].LimitUpMbps = *upMbps
	}
	if downMbps != nil {
		b.Devices[idx].LimitDownMbps = *downMbps
	}
	return c.setBandwidth(ctx, b.Devices)
}

// RemoveBandwidthLimit resets a device to unlimited in both directions.
func (c *Client) RemoveBandwidthLimit(ctx context.Context, mac string) error {
	zero := 0.0
	return c.SetBandwidthLimit(ctx, mac, &zero, &zero)
}

func bandwidthFind(b Bandwidth, mac string) (int, error) {
	for i, d := range b.Devices {
		if EqualMAC(d.MAC, mac) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%w (%s)", ErrBandwidthNoDevice, mac)
}
