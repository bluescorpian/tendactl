package tenda

import (
	"net/url"
	"testing"
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
