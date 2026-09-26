package tenda

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestParentalDevices(t *testing.T) {
	r := tendatest.New(t)
	d, err := newTestClient(t, r).ParentalDevices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 13 {
		t.Fatalf("devices = %d", len(d))
	}
	first := d[0]
	if first.MAC != "02:00:00:00:00:04" || first.Name != "Device-3" || first.OnlineTime != 210 || !first.Online || first.RuleSet || first.Blocked {
		t.Fatalf("first = %+v", first)
	}
}

func TestParentalRuleNotConfigured(t *testing.T) {
	r := tendatest.New(t)
	rule, ok, err := newTestClient(t, r).ParentalRule(context.Background(), "02:00:00:00:00:08")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("ok = true, rule = %+v", rule)
	}
}

func TestParentalRuleConfigured(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetParentControlInfo", 200, `{"enable":1,"mac":"02:00:00:00:00:08","url_enable":1,"urls":"example,video","time":"19:00-21:00","day":"1,1,1,1,1,1,1","limit_type":0}`)
	rule, ok, err := newTestClient(t, r).ParentalRule(context.Background(), "02:00:00:00:00:08")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !rule.Enabled || rule.AllowedWindow != "19:00-21:00" || !rule.EveryDay || !rule.URLFilterOn || rule.LimitType != "blacklist" {
		t.Fatalf("rule = %+v", rule)
	}
	if strings.Join(rule.URLs, ",") != "example,video" {
		t.Fatalf("urls = %v", rule.URLs)
	}
	if got := r.LastCall(t, "GetParentControlInfo").Query.Get("mac"); got != "02:00:00:00:00:08" {
		t.Fatalf("mac query = %q", got)
	}
}

func TestParentalRuleSpecifiedDays(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetParentControlInfo", 200, `{"enable":1,"mac":"x","url_enable":0,"urls":"","time":"08:00-18:00","day":"0,1,0,0,0,0,1","limit_type":1}`)
	rule, ok, err := newTestClient(t, r).ParentalRule(context.Background(), "02:00:00:00:00:08")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || rule.EveryDay || strings.Join(rule.Days, ",") != "mon,sat" || rule.LimitType != "whitelist" || rule.URLs != nil {
		t.Fatalf("rule = %+v", rule)
	}
}

func TestSetParentalRule(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	rule := ParentalRule{
		MAC: "02:00:00:00:00:08", Name: "Kids Tablet", Enabled: true,
		AllowedWindow: "19:00-21:00", EveryDay: false, Days: []string{"sat", "sun"},
		URLFilterOn: true, LimitType: "blacklist", URLs: []string{"example", "video"},
	}
	if err := c.SetParentalRule(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
	got := r.LastCall(t, "saveParentControlInfo").Form
	if got.Get("deviceId") != "02:00:00:00:00:08" || got.Get("deviceName") != "Kids Tablet" || got.Get("enable") != "1" {
		t.Fatalf("form = %v", got)
	}
	if got.Get("time") != "19:00-21:00" || got.Get("day") != "1,0,0,0,0,0,1" || got.Get("limit_type") != "0" || got.Get("urls") != "example,video" {
		t.Fatalf("form = %v", got)
	}
}

func TestSetParentalRuleDisabled(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetParentalRule(context.Background(), ParentalRule{MAC: "02:00:00:00:00:08", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	got := r.LastCall(t, "saveParentControlInfo").Form
	if got.Get("enable") != "0" || got.Get("time") != "" || got.Get("deviceName") != "" {
		t.Fatalf("form = %v", got)
	}
}

func TestSetParentalRuleValidation(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	base := ParentalRule{MAC: "02:00:00:00:00:08", Enabled: true}

	tooMany := base
	for i := range 11 {
		tooMany.URLs = append(tooMany.URLs, "kw"+strconv.Itoa(i))
	}
	if err := c.SetParentalRule(context.Background(), tooMany); !errors.Is(err, ErrParentalTooManyURLs) {
		t.Fatalf("err = %v", err)
	}

	bad := base
	bad.URLs = []string{"UPPER"}
	if err := c.SetParentalRule(context.Background(), bad); !errors.Is(err, ErrParentalInvalidURL) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("saveParentControlInfo")); n != 0 {
		t.Fatalf("posts = %d, want 0", n)
	}
}

func TestSetParentalRuleFull(t *testing.T) {
	r := tendatest.New(t)
	r.ErrCode("saveParentControlInfo", 1)
	err := newTestClient(t, r).SetParentalRule(context.Background(), ParentalRule{MAC: "02:00:00:00:00:08", Enabled: true})
	if !errors.Is(err, ErrParentalFull) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetParentalBlocked(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	if err := c.SetParentalBlocked(context.Background(), "02:00:00:00:00:08", true); err != nil {
		t.Fatal(err)
	}
	got := r.LastCall(t, "parentControlEn").Form
	if got.Get("mac") != "02:00:00:00:00:08" || got.Get("isControled") != "1" {
		t.Fatalf("form = %v", got)
	}
	if err := c.SetParentalBlocked(context.Background(), "02:00:00:00:00:08", false); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "parentControlEn").Form.Get("isControled"); got != "0" {
		t.Fatalf("isControled = %q", got)
	}
}

func TestRemoveParentalRule(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getParentalRuleList", 200, `[{"devName":"Kids Tablet","mac":"02:00:00:00:00:08","enable":"1"}]`)
	c := newTestClient(t, r)
	ctx := context.Background()
	if err := c.RemoveParentalRule(ctx, "02:00:00:00:00:08"); err != nil {
		t.Fatal(err)
	}
	if got := r.LastCall(t, "delParentalRule").Form.Get("mac"); got != "02:00:00:00:00:08" {
		t.Fatalf("mac = %q", got)
	}
	if err := c.RemoveParentalRule(ctx, "aa:bb:cc:dd:ee:ff"); !errors.Is(err, ErrParentalNoRule) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("delParentalRule")); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
}

func TestRemoveParentalRuleEmptyList(t *testing.T) {
	r := tendatest.New(t)
	err := newTestClient(t, r).RemoveParentalRule(context.Background(), "02:00:00:00:00:08")
	if !errors.Is(err, ErrParentalNoRule) {
		t.Fatalf("err = %v", err)
	}
	if n := len(r.CallsTo("delParentalRule")); n != 0 {
		t.Fatalf("posts = %d, want 0", n)
	}
}
