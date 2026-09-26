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

func TestRoutes(t *testing.T) {
	r := tendatest.New(t)
	cfg, err := newTestClient(t, r).Routes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LANIP != "192.168.0.1" || len(cfg.Routes) != 5 {
		t.Fatalf("cfg = %+v", cfg)
	}
	for _, rt := range cfg.Routes {
		if !rt.System || !rt.Active {
			t.Fatalf("route = %+v, want all system+active on the live fixture", rt)
		}
	}
	if got := cfg.Routes[2]; got != (Route{Network: "192.168.0.0", Mask: "255.255.255.0", Gateway: "0.0.0.0", Interface: "br0", System: true, Active: true}) {
		t.Fatalf("route 2 = %+v", got)
	}
}

func TestRouteAdd(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddRoute(context.Background(), Route{Network: "10.0.0.0", Mask: "255.255.255.0", Gateway: "192.168.0.254", Interface: "WAN1"}); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SetStaticRouteCfg")
	want := url.Values{"list": {"10.0.0.0,255.255.255.0,192.168.0.254,WAN1"}}.Encode()
	if call.RawBody != want {
		t.Fatalf("body = %q\nwant   %q", call.RawBody, want)
	}
}

func TestRouteAddDefaults(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddRoute(context.Background(), Route{Network: "10.0.0.0", Mask: "255.255.255.0"}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"list": {"10.0.0.0,255.255.255.0,0.0.0.0,WAN1"}}.Encode()
	if got := r.LastCall(t, "SetStaticRouteCfg").RawBody; got != want {
		t.Fatalf("body = %q\nwant   %q", got, want)
	}
}

func TestRouteAddMasksNetwork(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	// js/static_route.js's blur handler always rewrites #network to
	// mask&network before the UI can submit it, so a network with host
	// bits set (here, .5 under a /24) must reach the router as .0, not
	// verbatim.
	if err := c.AddRoute(context.Background(), Route{Network: "10.0.0.5", Mask: "255.255.255.0", Gateway: "192.168.0.254"}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"list": {"10.0.0.0,255.255.255.0,192.168.0.254,WAN1"}}.Encode()
	if got := r.LastCall(t, "SetStaticRouteCfg").RawBody; got != want {
		t.Fatalf("body = %q\nwant   %q", got, want)
	}
}

func TestRouteAddValidation(t *testing.T) {
	tests := []struct {
		name  string
		route Route
	}{
		{"invalid network", Route{Network: "not-an-ip", Mask: "255.255.255.0"}},
		{"invalid mask", Route{Network: "10.0.0.0", Mask: "bogus"}},
		{"invalid gateway", Route{Network: "10.0.0.0", Mask: "255.255.255.0", Gateway: "bogus"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tendatest.New(t)
			err := newTestClient(t, r).AddRoute(context.Background(), tt.route)
			if err == nil {
				t.Fatal("want error")
			}
			if n := len(r.CallsTo("SetStaticRouteCfg")); n != 0 {
				t.Fatalf("posted %d times", n)
			}
		})
	}
}

func TestRouteAddFull(t *testing.T) {
	r := tendatest.New(t)
	var rows []string
	for i := range routeMaxUserRoutes {
		rows = append(rows, `{"network":"10.0.`+strconv.Itoa(i)+`.0","mask":"255.255.255.0","gateway":"0.0.0.0","ifname":"WAN1","operateType":"1","effective":"1"}`)
	}
	r.Reply("GetStaticRouteCfg", 200, `{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","wanMask":"255.255.255.255","wanGateway":"203.0.113.11","routeList":[`+strings.Join(rows, ",")+`]}`)
	err := newTestClient(t, r).AddRoute(context.Background(), Route{Network: "10.1.0.0", Mask: "255.255.255.0"})
	if !errors.Is(err, ErrRouteFull) || len(r.CallsTo("SetStaticRouteCfg")) != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestRouteRemove(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetStaticRouteCfg", 200, `{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","wanMask":"255.255.255.255","wanGateway":"203.0.113.11","routeList":[`+
		`{"network":"0.0.0.0","mask":"0.0.0.0","gateway":"203.0.113.11","ifname":"WAN1","operateType":"0","effective":"1"},`+
		`{"network":"10.0.0.0","mask":"255.255.255.0","gateway":"0.0.0.0","ifname":"WAN1","operateType":"1","effective":"1"},`+
		`{"network":"10.1.0.0","mask":"255.255.255.0","gateway":"0.0.0.0","ifname":"WAN1","operateType":"1","effective":"1"}]}`)
	c := newTestClient(t, r)
	ctx := context.Background()
	if err := c.RemoveRoute(ctx, "10.0.0.0", "255.255.255.0"); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"list": {"10.1.0.0,255.255.255.0,0.0.0.0,WAN1"}}.Encode()
	if got := r.LastCall(t, "SetStaticRouteCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	// A system route with a matching network/mask does not count as a match.
	if err := c.RemoveRoute(ctx, "0.0.0.0", "0.0.0.0"); !errors.Is(err, ErrRouteNoMatch) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("SetStaticRouteCfg")); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
}

func TestRouteRemoveNoMatch(t *testing.T) {
	r := tendatest.New(t)
	if err := newTestClient(t, r).RemoveRoute(context.Background(), "10.0.0.0", "255.255.255.0"); !errors.Is(err, ErrRouteNoMatch) {
		t.Fatalf("err = %v", err)
	}
}
