package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestVPNClientShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "vpn", "client").Stdout
	golden(t, "vpnclient", text)
	if got := mustRun(t, r, "vpn", "client", "show").Stdout; got != text {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "vpnclient_json", mustRun(t, r, "vpn", "client", "-o", "json").Stdout)
}

func TestVPNClientShowPasswordMasked(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpClientCfg", 200, `{"clientEn":"1","clientType":"pptp","domain":"vpn.example.com","clientMppe":"0","clientMppeOp":"128","clientWanid":"1","userName":"bob","password":"s3cret","clientIp":"","clientMask":"","pptpStatus":"1","pptpIp":"10.8.0.5","l2tpStatus":"0","l2tpIp":"","wanConnType":"2","wanUser":"wanuser","wanIp":"203.0.113.10"}`)
	if got := mustRun(t, r, "vpn", "client").Stdout; strings.Contains(got, "s3cret") {
		t.Fatalf("password leaked: %q", got)
	}
	if got := mustRun(t, r, "vpn", "client", "--show-password").Stdout; !strings.Contains(got, "s3cret") {
		t.Fatalf("password not shown: %q", got)
	}
}

func TestVPNClientSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "client", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNClientSetInvalidType(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "client", "set", "--type", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --type") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNClientSetInvalidMPPE(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "client", "set", "--mppe", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --mppe") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNClientSetInvalidMPPEBits(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "client", "set", "--mppe-bits", "64")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --mppe-bits") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNClientSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	// The fixture's VPN client starts disabled; js/pptp_client.js's
	// getSubmitData() resends every field but clientEn from prev unchanged
	// whenever clientEn isn't "1", so these flags must not reach the wire.
	mustRun(t, r, "vpn", "client", "set", "--type", "l2tp", "--domain", "vpn.example.com", "--user", "bob", "--password", "s3cret")
	got := r.LastCall(t, "SetPptpClientCfg").Form
	if got.Get("clientEn") != "0" {
		t.Fatalf("clientEn = %v", got)
	}
	if got.Get("clientType") != "pptp" || got.Get("domain") != "" || got.Get("userName") != "" || got.Get("password") != "" {
		t.Fatalf("form = %v", got)
	}
}

func TestVPNClientSetWhileEnabled(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpClientCfg", 200, `{"clientEn":"1","clientType":"pptp","domain":"","clientMppe":"0","clientMppeOp":"128","clientWanid":"1","userName":"","password":"","clientIp":"","clientMask":"","pptpStatus":"0","pptpIp":"0.0.0.0","l2tpStatus":"0","l2tpIp":"","wanConnType":"2","wanUser":"REDACTED","wanIp":"203.0.113.10"}`)
	mustRun(t, r, "vpn", "client", "set", "--type", "l2tp", "--domain", "vpn.example.com", "--user", "bob", "--password", "s3cret")
	got := r.LastCall(t, "SetPptpClientCfg").Form
	if got.Get("clientType") != "l2tp" || got.Get("domain") != "vpn.example.com" || got.Get("userName") != "bob" || got.Get("password") != "s3cret" {
		t.Fatalf("form = %v", got)
	}
}

func TestVPNClientEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "vpn", "client", "enable")
	if got := r.LastCall(t, "SetPptpClientCfg").Form.Get("clientEn"); got != "1" {
		t.Fatalf("clientEn = %q", got)
	}
	mustRun(t, r, "vpn", "client", "disable")
	if got := r.LastCall(t, "SetPptpClientCfg").Form.Get("clientEn"); got != "0" {
		t.Fatalf("clientEn = %q", got)
	}
}
