package tenda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// VPNServer is "PPTP Server" (VPN tab): server-wide settings from
// GetPptpServerCfg / SetPptpServerCfg. The user account table is a separate
// endpoint (VPNServerUsers).
type VPNServer struct {
	Enabled  bool   `json:"enabled"`
	StartIP  string `json:"startIp"`
	EndIP    string `json:"endIp"`
	MPPE     bool   `json:"mppe"`
	MPPEBits int    `json:"mppeBits"` // 40 or 128
}

// VPNServerUser is one PPTP server user account (a setPptpUserList row).
type VPNServerUser struct {
	Name      string `json:"name"`
	Password  string `json:"password"`
	Enabled   bool   `json:"enabled"`
	Connected bool   `json:"connected"` // read-only: connsta
}

// VPNServerOnlineUser is one currently-connected PPTP server client
// (getPptpOnlineClient), this router acting as the PPTP server.
type VPNServerOnlineUser struct {
	Name          string `json:"name"`
	DialIP        string `json:"dialIp"`
	ClientIP      string `json:"clientIp"`
	OnlineMinutes int    `json:"onlineMinutes"`
}

// vpnServerWire is GetPptpServerCfg's element [0]. Elements [1..], when
// present, are vpnServerUserWire rows.
type vpnServerWire struct {
	ServerEn    Flag   `json:"serverEn"`
	Wanid       string `json:"wanid"`
	Mppe        Flag   `json:"mppe"`
	MppeOp      string `json:"mppeOp"`
	StartIp     string `json:"startIp"`
	EndIp       string `json:"endIp"`
	LanIp       string `json:"lanIp"`
	LanMask     string `json:"lanMask"`
	GuestIp     string `json:"guestIp"`
	GuestMask   string `json:"guestMask"`
	ServerIp    string `json:"serverIp"`
	Vlan2Ip     string `json:"vlan2Ip"`
	Vlan2Mask   string `json:"vlan2Mask"`
	WanIp       string `json:"wanIp"`
	WanMask     string `json:"wanMask"`
	PptpSvrIp   string `json:"pptpSvrIp"`
	PptpSvrMask string `json:"pptpSvrMask"`
}

type vpnServerUserWire struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	Enable   Flag   `json:"enable"`
	Connsta  Flag   `json:"connsta"`
}

type vpnServerOnlineWire struct {
	ClientList []vpnServerOnlineUserWire `json:"clientList"`
}

type vpnServerOnlineUserWire struct {
	Username   string `json:"username"`
	DialIP     string `json:"dialIP"`
	ClientIP   string `json:"clientIP"`
	OnlineTime int    `json:"onlineTime"` // minutes
}

// vpnServerMaxUsers is the UI's client-side limit on PPTP server accounts.
const vpnServerMaxUsers = 8

var (
	ErrVPNServerUserExists = errors.New("a PPTP server user with that name already exists")
	ErrVPNServerNoUser     = errors.New("no PPTP server user with that name")
	ErrVPNServerUsersFull  = fmt.Errorf("the router allows at most %d PPTP server users", vpnServerMaxUsers)
)

// vpnServerRaw reads GetPptpServerCfg, whose JSON array holds the
// server-wide config at [0] and one row per configured user after it.
func (c *Client) vpnServerRaw(ctx context.Context) ([]json.RawMessage, error) {
	var raw []json.RawMessage
	if err := c.get(ctx, "GetPptpServerCfg", nil, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, &DecodeError{Endpoint: "GetPptpServerCfg", Body: []byte("[]"), Err: errors.New("empty array")}
	}
	return raw, nil
}

// VPNServer reads the server-wide PPTP server settings.
func (c *Client) VPNServer(ctx context.Context) (VPNServer, error) {
	raw, err := c.vpnServerRaw(ctx)
	if err != nil {
		return VPNServer{}, err
	}
	var w vpnServerWire
	if err := c.decodeJSON("GetPptpServerCfg", raw[0], &w); err != nil {
		return VPNServer{}, err
	}
	bits, err := strconv.Atoi(w.MppeOp)
	if err != nil {
		return VPNServer{}, &DecodeError{Endpoint: "GetPptpServerCfg", Body: raw[0], Err: fmt.Errorf("mppeOp: %w", err)}
	}
	return VPNServer{
		Enabled: bool(w.ServerEn), StartIP: w.StartIp, EndIP: w.EndIp,
		MPPE: bool(w.Mppe), MPPEBits: bits,
	}, nil
}

// SetVPNServer sends the full UI form (SetPptpServerCfg). This never touches
// the user account list, which SetPptpServerCfg's own doc section notes it
// deliberately excludes; use AddVPNServerUser/RemoveVPNServerUser/
// SetVPNServerUserEnabled for that.
func (c *Client) SetVPNServer(ctx context.Context, s VPNServer) error {
	form := url.Values{
		"serverEn": {flag(s.Enabled)},
		"startIp":  {s.StartIP},
		"endIp":    {s.EndIP},
		"mppe":     {flag(s.MPPE)},
		"mppeOp":   {strconv.Itoa(s.MPPEBits)},
	}
	return c.set(ctx, "SetPptpServerCfg", form)
}

// VPNServerUsers reads the configured PPTP server user accounts, the rows
// GetPptpServerCfg appends after the server-wide config.
func (c *Client) VPNServerUsers(ctx context.Context) ([]VPNServerUser, error) {
	raw, err := c.vpnServerRaw(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]VPNServerUser, 0, len(raw)-1)
	for _, r := range raw[1:] {
		var w vpnServerUserWire
		if err := c.decodeJSON("GetPptpServerCfg", r, &w); err != nil {
			return nil, err
		}
		out = append(out, VPNServerUser{Name: w.UserName, Password: w.Password, Enabled: bool(w.Enable), Connected: bool(w.Connsta)})
	}
	return out, nil
}

// vpnServerEncodeURIComponent escapes s the way the UI's JS does before packing a
// field into setPptpUserList's list value (doc: `username`/`password` are
// `encodeURIComponent`-escaped). This must not be url.QueryEscape directly:
// QueryEscape turns a space into '+' and percent-escapes !'()* as well,
// where JS's encodeURIComponent percent-escapes a space (%20) and leaves
// !'()* literal. The space matters beyond cosmetics — after this string is
// split back out of the list by the firmware's own per-field decode (not
// the outer x-www-form-urlencoded decode, which already happened), that
// decode is a plain percent-decode matching decodeURIComponent, which
// treats '+' literally rather than as encoded space; a stray '+' from
// QueryEscape would therefore end up stored verbatim in the credential
// instead of decoding back to a space.
func vpnServerEncodeURIComponent(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// setVPNServerUsers replaces the whole user table (setPptpUserList). Name
// and Password are url-encoded per the doc, which prevents a ';' or '~' in
// either from being mistaken for a separator; a literal '~' is still
// refused by TildeSemi.Encode, since url-encoding does not escape it. The
// client always sends netEn=0 and empty serverIp/serverMask/remark, since
// those sub-fields are UI-disabled.
func (c *Client) setVPNServerUsers(ctx context.Context, users []VPNServerUser) error {
	rows := make([][]string, len(users))
	for i, u := range users {
		rows[i] = []string{vpnServerEncodeURIComponent(u.Name), vpnServerEncodeURIComponent(u.Password), flag(u.Enabled), "0", "", "", ""}
	}
	list, err := TildeSemi.Encode(rows)
	if err != nil {
		return err
	}
	return c.set(ctx, "setPptpUserList", url.Values{"list": {list}})
}

// AddVPNServerUser adds a PPTP server user account, after the UI's checks:
// a unique user name and at most 8 accounts. Nothing is posted when a check
// fails. The new account starts enabled, matching the UI's "+New" row.
func (c *Client) AddVPNServerUser(ctx context.Context, name, password string) error {
	users, err := c.VPNServerUsers(ctx)
	if err != nil {
		return err
	}
	for _, u := range users {
		if u.Name == name {
			return fmt.Errorf("%w (%s)", ErrVPNServerUserExists, name)
		}
	}
	if len(users) >= vpnServerMaxUsers {
		return ErrVPNServerUsersFull
	}
	return c.setVPNServerUsers(ctx, append(users, VPNServerUser{Name: name, Password: password, Enabled: true}))
}

// RemoveVPNServerUser deletes the user account named name. It returns
// ErrVPNServerNoUser, having posted nothing, when there is none.
func (c *Client) RemoveVPNServerUser(ctx context.Context, name string) error {
	users, err := c.VPNServerUsers(ctx)
	if err != nil {
		return err
	}
	kept := make([]VPNServerUser, 0, len(users))
	for _, u := range users {
		if u.Name != name {
			kept = append(kept, u)
		}
	}
	if len(kept) == len(users) {
		return fmt.Errorf("%w (%s)", ErrVPNServerNoUser, name)
	}
	return c.setVPNServerUsers(ctx, kept)
}

// SetVPNServerUserEnabled turns a user account on or off in place, keeping
// its password and position. It returns ErrVPNServerNoUser, having posted
// nothing, when there is no account with that name.
func (c *Client) SetVPNServerUserEnabled(ctx context.Context, name string, enabled bool) error {
	users, err := c.VPNServerUsers(ctx)
	if err != nil {
		return err
	}
	found := false
	for i, u := range users {
		if u.Name == name {
			users[i].Enabled = enabled
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%w (%s)", ErrVPNServerNoUser, name)
	}
	return c.setVPNServerUsers(ctx, users)
}

// VPNServerOnlineUsers reads the PPTP clients currently connected to this
// router's PPTP server (getPptpOnlineClient). It is always non-nil.
func (c *Client) VPNServerOnlineUsers(ctx context.Context) ([]VPNServerOnlineUser, error) {
	var w vpnServerOnlineWire
	if err := c.get(ctx, "getPptpOnlineClient", nil, &w); err != nil {
		return nil, err
	}
	out := make([]VPNServerOnlineUser, 0, len(w.ClientList))
	for _, e := range w.ClientList {
		out = append(out, VPNServerOnlineUser{Name: e.Username, DialIP: e.DialIP, ClientIP: e.ClientIP, OnlineMinutes: e.OnlineTime})
	}
	return out, nil
}
