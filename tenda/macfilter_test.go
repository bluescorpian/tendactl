package tenda

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestMACFilter(t *testing.T) {
	r := tendatest.New(t)
	f, err := newTestClient(t, r).MACFilter(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != "black" || len(f.Devices) != 0 {
		t.Fatalf("f = %+v", f)
	}
}

func TestMACFilterWhiteList(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getMacFilterCfg", 200, `{"localhostIP":"192.168.0.2","localhostName":"h","localhostMac":"m","macFilterType":"white","blackList":[],"whiteList":[{"devName":"Device6","devMac":"6C:4C:BC:83:95:2B"}],"onlineList":[]}`)
	f, err := newTestClient(t, r).MACFilter(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f.Mode != "white" || len(f.Devices) != 1 || f.Devices[0].MAC != "6C:4C:BC:83:95:2B" {
		t.Fatalf("f = %+v", f)
	}
}

func TestMACFilterAdd(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.AddMACFilterEntry(context.Background(), "6c:4c:bc:83:95:2b", "Device6"); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "setMacFilterCfg")
	if call.Form.Get("macFilterType") != "black" {
		t.Fatalf("macFilterType = %q", call.Form.Get("macFilterType"))
	}
	if got := call.Form.Get("deviceList"); got != "Device6\r6C:4C:BC:83:95:2B" {
		t.Fatalf("deviceList = %q", got)
	}
}

func TestMACFilterAddDuplicate(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getMacFilterCfg", 200, `{"localhostIP":"1","localhostName":"h","localhostMac":"m","macFilterType":"black","blackList":[{"devName":"D","devMac":"AA:BB:CC:DD:EE:FF"}],"whiteList":[],"onlineList":[]}`)
	err := newTestClient(t, r).AddMACFilterEntry(context.Background(), "aa:bb:cc:dd:ee:ff", "D2")
	if !errors.Is(err, ErrMACFilterDuplicate) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("setMacFilterCfg")); n != 0 {
		t.Fatalf("posts = %d", n)
	}
}

func TestMACFilterAddFull(t *testing.T) {
	r := tendatest.New(t)
	var rows []string
	for i := range macFilterMax {
		rows = append(rows, `{"devName":"d","devMac":"AA:BB:CC:DD:EE:`+strconv.Itoa(10+i)+`"}`)
	}
	r.Reply("getMacFilterCfg", 200, `{"localhostIP":"1","localhostName":"h","localhostMac":"m","macFilterType":"black","blackList":[`+strings.Join(rows, ",")+`],"whiteList":[],"onlineList":[]}`)
	err := newTestClient(t, r).AddMACFilterEntry(context.Background(), "AA:BB:CC:DD:EE:FF", "x")
	if !errors.Is(err, ErrMACFilterFull) || len(r.CallsTo("setMacFilterCfg")) != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestMACFilterRemove(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getMacFilterCfg", 200, `{"localhostIP":"1","localhostName":"h","localhostMac":"m","macFilterType":"black","blackList":[{"devName":"D","devMac":"AA:BB:CC:DD:EE:FF"}],"whiteList":[],"onlineList":[]}`)
	c := newTestClient(t, r)
	if err := c.RemoveMACFilterEntry(context.Background(), "aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "setMacFilterCfg").Form.Get("deviceList"); got != "" {
		t.Fatalf("deviceList = %q, want empty", got)
	}
	if err := c.RemoveMACFilterEntry(context.Background(), "11:22:33:44:55:66"); !errors.Is(err, ErrMACFilterNoEntry) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("setMacFilterCfg")); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
}

func TestMACFilterSetMode(t *testing.T) {
	r := tendatest.New(t)
	r.Allow("setMacFilterCfg")
	r.Reply("getMacFilterCfg", 200, `{"localhostIP":"1","localhostName":"h","localhostMac":"m","macFilterType":"black","blackList":[],"whiteList":[{"devName":"D","devMac":"AA:BB:CC:DD:EE:FF"}],"onlineList":[]}`)
	c := newTestClient(t, r, WithConfirm(allowAll))
	if err := c.SetMACFilterMode(context.Background(), "white"); err != nil {
		t.Fatal(err)
	}
	call := r.LastCall(t, "setMacFilterCfg")
	if call.Form.Get("macFilterType") != "white" || call.Form.Get("deviceList") != "D\rAA:BB:CC:DD:EE:FF" {
		t.Fatalf("call = %+v", call)
	}
	if err := c.SetMACFilterMode(context.Background(), "bogus"); err == nil {
		t.Fatal("want error")
	}
}

func TestMACFilterModeWhiteIsHazard(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	err := c.SetMACFilterMode(context.Background(), "white")
	var he *HazardError
	if !errors.As(err, &he) {
		t.Fatalf("err = %v, want *HazardError", err)
	}
	if n := len(r.CallsTo("setMacFilterCfg")); n != 0 {
		t.Fatalf("posts = %d, want 0", n)
	}
}
