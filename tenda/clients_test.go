package tenda

import (
	"context"
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
