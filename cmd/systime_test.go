package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSysTimeShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "systime", mustRun(t, r, "system", "time").Stdout)
	if got := mustRun(t, r, "system", "time", "show").Stdout; got != mustRun(t, r, "system", "time").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "systime_json", mustRun(t, r, "system", "time", "-o", "json").Stdout)
}

func TestSysTimeSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "system", "time", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestSysTimeSetInvalidZone(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "system", "time", "set", "--zone", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --zone") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestSysTimeSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "system", "time", "set", "--zone", "17:30")
	want := url.Values{"timeZone": {"17:30"}, "timePeriod": {"86400"}, "ntpServer": {"time.windows.com"}}.Encode()
	if got := r.LastCall(t, "SetSysTimeCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
