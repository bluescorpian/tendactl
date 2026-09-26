package cmd

import (
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestBeamformingShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "beamforming", mustRun(t, r, "wifi", "beamforming").Stdout)
	if got := mustRun(t, r, "wifi", "beamforming", "show").Stdout; got != mustRun(t, r, "wifi", "beamforming").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "beamforming_json", mustRun(t, r, "wifi", "beamforming", "-o", "json").Stdout)
}

func TestBeamformingEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "wifi", "beamforming", "disable")
	want := url.Values{"beamformingEn": {"0"}}.Encode()
	if got := r.LastCall(t, "WifiBeamformingSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if res.Stdout != "Beamforming+ disabled\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	mustRun(t, r, "wifi", "beamforming", "enable")
	want = url.Values{"beamformingEn": {"1"}}.Encode()
	if got := r.LastCall(t, "WifiBeamformingSet").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
