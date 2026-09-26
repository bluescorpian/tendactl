package cmd

import (
	"testing"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestBandwidthList(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "bandwidth").Stdout
	golden(t, "bandwidth", text)
	if got := mustRun(t, r, "bandwidth", "list").Stdout; got != text {
		t.Fatalf("bandwidth list differs from bandwidth:\n%s", got)
	}
	golden(t, "bandwidth_json", mustRun(t, r, "bandwidth", "-o", "json").Stdout)
}

func TestBandwidthSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "bandwidth", "set", "02:00:00:00:00:04")
	if res.Code != 1 || res.Stderr == "" {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestBandwidthSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "bandwidth", "set", "02:00:00:00:00:04", "--up", "2", "--down", "0.5")
	rows := tenda.LineCR.Decode(r.LastCall(t, "SetNetControlList").Form.Get("list"))
	if rows[0][2] != "256" || rows[0][3] != "64" {
		t.Fatalf("row 0 = %v", rows[0])
	}
	if res.Stdout != "Bandwidth cap set for 02:00:00:00:00:04\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestBandwidthRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "bandwidth", "rm", "02:00:00:00:00:04")
	if r.LastCall(t, "SetNetControlList").Form.Get("list") == "" {
		t.Fatal("empty list")
	}
}

func TestBandwidthEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "bandwidth", "enable")
	if got := r.LastCall(t, "SetNetControlList").Form.Get("netControlEn"); got != "1" {
		t.Fatalf("netControlEn = %q", got)
	}
	mustRun(t, r, "bandwidth", "disable")
	if got := r.LastCall(t, "SetNetControlList").Form.Get("netControlEn"); got != "0" {
		t.Fatalf("netControlEn = %q", got)
	}
}
