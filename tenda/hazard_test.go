package tenda

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestClassify(t *testing.T) {
	v := func(kv ...string) url.Values {
		out := url.Values{}
		for i := 0; i < len(kv); i += 2 {
			out.Add(kv[i], kv[i+1])
		}
		return out
	}
	tests := []struct {
		name string
		req  Request
		want bool
	}{
		{"reboot post", Request{Method: "POST", Endpoint: "SysToolReboot", Form: v("action", "0")}, true},
		{"reboot plain get", Request{Method: "GET", Endpoint: "SysToolReboot"}, true},
		{"reboot lower case", Request{Method: "GET", Endpoint: "systoolreboot"}, true},
		{"ap mode", Request{Method: "POST", Endpoint: "setApModeCfg"}, true},
		{"wisp", Request{Method: "POST", Endpoint: "WifiExtraSet"}, true},
		{"iptv", Request{Method: "POST", Endpoint: "SetIPTVCfg"}, true},
		{"factory reset", Request{Method: "POST", Endpoint: "SysToolRestoreSet"}, true},
		{"upload cfg", Request{Method: "POST", Endpoint: "cgi-bin/UploadCfg"}, true},
		{"upgrade", Request{Method: "POST", Endpoint: "cgi-bin/upgrade"}, true},
		{"wan", Request{Method: "POST", Endpoint: "WanParameterSetting"}, true},
		{"fast setting wan", Request{Method: "POST", Endpoint: "fast_setting_internet_set"}, true},
		{"mac mtu", Request{Method: "POST", Endpoint: "AdvSetMacMtuWan"}, true},
		{"lan ip", Request{Method: "POST", Endpoint: "AdvSetLanip"}, true},
		{"change password", Request{Method: "POST", Endpoint: "SysToolChangePwd"}, true},
		{"queryupgrade", Request{Method: "GET", Endpoint: "cloudv2", Query: v("module", "olupgrade", "opt", "queryupgrade")}, true},
		{"queryversion", Request{Method: "GET", Endpoint: "cloudv2", Query: v("module", "olupgrade", "opt", "queryversion")}, false},
		{"macfilter white", Request{Method: "POST", Endpoint: "setMacFilterCfg", Form: v("macFilterType", "white", "deviceList", "")}, true},
		{"macfilter black", Request{Method: "POST", Endpoint: "setMacFilterCfg", Form: v("macFilterType", "black", "deviceList", "")}, false},
		{"wifi 5g off, was off", Request{Method: "POST", Endpoint: "WifiBasicSet", Form: v("wrlEn", "1", "wrlEn_5g", "0"), Before: v("wrlEn", "1", "wrlEn_5g", "0")}, false},
		{"wifi 5g off, was on", Request{Method: "POST", Endpoint: "WifiBasicSet", Form: v("wrlEn", "1", "wrlEn_5g", "0"), Before: v("wrlEn", "1", "wrlEn_5g", "1")}, true},
		{"wifi 5g off, before unknown", Request{Method: "POST", Endpoint: "WifiBasicSet", Form: v("wrlEn", "1", "wrlEn_5g", "0")}, true},
		{"wifi 2.4 off", Request{Method: "POST", Endpoint: "WifiBasicSet", Form: v("wrlEn", "0", "wrlEn_5g", "0"), Before: v("wrlEn", "1", "wrlEn_5g", "0")}, true},
		{"wifi both on", Request{Method: "POST", Endpoint: "WifiBasicSet", Form: v("wrlEn", "1", "wrlEn_5g", "1")}, false},
		{"sched enable new", Request{Method: "POST", Endpoint: "openSchedWifi", Form: v("schedWifiEnable", "1", "schedStartTime", "00:00"), Before: v("schedWifiEnable", "0", "schedStartTime", "00:00")}, true},
		{"sched enable unknown", Request{Method: "POST", Endpoint: "openSchedWifi", Form: v("schedWifiEnable", "1")}, true},
		{"sched resubmit", Request{Method: "POST", Endpoint: "openSchedWifi", Form: v("schedWifiEnable", "1", "schedStartTime", "00:00"), Before: v("schedWifiEnable", "1", "schedStartTime", "00:00")}, false},
		{"sched disable", Request{Method: "POST", Endpoint: "openSchedWifi", Form: v("schedWifiEnable", "0")}, false},
		{"sleep enable new", Request{Method: "POST", Endpoint: "PowerSaveSet", Form: v("powerSavingEn", "1"), Before: v("powerSavingEn", "0")}, true},
		{"sleep resubmit", Request{Method: "POST", Endpoint: "PowerSaveSet", Form: v("powerSavingEn", "1", "time", "00:00-07:00"), Before: v("powerSavingEn", "1", "time", "00:00-07:00")}, false},
		{"sleep disable", Request{Method: "POST", Endpoint: "PowerSaveSet", Form: v("powerSavingEn", "0")}, false},
		{"safe getter", Request{Method: "GET", Endpoint: "GetDMZCfg"}, false},
		{"safe setter", Request{Method: "POST", Endpoint: "SetDMZCfg", Form: v("dmzEn", "1")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, got := Classify(tt.req)
			if got != tt.want {
				t.Fatalf("Classify = %v (%+v), want %v", got, h, tt.want)
			}
			if got && (h.Endpoint == "" || h.Reason == "") {
				t.Fatalf("hazard without endpoint or reason: %+v", h)
			}
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct{ in, path, key, query string }{
		{"GetDMZCfg", "/goform/GetDMZCfg", "GetDMZCfg", ""},
		{"goform/GetDMZCfg", "/goform/GetDMZCfg", "GetDMZCfg", ""},
		{"/goform/GetDMZCfg", "/goform/GetDMZCfg", "GetDMZCfg", ""},
		{"goform/systoolreboot", "/goform/systoolreboot", "systoolreboot", ""},
		{"cgi-bin/DownloadCfg/RouterCfm.cfg", "/cgi-bin/DownloadCfg/RouterCfm.cfg", "cgi-bin/DownloadCfg/RouterCfm.cfg", ""},
		{"/cgi-bin/upgrade", "/cgi-bin/upgrade", "cgi-bin/upgrade", ""},
		{"cloudv2?module=m&opt=o", "/goform/cloudv2", "cloudv2", "module=m&opt=o"},
	}
	for _, tt := range tests {
		path, key, q, err := normalizeEndpoint(tt.in)
		if err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		if path != tt.path || key != tt.key || q.Encode() != tt.query {
			t.Fatalf("%s = %s %s %s", tt.in, path, key, q.Encode())
		}
	}
	for _, bad := range []string{"", "/", "goform/", "cgi-bin/", "login/Auth", "a/b"} {
		if _, _, _, err := normalizeEndpoint(bad); err == nil {
			t.Fatalf("%q: want error", bad)
		}
	}
}

// TestHazardGate sends every unconditional hazard through Raw: refused
// without confirmation (no request at all, not even a login), and sent
// exactly once when confirmed.
func TestHazardGate(t *testing.T) {
	for _, h := range hazards {
		if h.when != nil {
			continue
		}
		t.Run(h.endpoint, func(t *testing.T) {
			ctx := context.Background()
			r := tendatest.New(t)
			method := http.MethodPost
			if h.endpoint == "SysToolReboot" {
				method = http.MethodGet // a plain GET reboots too
			}
			_, err := newTestClient(t, r).Raw(ctx, method, h.endpoint, nil, url.Values{"action": {"0"}})
			var he *HazardError
			if !errors.As(err, &he) || !errors.Is(err, ErrNotConfirmed) {
				t.Fatalf("err = %v, want *HazardError", err)
			}
			if len(r.Calls()) != 0 || r.Logins() != 0 {
				t.Fatalf("calls = %d, logins = %d; want none", len(r.Calls()), r.Logins())
			}

			r.Allow(h.endpoint)
			if _, err := newTestClient(t, r, WithConfirm(allowAll)).Raw(ctx, method, h.endpoint, nil, url.Values{"action": {"0"}}); err != nil {
				t.Fatal(err)
			}
			if n := len(r.Calls()); n != 1 {
				t.Fatalf("calls = %d, want 1", n)
			}
		})
	}
}

func TestHazardConfirmError(t *testing.T) {
	r := tendatest.New(t)
	no := errors.New("no")
	c := newTestClient(t, r, WithConfirm(func(context.Context, Hazard) error { return no }))
	_, err := c.Raw(context.Background(), http.MethodGet, "cloudv2?module=olupgrade&opt=queryupgrade", nil, nil)
	var he *HazardError
	if !errors.As(err, &he) || !errors.Is(err, no) || he.Hazard.Endpoint != "cloudv2" {
		t.Fatalf("err = %v", err)
	}
	if len(r.Calls()) != 0 {
		t.Fatal("request sent")
	}
}

// TestHazardNotReplayedAfterDoctype: a DOCTYPE body does not prove the
// handler did not run, so a hazardous JSON request is never sent twice.
// (Native forms such as SysToolReboot answer HTML on success, so the DOCTYPE
// heuristic never applies to them.)
func TestHazardNotReplayedAfterDoctype(t *testing.T) {
	ctx := context.Background()
	r := tendatest.New(t)
	r.Allow("AdvSetLanip")
	c := newTestClient(t, r, WithConfirm(allowAll))
	if _, err := c.RouterStatus(ctx); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"lanIp": {"192.168.0.1"}}
	r.ExpireDoctype()
	if err := c.set(ctx, "AdvSetLanip", form); !errors.Is(err, ErrSessionLost) {
		t.Fatalf("err = %v, want ErrSessionLost", err)
	}
	if r.Expired() != 1 || r.Logins() != 1 {
		t.Fatalf("expired = %d, logins = %d; want 1, 1", r.Expired(), r.Logins())
	}

	// A 302 proves the handler did not run, so that one is replayed.
	r.Expire()
	if err := c.set(ctx, "AdvSetLanip", form); err != nil {
		t.Fatal(err)
	}
	if n := len(r.CallsTo("AdvSetLanip")); n != 1 {
		t.Fatalf("AdvSetLanip calls = %d, want 1", n)
	}
}
