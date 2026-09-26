package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestLED(t *testing.T) {
	r := tendatest.New(t)
	l, err := newTestClient(t, r).LED(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if l.Mode != "close" || l.Time != "00:00-07:00" || l.CloseType != "allClose" {
		t.Fatalf("l = %+v", l)
	}
}

func TestSetLED(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetLED(context.Background(), LED{Mode: "open", Time: "00:00-07:00", CloseType: "allClose"}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"ledType": {"open"}, "time": {"00:00-07:00"}, "ledCloseType": {"allClose"}}.Encode()
	if got := r.LastCall(t, "SetLEDCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
