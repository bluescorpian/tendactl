package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestDDNSShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "ddns", mustRun(t, r, "ddns").Stdout)
	if got := mustRun(t, r, "ddns", "show").Stdout; got != mustRun(t, r, "ddns").Stdout {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "ddns_json", mustRun(t, r, "ddns", "-o", "json").Stdout)
}

func TestDDNSShowPasswordMasked(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetDDNSCfg", 200, `{"ddnsEn":"1","serverName":"no-ip.com","ddnsUser":"bob","ddnsPwd":"s3cret","ddnsDomain":"bob.example.com","ddnsStatus":"1"}`)
	if got := mustRun(t, r, "ddns").Stdout; strings.Contains(got, "s3cret") {
		t.Fatalf("password leaked: %q", got)
	}
	if got := mustRun(t, r, "ddns", "--show-password").Stdout; !strings.Contains(got, "s3cret") {
		t.Fatalf("password not shown: %q", got)
	}
}

func TestDDNSSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "ddns", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestDDNSSetInvalidProvider(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "ddns", "set", "--provider", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --provider") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestDDNSSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "ddns", "set", "--provider", "dyndns.org", "--domain", "bob.example.com", "--user", "bob", "--password", "s3cret")
	got := r.LastCall(t, "SetDDNSCfg").Form
	if got.Get("serverName") != "dyn.com/dns/" || got.Get("ddnsDomain") != "bob.example.com" || got.Get("ddnsUser") != "bob" || got.Get("ddnsPwd") != "s3cret" {
		t.Fatalf("form = %v", got)
	}
}

func TestDDNSEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "ddns", "enable")
	if got := r.LastCall(t, "SetDDNSCfg").Form.Get("ddnsEn"); got != "1" {
		t.Fatalf("ddnsEn = %q", got)
	}
	mustRun(t, r, "ddns", "disable")
	if got := r.LastCall(t, "SetDDNSCfg").Form.Get("ddnsEn"); got != "0" {
		t.Fatalf("ddnsEn = %q", got)
	}
}
