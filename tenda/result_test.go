package tenda

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindCode(t *testing.T) {
	tests := []struct {
		body  string
		field string
		code  int
		found bool
	}{
		{`{"errCode":0}`, "errCode", 0, true},
		{`{"errCode":"0"}`, "errCode", 0, true},
		{`{"errCode":2}`, "errCode", 2, true},
		{`{"errCode":"2"}`, "errCode", 2, true},
		{`{"err_code":0,"enable":0,"sn":"1"}`, "err_code", 0, true},
		{`{"ver_info":{"err_code":0,"resp_type":1}}`, "err_code", 0, true},
		{`{"up_info":{"err_code":19}}`, "err_code", 19, true},
		{`{"wan_sta":1}`, "", 0, false},
	}
	for _, tt := range tests {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(unwrapJSONString([]byte(tt.body)), &m); err != nil {
			t.Fatal(err)
		}
		field, code, found, err := findCode(m)
		if err != nil || field != tt.field || code != tt.code || found != tt.found {
			t.Fatalf("%s = %q %d %v %v", tt.body, field, code, found, err)
		}
	}
}

func TestUnwrapJSONString(t *testing.T) {
	if got := string(unwrapJSONString([]byte(`"{\"err_code\":0}"`))); got != `{"err_code":0}` {
		t.Fatalf("got %s", got)
	}
	if got := string(unwrapJSONString([]byte(`{"a":1}`))); got != `{"a":1}` {
		t.Fatalf("got %s", got)
	}
}

func TestRedirectError(t *testing.T) {
	resp := func(status int, loc string) *RawResponse {
		return &RawResponse{Status: status, Header: http.Header{"Location": {loc}}, endpoint: "SysToolChangePwd"}
	}
	if err := redirectError(resp(302, "/system_password.html")); err != nil {
		t.Fatal(err)
	}
	if err := redirectError(resp(302, "/system_password.html?0")); err != nil {
		t.Fatal(err)
	}
	var ae *APIError
	if err := redirectError(resp(302, "/system_password.html?1")); !errors.As(err, &ae) || ae.Code != 1 || ae.Field != "redirect" {
		t.Fatalf("err = %v", err)
	}
	if err := redirectError(resp(200, "/x.html?1")); err != nil {
		t.Fatal(err)
	}
}

func TestMapCode(t *testing.T) {
	target := errors.New("target")
	api := &APIError{Endpoint: "SetDMZCfg", Field: "errCode", Code: 2}
	if err := mapCode(api, 2, target); !errors.Is(err, target) || !errors.As(err, new(*APIError)) {
		t.Fatalf("err = %v", err)
	}
	if err := mapCode(api, 3, target); errors.Is(err, target) {
		t.Fatalf("err = %v", err)
	}
	if mapCode(nil, 2, target) != nil {
		t.Fatal("nil must pass through")
	}
	if got := api.Error(); got != "SetDMZCfg: router returned errCode 2" {
		t.Fatalf("Error() = %q", got)
	}
}

// TestFixturesDecodeStrict proves the wire structs describe their fixtures
// completely.
func TestFixturesDecodeStrict(t *testing.T) {
	c := &Client{strict: true}
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	var rs routerStatusWire
	if err := c.decodeJSON("GetRouterStatus", read("GetRouterStatus.json"), &rs); err != nil {
		t.Fatal(err)
	}
	if rs.ClientNum != 13 || !bool(rs.Wl24gEn) || rs.OnlineUpgradeInfo.CurVersion != "V15.03.06.50_multi" {
		t.Fatalf("status = %+v", rs)
	}

	var raw []json.RawMessage
	if err := c.decodeJSON("getOnlineList", read("getOnlineList.json"), &raw); err != nil {
		t.Fatal(err)
	}
	var head onlineHeadWire
	if err := c.decodeJSON("getOnlineList", raw[0], &head); err != nil {
		t.Fatal(err)
	}
	if head.LocalhostName != "Device-11" || head.MacFilterType != "black" {
		t.Fatalf("head = %+v", head)
	}
	for _, r := range raw[1:] {
		var w onlineClientWire
		if err := c.decodeJSON("getOnlineList", r, &w); err != nil {
			t.Fatal(err)
		}
	}

	var nat natWire
	if err := c.decodeJSON("GetVirtualServerCfg", read("GetVirtualServerCfg.json"), &nat); err != nil {
		t.Fatal(err)
	}
	if len(nat.VirtualList) != 7 || nat.VirtualList[6].OutPort != "443" {
		t.Fatalf("nat = %+v", nat)
	}

	var bad struct {
		LanIp string `json:"lanIp"`
	}
	err := c.decodeJSON("GetDMZCfg", read("GetDMZCfg.json"), &bad)
	var de *DecodeError
	if !errors.As(err, &de) || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("strict decode of a partial struct: err = %v", err)
	}
}
