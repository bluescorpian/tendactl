package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestUPnP(t *testing.T) {
	r := tendatest.New(t)
	u, err := newTestClient(t, r).UPnP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !u.Enabled || len(u.Mappings) != 0 {
		t.Fatalf("u = %+v", u)
	}
}

func TestUPnPMappings(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetUpnpCfg", 200, `[{"upnpEn":"0"},{"remoteHost":"","outPort":"80","host":"192.168.0.5","inPort":"80","protocol":"TCP"}]`)
	u, err := newTestClient(t, r).UPnP(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if u.Enabled || len(u.Mappings) != 1 {
		t.Fatalf("u = %+v", u)
	}
	if want := (UPnPMapping{Host: "192.168.0.5", OutPort: "80", InPort: "80", Protocol: "TCP"}); u.Mappings[0] != want {
		t.Fatalf("mapping = %+v, want %+v", u.Mappings[0], want)
	}
}

func TestSetUPnP(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetUPnP(context.Background(), UPnP{Enabled: false}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"upnpEn": {"0"}}.Encode()
	if got := r.LastCall(t, "SetUpnpCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
