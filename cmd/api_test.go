package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestAPIGet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "api_get_dmz", mustRun(t, r, "api", "get", "GetDMZCfg").Stdout)
	if got := mustRun(t, r, "api", "get", "/goform/GetDMZCfg", "-o", "json").Stdout; got != readGolden(t, "api_get_dmz.golden") {
		t.Fatalf("got %q", got)
	}

	want := "{\n  \"wan_sta\": 1\n}\n"
	for _, args := range [][]string{
		{"api", "get", "cloudv2", "module=wansta", "opt=query"},
		{"api", "get", "cloudv2?module=wansta&opt=query"},
	} {
		if got := mustRun(t, r, args...).Stdout; got != want {
			t.Fatalf("%v = %q", args, got)
		}
	}

	mustRun(t, r, "api", "get", "GetDMZCfg", "a=1", "a=2", "b=x=y")
	if q := r.LastCall(t, "GetDMZCfg").Query; strings.Join(q["a"], ",") != "1,2" || q.Get("b") != "x=y" {
		t.Fatalf("query = %v", q)
	}

	if got := mustRun(t, r, "api", "get", "cgi-bin/DownloadCfg/RouterCfm.cfg").Stdout; !bytes.Equal([]byte(got), tendatest.ConfigBytes) {
		t.Fatalf("download = %q", got)
	}
}

func TestAPIBadPair(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "api", "set", "SetDMZCfg", "dmzEn")
	if res.Code != 1 || !strings.Contains(res.Stderr, "key=value") || len(r.Calls()) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestAPISet(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "api", "set", "SetDMZCfg", "dmzEn=1", "dmzIp=192.168.0.100")
	if call := r.LastCall(t, "SetDMZCfg"); call.Method != http.MethodPost || call.RawBody != "dmzEn=1&dmzIp=192.168.0.100" {
		t.Fatalf("call = %+v", call)
	}
	if res.Stdout != "{\n  \"errCode\": 0\n}\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}

	mustRun(t, r, "api", "set", "cloudv2", "module=manage", "opt=setbasic", "enable=0")
	call := r.LastCall(t, "cloudv2?module=manage&opt=setbasic")
	if call.RawBody != "enable=0" || call.Query.Get("module") != "manage" {
		t.Fatalf("cloudv2 call = %+v", call)
	}
}

func TestAPISetErrCode(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.ErrCode("SetDMZCfg", 1)
	res := runCLI(t, r, "api", "set", "SetDMZCfg", "dmzEn=1", "dmzIp=x")
	if res.Code != 1 || res.Stdout != "{\n  \"errCode\": 1\n}\n" || res.Stderr != "tendactl: SetDMZCfg: router returned errCode 1\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", res.Code, res.Stdout, res.Stderr)
	}
}

func TestAPIHazardNeedsYes(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "api", "get", "SysToolReboot")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--yes") || !strings.Contains(res.Stderr, "refusing SysToolReboot") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if len(r.Calls()) != 0 || r.Logins() != 0 {
		t.Fatalf("calls %d, logins %d", len(r.Calls()), r.Logins())
	}

	res = runCLI(t, r, "api", "set", "WifiBasicSet", "wrlEn=1", "wrlEn_5g=0")
	if res.Code != 1 || len(r.Calls()) != 0 {
		t.Fatalf("WifiBasicSet without --yes: exit %d, calls %d", res.Code, len(r.Calls()))
	}

	r.Allow("SysToolReboot")
	res = mustRun(t, r, "api", "get", "SysToolReboot", "--yes")
	if n := len(r.CallsTo("SysToolReboot")); n != 1 {
		t.Fatalf("reboot calls = %d, want 1", n)
	}
	if !strings.Contains(res.Stderr, "HTTP 302 Location: /system_reboot.html") {
		t.Fatalf("stderr = %q", res.Stderr)
	}
}
