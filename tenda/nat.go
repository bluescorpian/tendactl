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

// Protocol is a port-forwarding rule's protocol. It implements pflag.Value.
type Protocol int

const (
	ProtoBoth Protocol = iota // wire "0"
	ProtoTCP                  // wire "1"
	ProtoUDP                  // wire "2"
)

// String is the UI's label, used in tables.
func (p Protocol) String() string {
	switch p {
	case ProtoTCP:
		return "TCP"
	case ProtoUDP:
		return "UDP"
	}
	return "TCP&UDP"
}

// MarshalText is the JSON output spelling.
func (p Protocol) MarshalText() ([]byte, error) {
	switch p {
	case ProtoTCP:
		return []byte("tcp"), nil
	case ProtoUDP:
		return []byte("udp"), nil
	}
	return []byte("both"), nil
}

func (p *Protocol) Set(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "both", "0", "tcp&udp", "tcp+udp":
		*p = ProtoBoth
	case "tcp", "1":
		*p = ProtoTCP
	case "udp", "2":
		*p = ProtoUDP
	default:
		return fmt.Errorf("invalid protocol %q: want both, tcp or udp", s)
	}
	return nil
}

func (Protocol) Type() string { return "both|tcp|udp" }

// PortForward is one "Virtual Server" rule. InPort is the UI's "LAN Port",
// OutPort its "WAN Port"; OutPort is unique across rules.
type PortForward struct {
	IP       string   `json:"ip"`
	InPort   int      `json:"inPort"`
	OutPort  int      `json:"outPort"`
	Protocol Protocol `json:"protocol"`
}

// NATConfig is the "Virtual Server" page (GetVirtualServerCfg).
type NATConfig struct {
	LANIP   string        `json:"lanIp"`
	LANMask string        `json:"lanMask"`
	Rules   []PortForward `json:"rules"`
}

type natWire struct {
	LanIp       string `json:"lanIp"`
	LanMask     string `json:"lanMask"`
	VirtualList []struct {
		Ip       string `json:"ip"`
		InPort   string `json:"inPort"`
		OutPort  string `json:"outPort"`
		Protocol string `json:"protocol"`
	} `json:"virtualList"`
}

// natMaxRules is the UI's client-side limit.
const natMaxRules = 16

var (
	ErrNATNoRule        = errors.New("no port forwarding rule with that WAN port")
	ErrNATFull          = fmt.Errorf("the router allows at most %d port forwarding rules", natMaxRules)
	ErrNATDuplicatePort = errors.New("a port forwarding rule already uses that WAN port")
)

// NAT reads GetVirtualServerCfg.
func (c *Client) NAT(ctx context.Context) (NATConfig, error) {
	var w natWire
	if err := c.get(ctx, "GetVirtualServerCfg", nil, &w); err != nil {
		return NATConfig{}, err
	}
	cfg := NATConfig{LANIP: w.LanIp, LANMask: w.LanMask, Rules: make([]PortForward, 0, len(w.VirtualList))}
	for i, v := range w.VirtualList {
		in, err1 := strconv.Atoi(v.InPort)
		out, err2 := strconv.Atoi(v.OutPort)
		var p Protocol
		err3 := p.Set(v.Protocol)
		if err := errors.Join(err1, err2, err3); err != nil {
			return NATConfig{}, &DecodeError{Endpoint: "GetVirtualServerCfg", Err: fmt.Errorf("virtualList[%d]: %w", i, err)}
		}
		cfg.Rules = append(cfg.Rules, PortForward{IP: v.Ip, InPort: in, OutPort: out, Protocol: p})
	}
	return cfg, nil
}

// SetNATRules replaces the whole rule list (SetVirtualServerCfg).
func (c *Client) SetNATRules(ctx context.Context, rules []PortForward) error {
	rows := make([][]string, len(rules))
	for i, r := range rules {
		rows[i] = []string{r.IP, strconv.Itoa(r.InPort), strconv.Itoa(r.OutPort), strconv.Itoa(int(r.Protocol))}
	}
	list, err := TildeComma.Encode(rows)
	if err != nil {
		return err
	}
	return c.set(ctx, "SetVirtualServerCfg", url.Values{"list": {list}})
}

// AddNATRule appends r after the UI's checks: at most 16 rules, a unique WAN
// port, ports in 1..65535, and a target inside the LAN subnet that is not
// the router itself. Nothing is posted when a check fails.
func (c *Client) AddNATRule(ctx context.Context, r PortForward) error {
	cfg, err := c.NAT(ctx)
	if err != nil {
		return err
	}
	if err := natValidate(cfg, r); err != nil {
		return err
	}
	return c.SetNATRules(ctx, append(cfg.Rules, r))
}

func natValidate(cfg NATConfig, r PortForward) error {
	for _, p := range []int{r.InPort, r.OutPort} {
		if p < 1 || p > 65535 {
			return fmt.Errorf("port %d out of range 1-65535", p)
		}
	}
	ip := net.ParseIP(r.IP).To4()
	if ip == nil {
		return fmt.Errorf("invalid IPv4 address %q", r.IP)
	}
	if lan := net.ParseIP(cfg.LANIP).To4(); lan != nil {
		if ip.Equal(lan) {
			return fmt.Errorf("%s is the router's own LAN IP", r.IP)
		}
		if mask := net.ParseIP(cfg.LANMask).To4(); mask != nil {
			m := net.IPMask(mask)
			if !ip.Mask(m).Equal(lan.Mask(m)) {
				return fmt.Errorf("%s is outside the LAN subnet %s/%s", r.IP, cfg.LANIP, cfg.LANMask)
			}
		}
	}
	if len(cfg.Rules) >= natMaxRules {
		return ErrNATFull
	}
	for _, e := range cfg.Rules {
		if e.OutPort == r.OutPort {
			return fmt.Errorf("%w (%d)", ErrNATDuplicatePort, r.OutPort)
		}
	}
	return nil
}

// RemoveNATRule deletes the rule with WAN port outPort. It returns
// ErrNATNoRule, having posted nothing, when there is none.
func (c *Client) RemoveNATRule(ctx context.Context, outPort int) error {
	cfg, err := c.NAT(ctx)
	if err != nil {
		return err
	}
	kept := make([]PortForward, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		if r.OutPort != outPort {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(cfg.Rules) {
		return fmt.Errorf("%w (%d)", ErrNATNoRule, outPort)
	}
	return c.SetNATRules(ctx, kept)
}
