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

func TestDHCP(t *testing.T) {
	r := tendatest.New(t)
	d, err := newTestClient(t, r).DHCP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.LANIP != "192.168.0.1" || d.LANMask != "255.255.255.0" {
		t.Fatalf("lan = %q %q", d.LANIP, d.LANMask)
	}
	if len(d.Clients) != 6 || len(d.Bindings) != 3 {
		t.Fatalf("clients = %d, bindings = %d", len(d.Clients), len(d.Bindings))
	}
	if got := d.Bindings[0]; got != (DHCPBinding{MAC: "02:00:00:00:00:08", IP: "192.168.0.31", Name: "Device-7", Online: true}) {
		t.Fatalf("binding 0 = %+v", got)
	}
}

func TestDHCPAddBinding(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddDHCPBinding(context.Background(), DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.0.50", Name: "New Device"}); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetIpMacBind")
	if call.Form.Get("bindnum") != "4" {
		t.Fatalf("bindnum = %q", call.Form.Get("bindnum"))
	}
	wantList := "Device-7\r02:00:00:00:00:08\r192.168.0.31\n" +
		"Device-8\r02:00:00:00:00:09\r192.168.0.32\n" +
		"Device-9\r02:00:00:00:00:0a\r192.168.0.30\n" +
		"New Device\raa:bb:cc:dd:ee:ff\r192.168.0.50"
	if got := call.Form.Get("list"); got != wantList {
		t.Fatalf("list = %q\nwant   %q", got, wantList)
	}
}

func TestDHCPAddBindingBody(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddDHCPBinding(context.Background(), DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.0.50", Name: "New Device"}); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetIpMacBind")
	if call.Form.Get("bindnum") != "4" {
		t.Fatalf("bindnum = %q", call.Form.Get("bindnum"))
	}
	wantList := "Device-7\r02:00:00:00:00:08\r192.168.0.31\n" +
		"Device-8\r02:00:00:00:00:09\r192.168.0.32\n" +
		"Device-9\r02:00:00:00:00:0a\r192.168.0.30\n" +
		"New Device\raa:bb:cc:dd:ee:ff\r192.168.0.50"
	want := url.Values{"bindnum": {"4"}, "list": {wantList}}.Encode()
	if got := call.RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestDHCPAddBindingOverwritesSameMAC(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddDHCPBinding(context.Background(), DHCPBinding{MAC: "02:00:00:00:00:08", IP: "192.168.0.99", Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetIpMacBind")
	if call.Form.Get("bindnum") != "3" {
		t.Fatalf("bindnum = %q, want unchanged count", call.Form.Get("bindnum"))
	}
	if !strings.Contains(call.Form.Get("list"), "Renamed\r02:00:00:00:00:08\r192.168.0.99") {
		t.Fatalf("list = %q", call.Form.Get("list"))
	}
}

func TestDHCPAddBindingValidation(t *testing.T) {
	tests := []struct {
		name string
		b    DHCPBinding
		is   error
	}{
		{"duplicate IP", DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.0.31"}, ErrDHCPDuplicateIP},
		{"router IP", DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.0.1"}, nil},
		{"reserved dhttpIP", DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "0.0.0.0"}, nil},
		{"invalid IP", DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "not-an-ip"}, nil},
		{"invalid MAC", DHCPBinding{MAC: "nope", IP: "192.168.0.60"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tendatest.New(t)
			err := newTestClient(t, r).AddDHCPBinding(context.Background(), tt.b)
			if err == nil || (tt.is != nil && !errors.Is(err, tt.is)) {
				t.Fatalf("err = %v, want %v", err, tt.is)
			}
			if n := len(r.CallsTo("SetIpMacBind")); n != 0 {
				t.Fatalf("posted %d times", n)
			}
		})
	}
}

func TestDHCPAddBindingFull(t *testing.T) {
	r := tendatest.New(t)
	var rows []string
	for i := range dhcpMaxBindings {
		rows = append(rows, `{"ipaddr":"10.0.0.`+strconv.Itoa(i+1)+`","macaddr":"aa:bb:cc:dd:ee:`+strconv.Itoa(10+i)+`","devname":"d","status":"1"}`)
	}
	r.Reply("GetIpMacBind", 200, `{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","dhttpIP":"0.0.0.0","dhcpClientList":[],"bindList":[`+strings.Join(rows, ",")+`]}`)
	err := newTestClient(t, r).AddDHCPBinding(context.Background(), DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.0.60"})
	if !errors.Is(err, ErrDHCPFull) || len(r.CallsTo("SetIpMacBind")) != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestDHCPRemoveBinding(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	ctx := context.Background()
	if err := c.RemoveDHCPBinding(ctx, "02:00:00:00:00:09"); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "SetIpMacBind").Form.Get("bindnum"); got != "2" {
		t.Fatalf("bindnum = %q", got)
	}
	if err := c.RemoveDHCPBinding(ctx, "AA:BB:CC:DD:EE:FF"); !errors.Is(err, ErrDHCPNoBinding) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("SetIpMacBind")); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
}

func TestDHCPSetReportsErrCode(t *testing.T) {
	r := tendatest.New(t)
	r.ErrCode("SetIpMacBind", 1)
	c := newTestClient(t, r)
	err := c.AddDHCPBinding(context.Background(), DHCPBinding{MAC: "AA:BB:CC:DD:EE:FF", IP: "192.168.0.50"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 1 {
		t.Fatalf("err = %v, want *APIError code 1", err)
	}
}
