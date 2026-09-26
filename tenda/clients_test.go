package tenda

import (
	"context"
	"errors"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestClientOnlineList(t *testing.T) {
	r := tendatest.New(t)
	l, err := newTestClient(t, r).OnlineList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if l.Host.Name != "Device-11" || l.Host.IP != "192.168.0.2" || l.Host.Wireless || l.MACFilterMode != "black" {
		t.Fatalf("head = %+v %q", l.Host, l.MACFilterMode)
	}
	if len(l.Clients) != 13 {
		t.Fatalf("clients = %d", len(l.Clients))
	}
	first := l.Clients[0]
	if first.MAC != "02:00:00:00:00:04" || first.DownKBps != "144" || first.Line != "wired" || first.Guest || first.Blocked {
		t.Fatalf("first = %+v", first)
	}
	if l.Clients[2].Line != "2.4" {
		t.Fatalf("line = %q", l.Clients[2].Line)
	}
}

func TestClientOnlineListOnlyHost(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getOnlineList", 200, `[{"blackNum":0,"macFilterType":"white","localhostIP":"1","isWirelessConnect":"true","localhostName":"h","localhostMac":"m"}]`)
	l, err := newTestClient(t, r).OnlineList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if l.Clients == nil || len(l.Clients) != 0 || !l.Host.Wireless {
		t.Fatalf("list = %+v", l)
	}
}

func TestClientRename(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.RenameClient(context.Background(), "aa:bb:cc:dd:ee:ff", "Kids Tablet"); err != nil {
		t.Fatal(err)
	}
	got := r.LastCall(t, "SetOnlineDevName").Form
	if got.Get("mac") != "aa:bb:cc:dd:ee:ff" || got.Get("devName") != "Kids Tablet" {
		t.Fatalf("form = %v", got)
	}
	if err := c.RenameClient(context.Background(), "aa:bb:cc:dd:ee:ff", "01234567890123456789X"); !errors.Is(err, ErrClientNameTooLong) {
		t.Fatalf("err = %v", err)
	}
	if err := c.RenameClient(context.Background(), "not-a-mac", "x"); err == nil {
		t.Fatal("want error")
	}
}

func TestClientBlock(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	ctx := context.Background()

	// The fixture's local host is 02:00:00:00:00:02.
	if err := c.BlockClient(ctx, "02:00:00:00:00:02"); !errors.Is(err, ErrClientIsLocalhost) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("setBlackRule")); n != 0 {
		t.Fatalf("posts = %d, want 0", n)
	}

	if err := c.BlockClient(ctx, "02:00:00:00:00:04"); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "setBlackRule").Form.Get("mac"); got != "02:00:00:00:00:04" {
		t.Fatalf("mac = %q", got)
	}

	r.ErrCode("setBlackRule", 1)
	if err := c.BlockClient(ctx, "02:00:00:00:00:05"); !errors.Is(err, ErrClientBlocklistFull) {
		t.Fatalf("err = %v", err)
	}
}

func TestClientUnblock(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.UnblockClient(context.Background(), "aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "delBlackRule").Form.Get("mac"); got != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("mac = %q", got)
	}
}

func TestClientBlockedClients(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getBlackRuleList", 200, `[{"deviceId":"AA:BB:CC:DD:EE:FF","devName":"Old Phone"}]`)
	bl, err := newTestClient(t, r).BlockedClients(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(bl) != 1 || bl[0].MAC != "AA:BB:CC:DD:EE:FF" || bl[0].Name != "Old Phone" {
		t.Fatalf("bl = %+v", bl)
	}
}

func TestClientBlockedClientsEmpty(t *testing.T) {
	r := tendatest.New(t)
	bl, err := newTestClient(t, r).BlockedClients(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bl == nil || len(bl) != 0 {
		t.Fatalf("bl = %+v", bl)
	}
}
