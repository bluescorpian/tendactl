package tenda

import (
	"context"
	"errors"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestReboot(t *testing.T) {
	r := tendatest.New(t)
	r.Allow("SysToolReboot")
	c := newTestClient(t, r, WithConfirm(allowAll))
	if err := c.Reboot(context.Background()); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "SysToolReboot")
	if call.Method != "POST" || call.Form.Get("action") != "0" {
		t.Fatalf("call = %+v", call)
	}
}

func TestRebootNeedsConfirm(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r) // no WithConfirm: every hazard is refused
	err := c.Reboot(context.Background())
	var he *HazardError
	if !errors.As(err, &he) || !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("err = %v, want *HazardError", err)
	}
	if len(r.Calls()) != 0 || r.Logins() != 0 {
		t.Fatalf("calls = %d, logins = %d; want none", len(r.Calls()), r.Logins())
	}
}
