package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestLEDShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "led", mustRun(t, r, "led").Stdout)
	if got := mustRun(t, r, "led", "show").Stdout; got != mustRun(t, r, "led").Stdout {
		t.Fatalf("led show differs from led:\n%s", got)
	}
	golden(t, "led_json", mustRun(t, r, "led", "-o", "json").Stdout)
}

func TestLEDEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "led", "enable")
	want := url.Values{"ledType": {"open"}, "time": {"00:00-07:00"}, "ledCloseType": {"allClose"}}.Encode()
	if got := r.LastCall(t, "SetLEDCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	mustRun(t, r, "led", "disable")
	want = url.Values{"ledType": {"close"}, "time": {"00:00-07:00"}, "ledCloseType": {"allClose"}}.Encode()
	if got := r.LastCall(t, "SetLEDCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestLEDSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "led", "set", "--mode", "time", "--time", "22:00-06:00")
	want := url.Values{"ledType": {"time"}, "time": {"22:00-06:00"}, "ledCloseType": {"allClose"}}.Encode()
	if got := r.LastCall(t, "SetLEDCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if res.Stdout != "LED settings updated\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestLEDSetErrors(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"led", "set"},
		{"led", "set", "--mode", "bogus"},
		{"led", "set", "--time", "bogus"},
		{"led", "set", "--close-type", "bogus"},
	} {
		r := tendatest.New(t)
		res := runCLI(t, r, args...)
		if res.Code != 1 || res.Stderr == "" || len(r.CallsTo("SetLEDCfg")) != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, res.Code, res.Stderr)
		}
	}
	r := tendatest.New(t)
	res := runCLI(t, r, "led", "set")
	if !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("stderr = %q", res.Stderr)
	}
}
