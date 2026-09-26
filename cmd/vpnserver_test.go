package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestVPNServerShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "vpn", "server").Stdout
	golden(t, "vpnserver", text)
	if got := mustRun(t, r, "vpn", "server", "show").Stdout; got != text {
		t.Fatalf("show differs from bare:\n%s", got)
	}
	golden(t, "vpnserver_json", mustRun(t, r, "vpn", "server", "-o", "json").Stdout)
}

func TestVPNServerSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "server", "set")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNServerSetInvalidMPPE(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "server", "set", "--mppe", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --mppe") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNServerSetInvalidMPPEBits(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "vpn", "server", "set", "--mppe-bits", "64")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --mppe-bits") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNServerSet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	// The fixture's PPTP server starts disabled; js/pptp_server.js's
	// getSubmitData() resends prev's startIp/endIp/mppe/mppeOp unchanged
	// whenever serverEn isn't "1", so these flags must not reach the wire.
	mustRun(t, r, "vpn", "server", "set", "--pool-start", "10.0.0.50", "--pool-end", "10.0.0.60", "--mppe", "on", "--mppe-bits", "40")
	got := r.LastCall(t, "SetPptpServerCfg").Form
	if got.Get("serverEn") != "0" {
		t.Fatalf("serverEn = %v", got)
	}
	if got.Get("startIp") != "10.0.0.100" || got.Get("endIp") != "10.0.0.200" || got.Get("mppe") != "0" || got.Get("mppeOp") != "128" {
		t.Fatalf("form = %v", got)
	}
}

func TestVPNServerSetWhileEnabled(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, `[{"serverEn":"1","wanid":"1","mppe":"0","mppeOp":"128","startIp":"10.0.0.100","endIp":"10.0.0.200","lanIp":"192.168.0.1","lanMask":"255.255.255.0","guestIp":"192.168.10.1","guestMask":"255.255.255.0","serverIp":"","vlan2Ip":"","vlan2Mask":"","wanIp":"203.0.113.10","wanMask":"255.255.255.255","pptpSvrIp":"10.0.0.1","pptpSvrMask":"255.255.255.0"}]`)
	mustRun(t, r, "vpn", "server", "set", "--pool-start", "10.0.0.50", "--pool-end", "10.0.0.60", "--mppe", "on", "--mppe-bits", "40")
	got := r.LastCall(t, "SetPptpServerCfg").Form
	if got.Get("startIp") != "10.0.0.50" || got.Get("endIp") != "10.0.0.60" || got.Get("mppe") != "1" || got.Get("mppeOp") != "40" {
		t.Fatalf("form = %v", got)
	}
}

func TestVPNServerEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "vpn", "server", "enable")
	if got := r.LastCall(t, "SetPptpServerCfg").Form.Get("serverEn"); got != "1" {
		t.Fatalf("serverEn = %q", got)
	}
	mustRun(t, r, "vpn", "server", "disable")
	if got := r.LastCall(t, "SetPptpServerCfg").Form.Get("serverEn"); got != "0" {
		t.Fatalf("serverEn = %q", got)
	}
}

const vpnServerUsersFixture = `[{"serverEn":"0","wanid":"1","mppe":"0","mppeOp":"128","startIp":"10.0.0.100","endIp":"10.0.0.200","lanIp":"192.168.0.1","lanMask":"255.255.255.0","guestIp":"192.168.10.1","guestMask":"255.255.255.0","serverIp":"","vlan2Ip":"","vlan2Mask":"","wanIp":"203.0.113.10","wanMask":"255.255.255.255","pptpSvrIp":"10.0.0.1","pptpSvrMask":"255.255.255.0"},{"userName":"alice","password":"s3cret","enable":"1","connsta":"1"}]`

func TestVPNServerUsersList(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerUsersFixture)
	text := mustRun(t, r, "vpn", "users").Stdout
	if strings.Contains(text, "s3cret") {
		t.Fatalf("password leaked: %q", text)
	}
	golden(t, "vpnserverusers", text)
	if got := mustRun(t, r, "vpn", "users", "list").Stdout; got != text {
		t.Fatalf("list differs from bare:\n%s", got)
	}
	jsonOut := mustRun(t, r, "vpn", "users", "-o", "json").Stdout
	if strings.Contains(jsonOut, "s3cret") {
		t.Fatalf("password leaked in json: %q", jsonOut)
	}
	golden(t, "vpnserverusers_json", jsonOut)
	if got := mustRun(t, r, "vpn", "users", "--show-password").Stdout; !strings.Contains(got, "s3cret") {
		t.Fatalf("password not shown: %q", got)
	}
}

func TestVPNServerUsersListEmpty(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	if got := mustRun(t, r, "vpn", "users").Stdout; got != "No PPTP server user accounts configured\n" {
		t.Fatalf("got %q", got)
	}
	if got := mustRun(t, r, "vpn", "users", "-o", "json").Stdout; got != "[]\n" {
		t.Fatalf("got %q", got)
	}
}

func TestVPNServerUsersAdd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerUsersFixture)
	res := mustRun(t, r, "vpn", "users", "add", "carol", "--password", "hunter2")
	if res.Stdout != "Added PPTP server user carol\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if got := r.LastCall(t, "setPptpUserList").Form.Get("list"); !strings.Contains(got, "carol;hunter2;1;0;;;") {
		t.Fatalf("list = %q", got)
	}
}

func TestVPNServerUsersRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerUsersFixture)
	mustRun(t, r, "vpn", "users", "rm", "alice")
	if got := r.LastCall(t, "setPptpUserList").Form.Get("list"); got != "" {
		t.Fatalf("list = %q, want empty", got)
	}

	res := runCLI(t, r, "vpn", "users", "rm", "nobody")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no PPTP server user") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNServerUsersEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerUsersFixture)
	mustRun(t, r, "vpn", "users", "disable", "alice")
	if got := r.LastCall(t, "setPptpUserList").Form.Get("list"); !strings.Contains(got, "alice;s3cret;0;0;;;") {
		t.Fatalf("list = %q", got)
	}

	res := runCLI(t, r, "vpn", "users", "enable", "nobody")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no PPTP server user") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestVPNServerOnline(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "vpnserveronline", mustRun(t, r, "vpn", "online").Stdout)
	golden(t, "vpnserveronline_json", mustRun(t, r, "vpn", "online", "-o", "json").Stdout)
}

func TestVPNServerOnlineWithClients(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("getPptpOnlineClient", 200, `{"clientList":[{"username":"alice","dialIP":"203.0.113.5","clientIP":"10.0.0.101","onlineTime":42}]}`)
	got := mustRun(t, r, "vpn", "online").Stdout
	if !strings.Contains(got, "alice") || !strings.Contains(got, "203.0.113.5") || !strings.Contains(got, "42") {
		t.Fatalf("got %q", got)
	}
}
