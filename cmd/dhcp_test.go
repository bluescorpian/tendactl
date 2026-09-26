package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestDHCPList(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "dhcp").Stdout
	golden(t, "dhcp", text)
	if got := mustRun(t, r, "dhcp", "list").Stdout; got != text {
		t.Fatalf("dhcp list differs from dhcp:\n%s", got)
	}
	golden(t, "dhcp_json", mustRun(t, r, "dhcp", "-o", "json").Stdout)
}

func TestDHCPListEmpty(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetIpMacBind", 200, `{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","dhttpIP":"0.0.0.0","dhcpClientList":[],"bindList":[]}`)
	got := mustRun(t, r, "dhcp").Stdout
	if !strings.Contains(got, "No DHCP reservations configured") {
		t.Fatalf("got %q", got)
	}
	if got := mustRun(t, r, "dhcp", "-o", "json").Stdout; !strings.Contains(got, `"bindings": []`) || !strings.Contains(got, `"clients": []`) {
		t.Fatalf("got %q", got)
	}
}

func TestDHCPAdd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "dhcp", "add", "aa:bb:cc:dd:ee:ff", "192.168.0.50", "--name", "New Device")
	if got := r.LastCall(t, "SetIpMacBind").Form.Get("list"); !strings.Contains(got, "New Device\raa:bb:cc:dd:ee:ff\r192.168.0.50") {
		t.Fatalf("list = %q", got)
	}
	if res.Stdout != "Reserved 192.168.0.50 for aa:bb:cc:dd:ee:ff\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestDHCPAddRefused(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "dhcp", "add", "aa:bb:cc:dd:ee:ff", "192.168.0.31") // duplicate IP
	if res.Code != 1 || res.Stderr == "" || len(r.CallsTo("SetIpMacBind")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestDHCPRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "dhcp", "rm", "02:00:00:00:00:09")
	if got := r.LastCall(t, "SetIpMacBind").Form.Get("bindnum"); got != "2" {
		t.Fatalf("bindnum = %q", got)
	}

	r = tendatest.New(t)
	res := runCLI(t, r, "dhcp", "rm", "aa:bb:cc:dd:ee:ff")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no DHCP reservation") || len(r.CallsTo("SetIpMacBind")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}
