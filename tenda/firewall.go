package tenda

import (
	"context"
	"fmt"
	"net/url"
)

// Firewall is the "Firewall" page (Advanced Settings): GetFirewallCfg /
// SetFirewallCfg.
type Firewall struct {
	ICMPFloodDefense bool `json:"icmpFloodDefense"`
	TCPFloodDefense  bool `json:"tcpFloodDefense"`
	UDPFloodDefense  bool `json:"udpFloodDefense"`
	IgnoreWANPing    bool `json:"ignoreWanPing"`
}

type firewallWire struct {
	FirewallEn string `json:"firewallEn"` // 4 chars: ICMP, TCP, UDP flood defense, then ignore-WAN-ping
}

// Firewall reads GetFirewallCfg.
func (c *Client) Firewall(ctx context.Context) (Firewall, error) {
	var w firewallWire
	if err := c.get(ctx, "GetFirewallCfg", nil, &w); err != nil {
		return Firewall{}, err
	}
	if len(w.FirewallEn) != 4 {
		return Firewall{}, &DecodeError{Endpoint: "GetFirewallCfg", Body: []byte(w.FirewallEn), Err: fmt.Errorf("firewallEn: want 4 characters, got %q", w.FirewallEn)}
	}
	bits := [4]bool{}
	for i, r := range w.FirewallEn {
		switch r {
		case '1':
			bits[i] = true
		case '0':
			bits[i] = false
		default:
			return Firewall{}, &DecodeError{Endpoint: "GetFirewallCfg", Body: []byte(w.FirewallEn), Err: fmt.Errorf("firewallEn: unexpected character %q", r)}
		}
	}
	return Firewall{ICMPFloodDefense: bits[0], TCPFloodDefense: bits[1], UDPFloodDefense: bits[2], IgnoreWANPing: bits[3]}, nil
}

// SetFirewall sends the full UI form.
func (c *Client) SetFirewall(ctx context.Context, f Firewall) error {
	bit := func(b bool) byte {
		if b {
			return '1'
		}
		return '0'
	}
	s := string([]byte{bit(f.ICMPFloodDefense), bit(f.TCPFloodDefense), bit(f.UDPFloodDefense), bit(f.IgnoreWANPing)})
	return c.set(ctx, "SetFirewallCfg", url.Values{"firewallEn": {s}})
}
