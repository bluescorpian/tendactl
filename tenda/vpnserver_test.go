package tenda

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestVPNServer(t *testing.T) {
	r := tendatest.New(t)
	s, err := newTestClient(t, r).VPNServer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled || s.StartIP != "10.0.0.100" || s.EndIP != "10.0.0.200" || s.MPPE || s.MPPEBits != 128 {
		t.Fatalf("s = %+v", s)
	}
}

func TestVPNServerSet(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	s := VPNServer{Enabled: true, StartIP: "10.0.0.100", EndIP: "10.0.0.200", MPPE: true, MPPEBits: 40}
	if err := c.SetVPNServer(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"serverEn": {"1"}, "startIp": {"10.0.0.100"}, "endIp": {"10.0.0.200"}, "mppe": {"1"}, "mppeOp": {"40"}}.Encode()
	if got := r.LastCall(t, "SetPptpServerCfg").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestVPNServerUsersEmpty(t *testing.T) {
	r := tendatest.New(t)
	users, err := newTestClient(t, r).VPNServerUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if users == nil || len(users) != 0 {
		t.Fatalf("users = %+v", users)
	}
}

const vpnServerFixtureWithUsers = `[{"serverEn":"1","wanid":"1","mppe":"0","mppeOp":"128","startIp":"10.0.0.100","endIp":"10.0.0.200","lanIp":"192.168.0.1","lanMask":"255.255.255.0","guestIp":"192.168.10.1","guestMask":"255.255.255.0","serverIp":"","vlan2Ip":"","vlan2Mask":"","wanIp":"203.0.113.10","wanMask":"255.255.255.255","pptpSvrIp":"10.0.0.1","pptpSvrMask":"255.255.255.0"},{"userName":"alice","password":"s3cret","enable":"1","connsta":"1"},{"userName":"bob","password":"hunter2","enable":"0","connsta":"0"}]`

func TestVPNServerUsers(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerFixtureWithUsers)
	users, err := newTestClient(t, r).VPNServerUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("users = %+v", users)
	}
	if users[0] != (VPNServerUser{Name: "alice", Password: "s3cret", Enabled: true, Connected: true}) {
		t.Fatalf("users[0] = %+v", users[0])
	}
	if users[1] != (VPNServerUser{Name: "bob", Password: "hunter2", Enabled: false, Connected: false}) {
		t.Fatalf("users[1] = %+v", users[1])
	}
}

func TestVPNServerAddUser(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerFixtureWithUsers)
	c := newTestClient(t, r)
	if err := c.AddVPNServerUser(context.Background(), "carol", "p@ss;word"); err != nil {
		t.Fatal(err)
	}
	list := "alice;s3cret;1;0;;;~bob;hunter2;0;0;;;~carol;" + url.QueryEscape("p@ss;word") + ";1;0;;;"
	want := url.Values{"list": {list}}.Encode()
	if got := r.LastCall(t, "setPptpUserList").RawBody; got != want {
		t.Fatalf("body = %q\nwant   %q", got, want)
	}
}

// TestVPNServerAddUserSeparator: url-encoding a password does not escape
// '~' (Go's url.QueryEscape leaves it unreserved), so a literal '~' is
// still refused by TildeSemi.Encode rather than silently corrupting the
// list, the same fallback nat and macfilter use for a separator in a cell.
func TestVPNServerAddUserSeparator(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerFixtureWithUsers)
	c := newTestClient(t, r)
	var sepErr *SeparatorError
	if err := c.AddVPNServerUser(context.Background(), "carol", "p~ord"); !errors.As(err, &sepErr) {
		t.Fatalf("err = %v, want *SeparatorError", err)
	}
	if n := len(r.CallsTo("setPptpUserList")); n != 0 {
		t.Fatalf("posts = %d", n)
	}
}

func TestVPNServerAddUserDuplicate(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerFixtureWithUsers)
	c := newTestClient(t, r)
	if err := c.AddVPNServerUser(context.Background(), "alice", "x"); !errors.Is(err, ErrVPNServerUserExists) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("setPptpUserList")); n != 0 {
		t.Fatalf("posts = %d", n)
	}
}

func TestVPNServerAddUserFull(t *testing.T) {
	r := tendatest.New(t)
	var rows []string
	for i := range vpnServerMaxUsers {
		rows = append(rows, `{"userName":"u`+strconv.Itoa(i)+`","password":"p","enable":"1","connsta":"0"}`)
	}
	r.Reply("GetPptpServerCfg", 200, `[{"serverEn":"0","wanid":"1","mppe":"0","mppeOp":"128","startIp":"10.0.0.100","endIp":"10.0.0.200","lanIp":"192.168.0.1","lanMask":"255.255.255.0","guestIp":"192.168.10.1","guestMask":"255.255.255.0","serverIp":"","vlan2Ip":"","vlan2Mask":"","wanIp":"203.0.113.10","wanMask":"255.255.255.255","pptpSvrIp":"10.0.0.1","pptpSvrMask":"255.255.255.0"},`+strings.Join(rows, ",")+`]`)
	c := newTestClient(t, r)
	if err := c.AddVPNServerUser(context.Background(), "newuser", "p"); !errors.Is(err, ErrVPNServerUsersFull) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("setPptpUserList")); n != 0 {
		t.Fatalf("posts = %d", n)
	}
}

func TestVPNServerRemoveUser(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerFixtureWithUsers)
	c := newTestClient(t, r)
	ctx := context.Background()
	if err := c.RemoveVPNServerUser(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"list": {"bob;hunter2;0;0;;;"}}.Encode()
	if got := r.LastCall(t, "setPptpUserList").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if err := c.RemoveVPNServerUser(ctx, "nobody"); !errors.Is(err, ErrVPNServerNoUser) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("setPptpUserList")); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
}

func TestVPNServerSetUserEnabled(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetPptpServerCfg", 200, vpnServerFixtureWithUsers)
	c := newTestClient(t, r)
	ctx := context.Background()
	if err := c.SetVPNServerUserEnabled(ctx, "bob", true); err != nil {
		t.Fatal(err)
	}
	want := url.Values{"list": {"alice;s3cret;1;0;;;~bob;hunter2;1;0;;;"}}.Encode()
	if got := r.LastCall(t, "setPptpUserList").RawBody; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if err := c.SetVPNServerUserEnabled(ctx, "nobody", false); !errors.Is(err, ErrVPNServerNoUser) {
		t.Fatalf("err = %v", err)
	}
}

func TestVPNServerOnlineUsersEmpty(t *testing.T) {
	r := tendatest.New(t)
	users, err := newTestClient(t, r).VPNServerOnlineUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if users == nil || len(users) != 0 {
		t.Fatalf("users = %+v", users)
	}
}

func TestVPNServerOnlineUsers(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getPptpOnlineClient", 200, `{"clientList":[{"username":"alice","dialIP":"203.0.113.5","clientIP":"10.0.0.101","onlineTime":42}]}`)
	users, err := newTestClient(t, r).VPNServerOnlineUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0] != (VPNServerOnlineUser{Name: "alice", DialIP: "203.0.113.5", ClientIP: "10.0.0.101", OnlineMinutes: 42}) {
		t.Fatalf("users = %+v", users)
	}
}
