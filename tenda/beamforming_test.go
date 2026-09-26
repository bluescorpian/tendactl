package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestBeamforming(t *testing.T) {
	r := tendatest.New(t)
	b, err := newTestClient(t, r).Beamforming(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !b.Enabled {
		t.Fatalf("b = %+v", b)
	}
}

func TestBeamformingSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetBeamforming(context.Background(), Beamforming{Enabled: false}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"beamformingEn": {"0"}}.Encode()
	if got := r.LastCall(t, "WifiBeamformingSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
