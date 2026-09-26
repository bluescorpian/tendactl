package tenda

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSetErrCode(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*tendatest.Router)
		code  int  // want *APIError with this code; 0 = nil error
		dec   bool // want *DecodeError
	}{
		{"number 0", func(r *tendatest.Router) { r.ErrCode("SetDMZCfg", 0) }, 0, false},
		{"string 0", func(r *tendatest.Router) { r.ErrCode("SetDMZCfg", "0") }, 0, false},
		{"number 2", func(r *tendatest.Router) { r.ErrCode("SetDMZCfg", 2) }, 2, false},
		{"string 2", func(r *tendatest.Router) { r.ErrCode("SetDMZCfg", "2") }, 2, false},
		{"no errCode", func(r *tendatest.Router) { r.Reply("SetDMZCfg", 200, `{}`) }, 0, true},
		{"not JSON", func(r *tendatest.Router) { r.Reply("SetDMZCfg", 200, `ok`) }, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tendatest.New(t)
			tt.setup(r)
			err := newTestClient(t, r).set(context.Background(), "SetDMZCfg", url.Values{"dmzEn": {"0"}})
			var ae *APIError
			var de *DecodeError
			switch {
			case tt.dec:
				if !errors.As(err, &de) {
					t.Fatalf("err = %v, want *DecodeError", err)
				}
			case tt.code != 0:
				if !errors.As(err, &ae) || ae.Code != tt.code || ae.Endpoint != "SetDMZCfg" || ae.Field != "errCode" {
					t.Fatalf("err = %v, want *APIError code %d", err, tt.code)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPostIgnoresBody(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("parentControlEn", 200, "")
	if err := newTestClient(t, r).post(context.Background(), "parentControlEn", url.Values{"isControled": {"1"}}); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "parentControlEn").Form.Get("isControled"); got != "1" {
		t.Fatalf("form = %q", got)
	}
}

func TestCloud(t *testing.T) {
	ctx := context.Background()
	r := tendatest.New(t)
	c := newTestClient(t, r)

	var basic struct {
		ErrCode Number `json:"err_code"`
		Enable  Flag   `json:"enable"`
		SN      string `json:"sn"`
	}
	if err := c.cloud(ctx, "manage", "querybasic", nil, &basic); err != nil {
		t.Fatal(err)
	}
	if basic.SN != "000000000" || basic.Enable {
		t.Fatalf("basic = %+v", basic)
	}
	var ver struct {
		VerInfo struct {
			ErrCode  Number `json:"err_code"`
			RespType int    `json:"resp_type"`
		} `json:"ver_info"`
	}
	if err := c.cloud(ctx, "olupgrade", "queryversion", nil, &ver); err != nil || ver.VerInfo.RespType != 1 {
		t.Fatalf("queryversion = %+v, %v", ver, err)
	}

	r.ErrCode("cloudv2?module=manage&opt=queryaccount", 15)
	var ae *APIError
	if err := c.cloud(ctx, "manage", "queryaccount", nil, nil); !errors.As(err, &ae) || ae.Code != 15 || ae.Field != "err_code" {
		t.Fatalf("top-level err_code: err = %v", err)
	}
	r.Reply("cloudv2?module=olupgrade&opt=queryversion", 200, `{"ver_info":{"err_code":4,"resp_type":1}}`)
	if err := c.cloud(ctx, "olupgrade", "queryversion", nil, nil); !errors.As(err, &ae) || ae.Code != 4 {
		t.Fatalf("nested err_code: err = %v", err)
	}

	r.Reply("cloudv2?module=manage&opt=setbasic", 200, `"{\"err_code\":0}"`)
	if err := c.cloud(ctx, "manage", "setbasic", url.Values{"enable": {"0"}}, nil); err != nil {
		t.Fatalf("string-wrapped reply: %v", err)
	}
	call := r.LastCall(t, "cloudv2?module=manage&opt=setbasic")
	if call.Method != http.MethodPost || call.RawBody != "enable=0" {
		t.Fatalf("setbasic call = %+v", call)
	}
	r.Reply("cloudv2?module=manage&opt=setbasic", 200, `"{\"err_code\":16}"`)
	if err := c.cloud(ctx, "manage", "setbasic", url.Values{"enable": {"0"}}, nil); !errors.As(err, &ae) || ae.Code != 16 {
		t.Fatalf("string-wrapped err_code: err = %v", err)
	}
}

func TestSubmit(t *testing.T) {
	ctx := context.Background()
	r := tendatest.New(t)
	r.Allow("SysToolChangePwd")
	c := newTestClient(t, r, WithConfirm(allowAll))

	resp, err := c.submit(ctx, http.MethodPost, "SysToolChangePwd", url.Values{"SYSOPS": {"a"}})
	if err != nil || resp.Status != http.StatusFound {
		t.Fatalf("submit = %+v, %v", resp, err)
	}

	r.Handle("SysToolChangePwd", func(tendatest.Call) tendatest.Response {
		return tendatest.Response{Status: http.StatusFound, Header: http.Header{"Location": {"/system_password.html?1"}}}
	})
	var ae *APIError
	if _, err := c.submit(ctx, http.MethodPost, "SysToolChangePwd", url.Values{"SYSOPS": {"a"}}); !errors.As(err, &ae) || ae.Code != 1 || ae.Field != "redirect" {
		t.Fatalf("err = %v, want redirect *APIError", err)
	}

	r.Reply("SysToolChangePwd", http.StatusOK, "<!DOCTYPE html><html>saved</html>")
	if _, err := c.submit(ctx, http.MethodPost, "SysToolChangePwd", url.Values{"SYSOPS": {"a"}}); err != nil {
		t.Fatalf("HTML page after a native form must be success: %v", err)
	}
	if r.Logins() != 1 {
		t.Fatalf("logins = %d, want 1", r.Logins())
	}
}

func TestDownload(t *testing.T) {
	ctx := context.Background()
	r := tendatest.New(t)
	c := newTestClient(t, r)
	var buf bytes.Buffer
	n, err := c.download(ctx, "cgi-bin/DownloadCfg/RouterCfm.cfg", &buf)
	if err != nil || n != int64(len(tendatest.ConfigBytes)) || !bytes.Equal(buf.Bytes(), tendatest.ConfigBytes) {
		t.Fatalf("download = %d %q %v", n, buf.Bytes(), err)
	}

	r.ExpireDoctype()
	buf.Reset()
	if _, err := c.download(ctx, "cgi-bin/DownloadLog/syslog.tar", &buf); err != nil || !bytes.Equal(buf.Bytes(), tendatest.SyslogBytes) {
		t.Fatalf("download after DOCTYPE expiry = %q, %v", buf.Bytes(), err)
	}
	if r.Logins() != 2 {
		t.Fatalf("logins = %d, want 2 (DOCTYPE body counts as expiry)", r.Logins())
	}
}

func TestUnsupported(t *testing.T) {
	r := tendatest.New(t)
	r.Unsupported("GetUsbCfg")
	err := newTestClient(t, r).get(context.Background(), "GetUsbCfg", nil, &struct{}{})
	var ue *UnsupportedError
	if !errors.Is(err, ErrUnsupported) || !errors.As(err, &ue) || ue.Form != "GetUsbCfg" {
		t.Fatalf("err = %v", err)
	}
	if r.Logins() != 1 {
		t.Fatalf("logins = %d, want 1 (no re-login)", r.Logins())
	}
}

func TestRaw(t *testing.T) {
	ctx := context.Background()
	r := tendatest.New(t)
	c := newTestClient(t, r)

	resp, err := c.Raw(ctx, http.MethodGet, "goform/GetDMZCfg", nil, nil)
	if err != nil || !resp.IsJSON() {
		t.Fatalf("Raw = %+v, %v", resp, err)
	}

	r.ErrCode("SetDMZCfg", 1)
	resp, err = c.Raw(ctx, http.MethodPost, "SetDMZCfg", nil, url.Values{"dmzEn": {"1"}})
	var ae *APIError
	if !errors.As(err, &ae) || ae.Code != 1 || resp == nil || string(resp.Body) != `{"errCode":1}` {
		t.Fatalf("Raw = %+v, %v; want body and *APIError", resp, err)
	}

	resp, err = c.Raw(ctx, http.MethodGet, "cgi-bin/DownloadCfg/RouterCfm.cfg", nil, nil)
	if err != nil || resp.IsJSON() || !bytes.Equal(resp.Body, tendatest.ConfigBytes) {
		t.Fatalf("Raw download = %+v, %v", resp, err)
	}

	resp, err = c.Raw(ctx, http.MethodGet, "cloudv2?module=wansta&opt=query", nil, nil)
	if err != nil || string(bytes.TrimSpace(resp.Body)) != `{"wan_sta":1}` {
		t.Fatalf("Raw cloudv2 = %+v, %v", resp, err)
	}
}
