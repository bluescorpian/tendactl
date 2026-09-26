package tenda

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// DHCP is "DHCP Reservation" (System Settings tab: GetIpMacBind /
// SetIpMacBind). Clients are leased devices that are not (yet) reserved;
// Bindings are the reserved IP-MAC pairs.
type DHCP struct {
	LANIP    string        `json:"lanIp"`
	LANMask  string        `json:"lanMask"`
	Clients  []DHCPClient  `json:"clients"`
	Bindings []DHCPBinding `json:"bindings"`

	dhttpIP string // reserved IP the UI forbids binding to; used by AddDHCPBinding
}

// DHCPClient is one currently-leased device that has no reservation.
type DHCPClient struct {
	MAC    string `json:"mac"`
	IP     string `json:"ip"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

// DHCPBinding is one reserved IP-MAC pair.
type DHCPBinding struct {
	MAC    string `json:"mac"`
	IP     string `json:"ip"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

// dhcpEntryWire is the shape shared by dhcpClientList and bindList rows.
type dhcpEntryWire struct {
	Ipaddr  string `json:"ipaddr"`
	Macaddr string `json:"macaddr"`
	Devname string `json:"devname"`
	Status  string `json:"status"` // "1" online, "0" offline
}

type dhcpWire struct {
	LanIp          string          `json:"lanIp"`
	LanMask        string          `json:"lanMask"`
	DhttpIP        string          `json:"dhttpIP"`
	DhcpClientList []dhcpEntryWire `json:"dhcpClientList"`
	BindList       []dhcpEntryWire `json:"bindList"`
}

// dhcpMaxBindings is the UI's client-side limit (maxBindNum).
const dhcpMaxBindings = 32

var (
	ErrDHCPFull        = fmt.Errorf("the router allows at most %d DHCP reservations", dhcpMaxBindings)
	ErrDHCPNoBinding   = errors.New("no DHCP reservation for that MAC address")
	ErrDHCPDuplicateIP = errors.New("a DHCP reservation already uses that IP address")
)

// DHCP reads GetIpMacBind.
func (c *Client) DHCP(ctx context.Context) (DHCP, error) {
	var w dhcpWire
	if err := c.get(ctx, "GetIpMacBind", nil, &w); err != nil {
		return DHCP{}, err
	}
	d := DHCP{
		LANIP: w.LanIp, LANMask: w.LanMask, dhttpIP: w.DhttpIP,
		Clients:  make([]DHCPClient, 0, len(w.DhcpClientList)),
		Bindings: make([]DHCPBinding, 0, len(w.BindList)),
	}
	for _, e := range w.DhcpClientList {
		d.Clients = append(d.Clients, DHCPClient{MAC: e.Macaddr, IP: e.Ipaddr, Name: e.Devname, Online: e.Status == "1"})
	}
	for _, e := range w.BindList {
		d.Bindings = append(d.Bindings, DHCPBinding{MAC: e.Macaddr, IP: e.Ipaddr, Name: e.Devname, Online: e.Status == "1"})
	}
	return d, nil
}

// SetDHCPBindings replaces the whole reservation list (SetIpMacBind). The
// UI's callback for this endpoint always shows the same fixed success
// message and never reads errCode -- the real errCode handling is commented
// out (fw/js/ip_mac_bind.js) -- so this uses post, not set, and only
// transport/session errors are reported.
func (c *Client) SetDHCPBindings(ctx context.Context, bindings []DHCPBinding) error {
	rows := make([][]string, len(bindings))
	for i, b := range bindings {
		rows[i] = []string{b.Name, b.MAC, b.IP}
	}
	list, err := LineCR.Encode(rows)
	if err != nil {
		return err
	}
	return c.post(ctx, "SetIpMacBind", url.Values{"bindnum": {strconv.Itoa(len(bindings))}, "list": {list}})
}

// AddDHCPBinding reserves ip for mac, after the UI's checks: at most 32
// reservations, ip not the router's own LAN IP or its reserved dhttpIP, and
// ip not already used by a different MAC. Re-adding an existing MAC
// overwrites its row rather than erroring, per the doc. Nothing is posted
// when a check fails.
func (c *Client) AddDHCPBinding(ctx context.Context, b DHCPBinding) error {
	m, err := ParseMAC(b.MAC)
	if err != nil {
		return err
	}
	b.MAC = strings.ToLower(m)
	d, err := c.DHCP(ctx)
	if err != nil {
		return err
	}
	if err := dhcpValidate(d, b); err != nil {
		return err
	}
	bindings := make([]DHCPBinding, 0, len(d.Bindings)+1)
	replaced := false
	for _, e := range d.Bindings {
		if EqualMAC(e.MAC, b.MAC) {
			bindings = append(bindings, b)
			replaced = true
			continue
		}
		bindings = append(bindings, e)
	}
	if !replaced {
		if len(d.Bindings) >= dhcpMaxBindings {
			return ErrDHCPFull
		}
		bindings = append(bindings, b)
	}
	return c.SetDHCPBindings(ctx, bindings)
}

func dhcpValidate(d DHCP, b DHCPBinding) error {
	ip := net.ParseIP(b.IP).To4()
	if ip == nil {
		return fmt.Errorf("invalid IPv4 address %q", b.IP)
	}
	if b.IP == d.LANIP {
		return fmt.Errorf("%s is the router's own LAN IP", b.IP)
	}
	if d.dhttpIP != "" && b.IP == d.dhttpIP {
		return fmt.Errorf("%s is reserved by the router", b.IP)
	}
	for _, e := range d.Bindings {
		if !EqualMAC(e.MAC, b.MAC) && e.IP == b.IP {
			return fmt.Errorf("%w (%s)", ErrDHCPDuplicateIP, b.IP)
		}
	}
	return nil
}

// RemoveDHCPBinding deletes the reservation for mac. It returns
// ErrDHCPNoBinding, having posted nothing, when there is none.
func (c *Client) RemoveDHCPBinding(ctx context.Context, mac string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	d, err := c.DHCP(ctx)
	if err != nil {
		return err
	}
	kept := make([]DHCPBinding, 0, len(d.Bindings))
	for _, e := range d.Bindings {
		if !EqualMAC(e.MAC, m) {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(d.Bindings) {
		return fmt.Errorf("%w (%s)", ErrDHCPNoBinding, m)
	}
	return c.SetDHCPBindings(ctx, kept)
}
