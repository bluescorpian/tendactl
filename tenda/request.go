package tenda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// RawResponse is an unparsed router reply, as returned by Raw.
type RawResponse struct {
	Status int
	Header http.Header
	Body   []byte

	endpoint string
}

// IsJSON reports whether the body is valid JSON.
func (r *RawResponse) IsJSON() bool {
	b := bytes.TrimSpace(r.Body)
	return len(b) > 0 && (b[0] == '{' || b[0] == '[' || b[0] == '"') && json.Valid(b)
}

// nativeForm lists the endpoints the UI submits as plain HTML forms. They
// answer with a redirect or a page, never JSON (docs: Conventions -> Result).
var nativeForm = []string{"SysToolReboot", "SysToolRestoreSet", "SysToolChangePwd", "cgi-bin/UploadCfg", "cgi-bin/upgrade"}

func isNativeForm(key string) bool {
	for _, n := range nativeForm {
		if strings.EqualFold(n, key) {
			return true
		}
	}
	return false
}

// get fetches a JSON endpoint and decodes it into v.
func (c *Client) get(ctx context.Context, endpoint string, query url.Values, v any) error {
	resp, err := c.do(ctx, call{method: http.MethodGet, endpoint: endpoint, query: query, kind: kindJSON})
	if err != nil {
		return err
	}
	return c.decodeJSON(resp.endpoint, resp.Body, v)
}

// set posts form and requires a JSON reply whose errCode is 0.
func (c *Client) set(ctx context.Context, endpoint string, form url.Values) error {
	return c.setWith(ctx, endpoint, form, nil)
}

// setWith is set with the previous form values, for the hazard rules that
// only fire when a value changes (WifiBasicSet, openSchedWifi, PowerSaveSet).
func (c *Client) setWith(ctx context.Context, endpoint string, form, before url.Values) error {
	if form == nil {
		form = url.Values{}
	}
	resp, err := c.do(ctx, call{method: http.MethodPost, endpoint: endpoint, form: form, before: before, kind: kindJSON})
	if err != nil {
		return err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(resp.Body, &m); err != nil {
		return &DecodeError{Endpoint: resp.endpoint, Body: resp.Body, Err: err}
	}
	raw, ok := m["errCode"]
	if !ok {
		return &DecodeError{Endpoint: resp.endpoint, Body: resp.Body, Err: errors.New("reply has no errCode")}
	}
	var code Number
	if err := json.Unmarshal(raw, &code); err != nil {
		return &DecodeError{Endpoint: resp.endpoint, Body: resp.Body, Err: err}
	}
	if code != 0 {
		return &APIError{Endpoint: resp.endpoint, Field: "errCode", Code: int(code), Body: resp.Body}
	}
	return nil
}

// post sends form to an endpoint whose reply the UI ignores
// (parentControlEn, fast_setting_internet_set). Only transport and session
// errors are reported.
func (c *Client) post(ctx context.Context, endpoint string, form url.Values) error {
	if form == nil {
		form = url.Values{}
	}
	_, err := c.do(ctx, call{method: http.MethodPost, endpoint: endpoint, form: form, kind: kindJSON})
	return err
}

// submit sends a native form (SysToolReboot action=0, ...). Any non-login
// 2xx or 3xx is success, except a redirect whose target ends in "?<n>" with
// n != 0, which is how these forms report failure.
func (c *Client) submit(ctx context.Context, method, endpoint string, form url.Values) (*RawResponse, error) {
	cl := call{method: method, endpoint: endpoint, kind: kindSubmit}
	if method == http.MethodGet {
		cl.query = form
	} else {
		cl.form = form
		if cl.form == nil {
			cl.form = url.Values{}
		}
	}
	resp, err := c.do(ctx, cl)
	if err != nil {
		return nil, err
	}
	if err := redirectError(resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func redirectError(resp *RawResponse) error {
	if resp.Status < 300 || resp.Status >= 400 {
		return nil
	}
	loc := resp.Header.Get("Location")
	i := strings.LastIndexByte(loc, '?')
	if i < 0 {
		return nil
	}
	n, err := strconv.Atoi(loc[i+1:])
	if err != nil || n == 0 {
		return nil
	}
	return &APIError{Endpoint: resp.endpoint, Field: "redirect", Code: n, Body: resp.Body}
}

// cloud calls goform/cloudv2?module=&opt=, with GET when form is nil and
// POST otherwise. err_code is checked at the top level or inside the single
// nested object (ver_info, up_info), then the body is decoded into v.
func (c *Client) cloud(ctx context.Context, module, opt string, form url.Values, v any) error {
	cl := call{method: http.MethodGet, endpoint: "cloudv2", query: url.Values{"module": {module}, "opt": {opt}}, kind: kindJSON}
	if form != nil {
		cl.method, cl.form = http.MethodPost, form
	}
	resp, err := c.do(ctx, cl)
	if err != nil {
		return err
	}
	body := unwrapJSONString(resp.Body)
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return &DecodeError{Endpoint: resp.endpoint, Body: resp.Body, Err: err}
	}
	if field, code, found, err := findCode(m); err != nil {
		return &DecodeError{Endpoint: resp.endpoint, Body: resp.Body, Err: err}
	} else if found && code != 0 {
		return &APIError{Endpoint: resp.endpoint, Field: field, Code: code, Body: resp.Body}
	}
	if v == nil {
		return nil
	}
	return c.decodeJSON(resp.endpoint, body, v)
}

// download streams a cgi-bin file (DownloadCfg/RouterCfm.cfg,
// DownloadLog/syslog.tar) to w.
func (c *Client) download(ctx context.Context, endpoint string, w io.Writer) (int64, error) {
	resp, err := c.do(ctx, call{method: http.MethodGet, endpoint: endpoint, kind: kindDownload})
	if err != nil {
		return 0, err
	}
	n, err := w.Write(resp.Body)
	return int64(n), err
}

// Raw sends any request, for the `api` command. The request kind follows the
// endpoint: cgi-bin/Download* is a download, a native form is a submit, the
// rest are JSON. Hazards are classified without previous values, so the
// conditional rules are conservative. A JSON reply with a non-zero errCode or
// err_code returns both the response and an *APIError; a non-JSON reply is
// not an error.
func (c *Client) Raw(ctx context.Context, method, endpoint string, query, form url.Values) (*RawResponse, error) {
	_, key, _, err := normalizeEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	k := kindJSON
	switch {
	case strings.HasPrefix(strings.ToLower(key), "cgi-bin/download"):
		k = kindDownload
	case isNativeForm(key):
		k = kindSubmit
	}
	if method != http.MethodGet && form == nil {
		form = url.Values{}
	}
	resp, err := c.do(ctx, call{method: method, endpoint: endpoint, query: query, form: form, kind: k})
	if err != nil {
		return nil, err
	}
	if err := redirectError(resp); err != nil {
		return resp, err
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(unwrapJSONString(resp.Body), &m) != nil {
		return resp, nil
	}
	if field, code, found, err := findCode(m); err == nil && found && code != 0 {
		return resp, &APIError{Endpoint: resp.endpoint, Field: field, Code: code, Body: resp.Body}
	}
	return resp, nil
}

// unwrapJSONString unwraps a body that is a JSON string holding JSON, as
// cloudv2 setbasic returns (the UI calls $.parseJSON on it).
func unwrapJSONString(b []byte) []byte {
	t := bytes.TrimSpace(b)
	if len(t) == 0 || t[0] != '"' {
		return b
	}
	var s string
	if json.Unmarshal(t, &s) != nil {
		return b
	}
	return []byte(s)
}

// findCode looks for errCode or err_code at the top level, then for err_code
// in a nested object.
func findCode(m map[string]json.RawMessage) (field string, code int, found bool, err error) {
	for _, f := range []string{"errCode", "err_code"} {
		if raw, ok := m[f]; ok {
			var n Number
			if err := json.Unmarshal(raw, &n); err != nil {
				return f, 0, true, err
			}
			return f, int(n), true, nil
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var inner map[string]json.RawMessage
		if json.Unmarshal(m[k], &inner) != nil {
			continue
		}
		if raw, ok := inner["err_code"]; ok {
			var n Number
			if err := json.Unmarshal(raw, &n); err != nil {
				return "err_code", 0, true, err
			}
			return "err_code", int(n), true, nil
		}
	}
	return "", 0, false, nil
}
