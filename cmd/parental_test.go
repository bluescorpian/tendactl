package cmd

import (
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestParentalList(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "parental").Stdout
	golden(t, "parental", text)
	if got := mustRun(t, r, "parental", "list").Stdout; got != text {
		t.Fatalf("list differs from bare:\n%s", got)
	}
	golden(t, "parental_json", mustRun(t, r, "parental", "-o", "json").Stdout)
}

func TestParentalShowNotConfigured(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	got := mustRun(t, r, "parental", "show", "02:00:00:00:00:08").Stdout
	if !strings.Contains(got, "No parental control rule configured") {
		t.Fatalf("got %q", got)
	}
	if got := mustRun(t, r, "parental", "show", "02:00:00:00:00:08", "-o", "json").Stdout; !strings.Contains(got, `"configured": false`) {
		t.Fatalf("got %q", got)
	}
}

func TestParentalShowConfigured(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("GetParentControlInfo", 200, `{"enable":1,"mac":"02:00:00:00:00:08","url_enable":1,"urls":"example,video","time":"19:00-21:00","day":"1,1,1,1,1,1,1","limit_type":0}`)
	golden(t, "parental_show", mustRun(t, r, "parental", "show", "02:00:00:00:00:08").Stdout)
}

func TestParentalSetNothingToChange(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "parental", "set", "02:00:00:00:00:08")
	if res.Code != 1 || !strings.Contains(res.Stderr, "nothing to change") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestParentalSetNewRule(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	mustRun(t, r, "parental", "set", "02:00:00:00:00:08", "--allow", "08:00-18:00", "--days", "sat,sun", "--url-filter", "on", "--limit-mode", "whitelist", "--urls", "example,video")
	got := r.LastCall(t, "saveParentControlInfo").Form
	if got.Get("time") != "08:00-18:00" || got.Get("day") != "1,0,0,0,0,0,1" || got.Get("limit_type") != "1" || got.Get("urls") != "example,video" || got.Get("enable") != "1" {
		t.Fatalf("form = %v", got)
	}
}

func TestParentalSetNewRuleRejectsBadMAC(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, mac, wantErr string
	}{
		{"all-zero", "00:00:00:00:00:00", "cannot be 00:00:00:00:00:00"},
		{"multicast bit set", "01:00:00:00:00:00", "must be an even number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := tendatest.New(t)
			// checkParentData() in fw/js/parental_control.js refuses to
			// submit either shape when adding a brand-new device.
			res := runCLI(t, r, "parental", "set", tt.mac, "--allow", "08:00-18:00")
			if res.Code != 1 || !strings.Contains(res.Stderr, tt.wantErr) {
				t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
			}
			if n := len(r.CallsTo("saveParentControlInfo")); n != 0 {
				t.Fatalf("posts = %d, want 0", n)
			}
		})
	}
}

func TestParentalSetExistingRuleAllowsAnyMAC(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	// The add-new-device MAC checks don't apply once a rule already exists
	// for that device (checkParentData() only runs them for
	// G_current_operate=="1"): editing must still work.
	r.Reply("GetParentControlInfo", 200, `{"enable":1,"mac":"01:00:00:00:00:00","url_enable":0,"urls":"","time":"19:00-21:00","day":"1,1,1,1,1,1,1","limit_type":0}`)
	mustRun(t, r, "parental", "set", "01:00:00:00:00:00", "--allow", "08:00-18:00")
	if got := r.LastCall(t, "saveParentControlInfo").Form.Get("time"); got != "08:00-18:00" {
		t.Fatalf("time = %q", got)
	}
}

func TestParentalSetInvalidTime(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "parental", "set", "02:00:00:00:00:08", "--allow", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, "invalid --allow") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestParentalRm(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	r.Reply("getParentalRuleList", 200, `[{"devName":"Kids Tablet","mac":"02:00:00:00:00:08","enable":"1"}]`)
	mustRun(t, r, "parental", "rm", "02:00:00:00:00:08")
	if got := r.LastCall(t, "delParentalRule").Form.Get("mac"); got != "02:00:00:00:00:08" {
		t.Fatalf("mac = %q", got)
	}

	res := runCLI(t, r, "parental", "rm", "aa:bb:cc:dd:ee:ff")
	if res.Code != 1 || !strings.Contains(res.Stderr, "no parental control rule") {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestParentalEnableDisable(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := mustRun(t, r, "parental", "enable", "02:00:00:00:00:08")
	if got := r.LastCall(t, "parentControlEn").Form; got.Get("mac") != "02:00:00:00:00:08" || got.Get("isControled") != "1" {
		t.Fatalf("form = %v", got)
	}
	if res.Stdout != "Blocked 02:00:00:00:00:08\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}

	mustRun(t, r, "parental", "disable", "02:00:00:00:00:08")
	if got := r.LastCall(t, "parentControlEn").Form.Get("isControled"); got != "0" {
		t.Fatalf("isControled = %q", got)
	}
}
