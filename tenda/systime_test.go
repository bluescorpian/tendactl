package tenda

import (
	"context"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSysTime(t *testing.T) {
	r := tendatest.New(t)
	s, err := newTestClient(t, r).SysTime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.TimeZone != "14:00" || s.Synced || s.NTPServer != "time.windows.com" || s.ResyncSeconds != 86400 {
		t.Fatalf("s = %+v", s)
	}
}

func TestSetSysTime(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	s := SysTime{TimeZone: "17:30", NTPServer: "time.windows.com", ResyncSeconds: 86400}
	if err := c.SetSysTime(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"timeZone": {"17:30"}, "timePeriod": {"86400"}, "ntpServer": {"time.windows.com"}}.Encode()
	if got := r.LastCall(t, "SetSysTimeCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
