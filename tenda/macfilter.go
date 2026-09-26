package tenda

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// MACFilter is "Filter MAC Address" (Advanced Settings): getMacFilterCfg /
// setMacFilterCfg. Devices is whichever list Mode currently enforces; the
// other mode's list is untouched by Set/Add/Remove, matching the doc's
// "only the selected radio's table is serialized".
type MACFilter struct {
	Mode    string           `json:"mode"` // "black" or "white"
	Devices []MACFilterEntry `json:"devices"`
}

// MACFilterEntry is one allow/deny row.
type MACFilterEntry struct {
	MAC  string `json:"mac"`
	Name string `json:"name"`
}

type macFilterEntryWire struct {
	DevName string `json:"devName"`
	DevMac  string `json:"devMac"`
}

type macFilterWire struct {
	LocalhostIP   string               `json:"localhostIP"`
	LocalhostName string               `json:"localhostName"`
	LocalhostMac  string               `json:"localhostMac"`
	MacFilterType string               `json:"macFilterType"`
	BlackList     []macFilterEntryWire `json:"blackList"`
	WhiteList     []macFilterEntryWire `json:"whiteList"`
	OnlineList    []macFilterEntryWire `json:"onlineList"` // helper list for the UI's "add all online"; not exposed
}

// macFilterMax is the UI's client-side limit, per list.
const macFilterMax = 30

var (
	ErrMACFilterFull      = fmt.Errorf("at most %d entries are allowed per list", macFilterMax)
	ErrMACFilterDuplicate = errors.New("that MAC address is already in the list")
	ErrMACFilterNoEntry   = errors.New("no entry for that MAC address")
)

func (c *Client) macFilterWire(ctx context.Context) (macFilterWire, error) {
	var w macFilterWire
	err := c.get(ctx, "getMacFilterCfg", nil, &w)
	return w, err
}

func macFilterEntries(list []macFilterEntryWire) []MACFilterEntry {
	out := make([]MACFilterEntry, 0, len(list))
	for _, e := range list {
		out = append(out, MACFilterEntry{MAC: e.DevMac, Name: e.DevName})
	}
	return out
}

func macFilterActiveList(w macFilterWire) []macFilterEntryWire {
	if w.MacFilterType == "white" {
		return w.WhiteList
	}
	return w.BlackList
}

// MACFilter reads getMacFilterCfg.
func (c *Client) MACFilter(ctx context.Context) (MACFilter, error) {
	w, err := c.macFilterWire(ctx)
	if err != nil {
		return MACFilter{}, err
	}
	return MACFilter{Mode: w.MacFilterType, Devices: macFilterEntries(macFilterActiveList(w))}, nil
}

// setMACFilter posts mode's full device list (setMacFilterCfg). Switching to
// "white" is a hazard: every device not on the list loses access.
func (c *Client) setMACFilter(ctx context.Context, mode string, devices []MACFilterEntry) error {
	rows := make([][]string, len(devices))
	for i, d := range devices {
		rows[i] = []string{d.Name, d.MAC}
	}
	list, err := LineCR.Encode(rows)
	if err != nil {
		return err
	}
	return c.set(ctx, "setMacFilterCfg", url.Values{"macFilterType": {mode}, "deviceList": {list}})
}

// SetMACFilterMode switches which list is enforced, resending that list's
// current entries unchanged (the other list is left as-is on the router).
func (c *Client) SetMACFilterMode(ctx context.Context, mode string) error {
	if mode != "black" && mode != "white" {
		return fmt.Errorf("invalid mode %q: want black or white", mode)
	}
	w, err := c.macFilterWire(ctx)
	if err != nil {
		return err
	}
	list := w.BlackList
	if mode == "white" {
		list = w.WhiteList
	}
	return c.setMACFilter(ctx, mode, macFilterEntries(list))
}

// AddMACFilterEntry adds mac to the currently-active list, after the UI's
// checks: at most 30 entries, no duplicate MAC. Nothing is posted when a
// check fails.
func (c *Client) AddMACFilterEntry(ctx context.Context, mac, name string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	f, err := c.MACFilter(ctx)
	if err != nil {
		return err
	}
	for _, e := range f.Devices {
		if EqualMAC(e.MAC, m) {
			return fmt.Errorf("%w (%s)", ErrMACFilterDuplicate, m)
		}
	}
	if len(f.Devices) >= macFilterMax {
		return ErrMACFilterFull
	}
	devices := append(f.Devices, MACFilterEntry{MAC: m, Name: name})
	return c.setMACFilter(ctx, f.Mode, devices)
}

// RemoveMACFilterEntry deletes mac from the currently-active list. It
// returns ErrMACFilterNoEntry, having posted nothing, when there is none.
func (c *Client) RemoveMACFilterEntry(ctx context.Context, mac string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	f, err := c.MACFilter(ctx)
	if err != nil {
		return err
	}
	kept := make([]MACFilterEntry, 0, len(f.Devices))
	for _, e := range f.Devices {
		if !EqualMAC(e.MAC, m) {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(f.Devices) {
		return fmt.Errorf("%w (%s)", ErrMACFilterNoEntry, m)
	}
	return c.setMACFilter(ctx, f.Mode, kept)
}
