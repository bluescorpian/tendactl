package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestAntijam(t *testing.T) {
	r := tendatest.New(t)
	a, err := newTestClient(t, r).Antijam(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a.Mode != "auto" {
		t.Fatalf("a = %+v", a)
	}
}

func TestAntijamSet(t *testing.T) {
	for mode, wire := range map[string]string{"auto": "auto", "enable": "true", "disable": "false"} {
		t.Run(mode, func(t *testing.T) {
			r := tendatest.New(t)
			c := newTestClient(t, r)
			if err := c.SetAntijam(context.Background(), Antijam{Mode: mode}); err != nil {
				t.Fatal(err)
			}
			want := url.Values{"WifiAntijamEn": {wire}}.Encode()
			if got := r.LastCall(t, "WifiAntijamSet").RawBody; got != want {
				t.Fatalf("body = %q, want %q", got, want)
			}
		})
	}
}
