package tenda

import (
	"net/url"
	"slices"
	"strings"
)

// Hazard describes why a request needs confirmation.
type Hazard struct {
	Endpoint string
	Reason   string // user-facing
}

// Request is what Classify judges. Endpoint is the normalised key
// ("SysToolReboot", "cgi-bin/upgrade"). Before holds the values the setting
// had before this change; nil means unknown.
type Request struct {
	Method, Endpoint    string
	Query, Form, Before url.Values
}

type hazardRule struct {
	endpoint string
	reason   string
	when     func(Request) bool // nil = always
}

// hazards is docs/router-api.md "Never call casually", the only copy in code.
// Rules match the endpoint case-insensitively and regardless of method: a
// plain GET of SysToolReboot reboots the router too.
var hazards = []hazardRule{
	{endpoint: "SysToolReboot", reason: "reboots the router; WiFi and WAN drop for about a minute"},
	{endpoint: "setApModeCfg", reason: "changes the working mode and reboots the router"},
	{endpoint: "WifiExtraSet", reason: "changes the working mode (Wireless Repeating) and reboots the router"},
	{endpoint: "SetIPTVCfg", reason: "changes IPTV/VLAN settings; the UI reboots the router afterwards"},
	{endpoint: "SysToolRestoreSet", reason: "restores factory settings, erasing all configuration"},
	{endpoint: "cgi-bin/UploadCfg", reason: "replaces the whole configuration and reboots the router"},
	{endpoint: "cgi-bin/upgrade", reason: "flashes firmware and reboots the router"},
	{endpoint: "cloudv2", reason: "the first queryupgrade call starts an online firmware upgrade",
		when: func(r Request) bool {
			return strings.EqualFold(reqValue(r, "module"), "olupgrade") && strings.EqualFold(reqValue(r, "opt"), "queryupgrade")
		}},
	{endpoint: "WanParameterSetting", reason: "rewrites the WAN connection; internet access can drop"},
	{endpoint: "fast_setting_internet_set", reason: "rewrites the WAN connection; internet access can drop"},
	{endpoint: "AdvSetMacMtuWan", reason: "changes WAN MAC/MTU/speed; internet access can drop"},
	{endpoint: "AdvSetLanip", reason: "changes LAN addressing; clients (and this tool) can lose the router"},
	{endpoint: "setMacFilterCfg", reason: "whitelist mode blocks every device not on the list",
		when: func(r Request) bool { return strings.EqualFold(reqValue(r, "macFilterType"), "white") }},
	{endpoint: "SysToolChangePwd", reason: "changes the admin password"},
	{endpoint: "WifiBasicSet", reason: "turns a WiFi band off; wireless clients disconnect",
		when: func(r Request) bool {
			for _, k := range []string{"wrlEn", "wrlEn_5g"} {
				if reqValue(r, k) == "0" && (r.Before == nil || r.Before.Get(k) != "0") {
					return true
				}
			}
			return false
		}},
	{endpoint: "openSchedWifi", reason: "enables the WiFi schedule, which turns WiFi off during its window",
		when: func(r Request) bool { return reqValue(r, "schedWifiEnable") == "1" && !reqUnchanged(r) }},
	{endpoint: "PowerSaveSet", reason: "enables Sleeping Mode, which turns WiFi off during its window",
		when: func(r Request) bool { return reqValue(r, "powerSavingEn") == "1" && !reqUnchanged(r) }},
}

// Classify reports whether r is on the hazard list.
func Classify(r Request) (Hazard, bool) {
	for _, h := range hazards {
		if !strings.EqualFold(h.endpoint, r.Endpoint) {
			continue
		}
		if h.when == nil || h.when(r) {
			return Hazard{Endpoint: h.endpoint, Reason: h.reason}, true
		}
	}
	return Hazard{}, false
}

// reqValue reads k from the form, falling back to the query.
func reqValue(r Request, k string) string {
	if v, ok := r.Form[k]; ok && len(v) > 0 {
		return v[0]
	}
	return r.Query.Get(k)
}

// reqUnchanged reports whether Before is known and equal to Form on every key
// Form sends: an identical resubmit.
func reqUnchanged(r Request) bool {
	if r.Before == nil {
		return false
	}
	for k, v := range r.Form {
		if !slices.Equal(v, r.Before[k]) {
			return false
		}
	}
	return true
}
