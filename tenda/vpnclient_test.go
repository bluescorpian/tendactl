package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestVPNClient(t *testing.T) {
	r := tendatest.New(t)
	v, err := newTestClient(t, r).VPNClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v.Enabled || v.Type != "pptp" || v.MPPE || v.MPPEBits != 128 || v.User != "" || v.Password != "" {
		t.Fatalf("v = %+v", v)
	}
	if v.PPTPStatus != "disconnected" || v.PPTPIP != "0.0.0.0" || v.L2TPStatus != "disconnected" {
		t.Fatalf("v = %+v", v)
	}
}

func TestVPNClientStatus(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpClientCfg", 200, `{"clientEn":"1","clientType":"l2tp","domain":"vpn.example.com","clientMppe":"0","clientMppeOp":"128","clientWanid":"1","userName":"bob","password":"s3cret","clientIp":"","clientMask":"","pptpStatus":"2","pptpIp":"0.0.0.0","l2tpStatus":"1","l2tpIp":"10.8.0.5","wanConnType":"2","wanUser":"wanuser","wanIp":"203.0.113.10"}`)
	v, err := newTestClient(t, r).VPNClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v.PPTPStatus != "connecting" || v.L2TPStatus != "connected" || v.L2TPIP != "10.8.0.5" {
		t.Fatalf("v = %+v", v)
	}
}

func TestVPNClientSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	prev := VPNClient{Enabled: true, Type: "pptp", Domain: "old.example.com", MPPE: false, MPPEBits: 128, User: "alice", Password: "old"}
	next := VPNClient{Enabled: true, Type: "pptp", Domain: "vpn.example.com", MPPE: true, MPPEBits: 40, User: "bob", Password: "s3cret"}
	if err := c.SetVPNClient(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"clientEn": {"1"}, "clientType": {"pptp"}, "clientMppe": {"1"}, "clientMppeOp": {"40"},
		"domain": {"vpn.example.com"}, "userName": {"bob"}, "password": {"s3cret"},
	}.Encode()
	if got := r.LastCall(t, "SetPptpClientCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestVPNClientSetWhileDisabledResendsPrevFields(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	// js/pptp_client.js's getSubmitData() resends every field but clientEn
	// from prev unchanged whenever the submitted clientEn isn't "1".
	prev := VPNClient{Enabled: false, Type: "pptp", Domain: "vpn.example.com", MPPE: false, MPPEBits: 128, User: "bob", Password: "s3cret"}
	next := prev
	next.Type, next.Domain, next.User, next.Password = "l2tp", "new.example.com", "carol", "hunter2"
	next.MPPE, next.MPPEBits = true, 40
	if err := c.SetVPNClient(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"clientEn": {"0"}, "clientType": {"pptp"}, "clientMppe": {"0"}, "clientMppeOp": {"128"},
		"domain": {"vpn.example.com"}, "userName": {"bob"}, "password": {"s3cret"},
	}.Encode()
	if got := r.LastCall(t, "SetPptpClientCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestVPNClientSetL2TPResendsPrevMPPE(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	// getSubmitData()'s ternary: clientMppe/clientMppeOp come from the live
	// form only for clientType=="pptp"; for "l2tp" they're resent from prev
	// unchanged even while enabling, since MPPE isn't user-configurable
	// there.
	prev := VPNClient{Enabled: true, Type: "pptp", MPPE: false, MPPEBits: 128, Domain: "vpn.example.com", User: "bob", Password: "s3cret"}
	next := prev
	next.Type = "l2tp"
	next.MPPE, next.MPPEBits = true, 40
	if err := c.SetVPNClient(context.Background(), prev, next); err != nil {
		t.Fatal(err)
	}
	got := r.LastCall(t, "SetPptpClientCfg").Form
	if got.Get("clientType") != "l2tp" || got.Get("clientMppe") != "0" || got.Get("clientMppeOp") != "128" {
		t.Fatalf("form = %v", got)
	}
}
