package tenda

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

const natFixtureList = "192.168.0.5,25565,25565,1~192.168.0.5,42069,42069,1~192.168.0.5,24454,24454,2~192.168.0.5,19284,19284,2~192.168.0.5,8100,8100,1~192.168.0.5,80,80,1~192.168.0.5,443,443,1"

func TestNAT(t *testing.T) {
	r := tendatest.New(t)
	cfg, err := newTestClient(t, r).NAT(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LANIP != "192.168.0.1" || len(cfg.Rules) != 7 {
		t.Fatalf("cfg = %+v", cfg)
	}
	if got := cfg.Rules[2]; got != (PortForward{IP: "192.168.0.5", InPort: 24454, OutPort: 24454, Protocol: ProtoUDP}) {
		t.Fatalf("rule 2 = %+v", got)
	}
}

func TestNATAddRule(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddNATRule(context.Background(), PortForward{IP: "192.168.0.9", InPort: 8080, OutPort: 9090, Protocol: ProtoTCP}); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetVirtualServerCfg")
	want := url.Values{"list": {natFixtureList + "~192.168.0.9,8080,9090,1"}}.Encode()
	if call.RawBody != want {
		t.Fatalf("body = %q\nwant   %q", call.RawBody, want)
	}
}

func TestNATAddRuleValidation(t *testing.T) {
	tests := []struct {
		name string
		rule PortForward
		is   error
	}{
		{"duplicate WAN port", PortForward{IP: "192.168.0.9", InPort: 1, OutPort: 443}, ErrNATDuplicatePort},
		{"router IP", PortForward{IP: "192.168.0.1", InPort: 1, OutPort: 1}, nil},
		{"outside subnet", PortForward{IP: "10.0.0.9", InPort: 1, OutPort: 1}, nil},
		{"separator in IP", PortForward{IP: "192.168.0.9~1", InPort: 1, OutPort: 1}, nil},
		{"port zero", PortForward{IP: "192.168.0.9", InPort: 0, OutPort: 1}, nil},
		{"port too big", PortForward{IP: "192.168.0.9", InPort: 1, OutPort: 65536}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tendatest.New(t)
			err := newTestClient(t, r).AddNATRule(context.Background(), tt.rule)
			if err == nil || (tt.is != nil && !errors.Is(err, tt.is)) {
				t.Fatalf("err = %v, want %v", err, tt.is)
			}
			if n := len(r.CallsTo("SetVirtualServerCfg")); n != 0 {
				t.Fatalf("posted %d times", n)
			}
		})
	}
}

func TestNATAddRuleFull(t *testing.T) {
	r := tendatest.New(t)
	var rows []string
	for i := range natMaxRules {
		rows = append(rows, `{"ip":"192.168.0.5","inPort":"`+strconv.Itoa(1000+i)+`","outPort":"`+strconv.Itoa(1000+i)+`","protocol":"0"}`)
	}
	r.Reply("GetVirtualServerCfg", 200, `{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","virtualList":[`+strings.Join(rows, ",")+`]}`)
	err := newTestClient(t, r).AddNATRule(context.Background(), PortForward{IP: "192.168.0.9", InPort: 1, OutPort: 1})
	if !errors.Is(err, ErrNATFull) || len(r.CallsTo("SetVirtualServerCfg")) != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestNATRemoveRule(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	ctx := context.Background()
	if err := c.RemoveNATRule(ctx, 443); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"list": {strings.TrimSuffix(natFixtureList, "~192.168.0.5,443,443,1")}}.Encode()
	if got := r.LastCall(t, "SetVirtualServerCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if err := c.RemoveNATRule(ctx, 9999); !errors.Is(err, ErrNATNoRule) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("SetVirtualServerCfg")); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
}

func TestNATSetRulesEmpty(t *testing.T) {
	r := tendatest.New(t)
	if err := newTestClient(t, r).SetNATRules(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "SetVirtualServerCfg").RawBody; got != "list=" {
		t.Fatalf("body = %q", got)
	}
}

func TestNATProtocol(t *testing.T) {
	for in, want := range map[string]Protocol{"both": ProtoBoth, "TCP": ProtoTCP, "udp": ProtoUDP, "0": ProtoBoth, "1": ProtoTCP, "2": ProtoUDP} {
		var p Protocol
		if err := p.Set(in); err != nil || p != want {
			t.Fatalf("Set(%q) = %v, %v", in, p, err)
		}
	}
	var p Protocol
	if p.Set("icmp") == nil {
		t.Fatal("want error")
	}
	if b, _ := ProtoTCP.MarshalText(); string(b) != "tcp" || ProtoBoth.String() != "TCP&UDP" {
		t.Fatal("spelling wrong")
	}
}
