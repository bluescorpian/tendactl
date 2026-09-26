package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestRemote(t *testing.T) {
	r := tendatest.New(t)
	rm, err := newTestClient(t, r).Remote(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rm.Enabled || rm.FromIP != "0.0.0.0" || rm.Port != 8080 || !rm.PasswordSet || rm.LANIP != "192.168.0.1" {
		t.Fatalf("rm = %+v", rm)
	}
}

func TestSetRemote(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	rm := Remote{Enabled: true, FromIP: "203.0.113.5", Port: 9443}
	if err := c.SetRemote(context.Background(), rm); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"remoteWebEn": {"1"}, "remoteIp": {"203.0.113.5"}, "remotePort": {"9443"}}.Encode()
	if got := r.LastCall(t, "SetRemoteWebCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
