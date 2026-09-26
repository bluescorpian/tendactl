package cmd

import (
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestAntijamShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "antijam", mustRun(t, r, "wifi", "antijam").Stdout)
	if got := mustRun(t, r, "wifi", "antijam", "show").Stdout; got != mustRun(t, r, "wifi", "antijam").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "antijam_json", mustRun(t, r, "wifi", "antijam", "-o", "json").Stdout)
}

func TestAntijamSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "wifi", "antijam", "set", "enable")
	want := url.Values{"WifiAntijamEn": {"true"}}.Encode()
	if got := r.LastCall(t, "WifiAntijamSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if res.Stdout != "Anti-interference set to enable\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestAntijamSetInvalid(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "antijam", "set", "bogus")
	if res.Code != 1 || res.Stderr == "" || len(r.CallsTo("WifiAntijamSet")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
