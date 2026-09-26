package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestClientsCmd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "clients").Stdout
	golden(t, "clients", text)
	for _, args := range [][]string{{"clients", "list"}, {"online"}, {"online", "list"}} {
		if got := mustRun(t, r, args...).Stdout; got != text {
			t.Fatalf("%v output differs from clients:\n%s", args, got)
		}
	}
	golden(t, "clients_json", mustRun(t, r, "clients", "-o", "json").Stdout)
}

func TestClientsRename(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "clients", "rename", "aa:bb:cc:dd:ee:ff", "New Name")
	if got := r.LastCall(t, "SetOnlineDevName").Form; got.Get("mac") != "aa:bb:cc:dd:ee:ff" || got.Get("devName") != "New Name" {
		t.Fatalf("form = %v", got)
	}
	if res.Stdout != `Renamed aa:bb:cc:dd:ee:ff to "New Name"`+"\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestClientsBlockUnblock(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "clients", "block", "02:00:00:00:00:04")
	if got := r.LastCall(t, "setBlackRule").Form.Get("mac"); got != "02:00:00:00:00:04" {
		t.Fatalf("mac = %q", got)
	}
	mustRun(t, r, "clients", "unblock", "02:00:00:00:00:04")
	if got := r.LastCall(t, "delBlackRule").Form.Get("mac"); got != "02:00:00:00:00:04" {
		t.Fatalf("mac = %q", got)
	}
}

func TestClientsBlockLocalhostRefused(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "clients", "block", "02:00:00:00:00:02")
	if res.Code != 1 || !strings.Contains(res.Stderr, "local host") || len(r.CallsTo("setBlackRule")) != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestClientsBlocked(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("getBlackRuleList", 200, `[{"deviceId":"AA:BB:CC:DD:EE:FF","devName":"Old Phone"}]`)
	golden(t, "clients_blocked", mustRun(t, r, "clients", "blocked").Stdout)
	golden(t, "clients_blocked_json", mustRun(t, r, "clients", "blocked", "-o", "json").Stdout)
}

func TestClientsBlockedEmpty(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	if got := mustRun(t, r, "clients", "blocked").Stdout; got != "No blocked devices\n" {
		t.Fatalf("got %q", got)
	}
	if got := mustRun(t, r, "clients", "blocked", "-o", "json").Stdout; got != "[]\n" {
		t.Fatalf("got %q", got)
	}
}
