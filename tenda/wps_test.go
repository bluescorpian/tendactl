package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestWPS(t *testing.T) {
	r := tendatest.New(t)
	w, err := newTestClient(t, r).WPS(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.Enabled || !w.APMode || !w.RadioOn || w.PIN == "" {
		t.Fatalf("w = %+v", w)
	}
}

func TestSetWPS(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetWPS(context.Background(), WPS{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"wpsEn": {"1"}}.Encode()
	if got := r.LastCall(t, "WifiWpsSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestStartWPS(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.StartWPS(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"action": {"wps"}}.Encode()
	if got := r.LastCall(t, "WifiWpsStart").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
