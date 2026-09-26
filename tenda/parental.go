package tenda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// parentalDays is the wire day order, Sunday first (docs/router-api.md UI
// names: Parental Control).
var parentalDays = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// ParentalDevice is one row of GetParentCtrlList: every LAN device the
// router knows, with its parental-control rule status.
type ParentalDevice struct {
	MAC        string `json:"mac"`
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Online     bool   `json:"online"`
	OnlineTime int    `json:"onlineTimeSeconds"`
	RuleSet    bool   `json:"ruleSet"` // isSet: a rule exists for this device
	Blocked    bool   `json:"blocked"` // isControled: the rule is currently blocking it
}

// ParentalRule is one device's rule (GetParentControlInfo /
// saveParentControlInfo). Enabled is the rule's own on/off switch in the
// edit popup; the separate isControled toggle (SetParentalBlocked) blocks a
// device immediately regardless of its schedule.
type ParentalRule struct {
	MAC           string   `json:"mac"`
	Name          string   `json:"name"` // sent as deviceName when saving; not returned by GetParentControlInfo
	Enabled       bool     `json:"enabled"`
	AllowedWindow string   `json:"allowedWindow"` // "HH:MM-HH:MM"; internet is allowed only in this window
	Days          []string `json:"days"`          // e.g. ["sun","sat"]; meaningful only when not every day
	EveryDay      bool     `json:"everyDay"`
	URLFilterOn   bool     `json:"urlFilterOn"`
	LimitType     string   `json:"limitType"` // "blacklist" or "whitelist"
	URLs          []string `json:"urls"`
}

type parentalDeviceWire struct {
	DevType     string `json:"devType"`
	OnlineTime  int    `json:"onlineTime"`
	DeviceId    string `json:"deviceId"`
	Ip          string `json:"ip"`
	DevName     string `json:"devName"`
	IsControled Flag   `json:"isControled"`
	IsSet       Flag   `json:"isSet"`
	Line        string `json:"line"` // "0" offline, "1" online
}

type parentalRuleWire struct {
	Enable    Number `json:"enable"`
	Mac       string `json:"mac"`
	UrlEnable Number `json:"url_enable"`
	Urls      string `json:"urls"`
	Time      string `json:"time"`
	Day       string `json:"day"`
	LimitType Number `json:"limit_type"`
}

type parentalRuleListEntryWire struct {
	DevName string `json:"devName"`
	Mac     string `json:"mac"`
	Enable  Flag   `json:"enable"`
}

// parentalMaxRules is the UI's client-side cap, shared with the quick
// blacklist's wording.
const parentalMaxRules = 30

// parentalURLRe is the doc's keyword shape: 2-31 chars of [-.a-z0-9].
var parentalURLRe = regexp.MustCompile(`^[-.a-z0-9]{2,31}$`)

var (
	ErrParentalNoRule      = errors.New("no parental control rule for that MAC address")
	ErrParentalFull        = fmt.Errorf("the router allows at most %d parental control rules", parentalMaxRules)
	ErrParentalTooManyURLs = errors.New("at most 10 blocked-website keywords are allowed")
	ErrParentalInvalidURL  = errors.New("invalid keyword: want 2-31 characters of a-z, 0-9, '.' or '-'")
)

// ParentalDevices reads GetParentCtrlList.
func (c *Client) ParentalDevices(ctx context.Context) ([]ParentalDevice, error) {
	var raw []parentalDeviceWire
	if err := c.get(ctx, "GetParentCtrlList", nil, &raw); err != nil {
		return nil, err
	}
	out := make([]ParentalDevice, 0, len(raw))
	for _, w := range raw {
		out = append(out, ParentalDevice{
			MAC: w.DeviceId, Name: w.DevName, IP: w.Ip, Online: w.Line == "1",
			OnlineTime: w.OnlineTime, RuleSet: bool(w.IsSet), Blocked: bool(w.IsControled),
		})
	}
	return out, nil
}

// ParentalRule reads GetParentControlInfo?mac= for mac. ok is false when the
// device has no rule (the router replies with just {"mac":"..."}).
func (c *Client) ParentalRule(ctx context.Context, mac string) (r ParentalRule, ok bool, err error) {
	m, err := ParseMAC(mac)
	if err != nil {
		return ParentalRule{}, false, err
	}
	var raw json.RawMessage
	if err := c.get(ctx, "GetParentControlInfo", url.Values{"mac": {strings.ToLower(m)}}, &raw); err != nil {
		return ParentalRule{}, false, err
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ParentalRule{}, false, &DecodeError{Endpoint: "GetParentControlInfo", Body: raw, Err: err}
	}
	if _, present := probe["enable"]; !present {
		return ParentalRule{}, false, nil
	}
	var w parentalRuleWire
	if err := c.decodeJSON("GetParentControlInfo", raw, &w); err != nil {
		return ParentalRule{}, false, err
	}
	days, everyDay := decodeParentalDays(w.Day)
	return ParentalRule{
		MAC: m, Enabled: w.Enable != 0, AllowedWindow: w.Time,
		Days: days, EveryDay: everyDay, URLFilterOn: w.UrlEnable != 0,
		LimitType: parentalLimitTypeLabel(w.LimitType), URLs: parentalSplitURLs(w.Urls),
	}, true, nil
}

func decodeParentalDays(raw string) (days []string, everyDay bool) {
	flags := strings.Split(raw, ",")
	everyDay = true
	for i, f := range flags {
		if i < len(parentalDays) && f == "1" {
			days = append(days, parentalDays[i])
		} else if i < len(parentalDays) {
			everyDay = false
		}
	}
	return days, everyDay
}

func encodeParentalDays(days []string, everyDay bool) string {
	flags := make([]string, len(parentalDays))
	for i := range flags {
		flags[i] = "0"
		if everyDay {
			flags[i] = "1"
		}
	}
	if !everyDay {
		for _, d := range days {
			for i, name := range parentalDays {
				if d == name {
					flags[i] = "1"
				}
			}
		}
	}
	return strings.Join(flags, ",")
}

func parentalLimitTypeLabel(n Number) string {
	if n == 1 {
		return "whitelist"
	}
	return "blacklist"
}

func parentalLimitTypeWire(s string) string {
	if s == "whitelist" {
		return "1"
	}
	return "0"
}

func parentalSplitURLs(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// SetParentalRule saves r (saveParentControlInfo). When r.Enabled is false,
// only deviceId and enable=0 are sent, per the doc; the schedule/URL-filter
// fields are otherwise sent in full, since this is the only way this CLI
// creates or edits a rule (deviceName is always included, unlike the UI's
// edit-existing-device path, which omits it and renames separately).
func (c *Client) SetParentalRule(ctx context.Context, r ParentalRule) error {
	m, err := ParseMAC(r.MAC)
	if err != nil {
		return err
	}
	form := url.Values{"deviceId": {strings.ToLower(m)}, "enable": {flag(r.Enabled)}}
	if r.Enabled {
		if len(r.URLs) > 10 {
			return ErrParentalTooManyURLs
		}
		for _, u := range r.URLs {
			if !parentalURLRe.MatchString(u) {
				return fmt.Errorf("%w: %q", ErrParentalInvalidURL, u)
			}
		}
		form.Set("deviceName", r.Name)
		form.Set("time", r.AllowedWindow)
		form.Set("url_enable", flag(r.URLFilterOn))
		form.Set("urls", strings.Join(r.URLs, ","))
		form.Set("day", encodeParentalDays(r.Days, r.EveryDay))
		form.Set("limit_type", parentalLimitTypeWire(r.LimitType))
	}
	err = c.set(ctx, "saveParentControlInfo", form)
	return mapCode(err, 1, ErrParentalFull)
}

// SetParentalBlocked toggles isControled for mac (parentControlEn),
// blocking or restoring its access immediately, independent of any rule's
// own schedule. The router's reply is not read by the UI, so only
// transport and session errors are reported.
func (c *Client) SetParentalBlocked(ctx context.Context, mac string, blocked bool) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	return c.post(ctx, "parentControlEn", url.Values{"mac": {strings.ToLower(m)}, "isControled": {flag(blocked)}})
}

// RemoveParentalRule deletes mac's rule (delParentalRule). It returns
// ErrParentalNoRule, having posted nothing, when getParentalRuleList shows
// no rule for mac.
func (c *Client) RemoveParentalRule(ctx context.Context, mac string) error {
	m, err := ParseMAC(mac)
	if err != nil {
		return err
	}
	var rows []parentalRuleListEntryWire
	if err := c.get(ctx, "getParentalRuleList", nil, &rows); err != nil {
		return err
	}
	found := false
	for _, row := range rows {
		if EqualMAC(row.Mac, m) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w (%s)", ErrParentalNoRule, m)
	}
	return c.set(ctx, "delParentalRule", url.Values{"mac": {strings.ToLower(m)}})
}
