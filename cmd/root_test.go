package cmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
	"github.com/spf13/cobra"
)

// leafNames is the verb and action vocabulary a leaf command may use
// (design 3.4, plus "get" for `api get`).
var leafNames = []string{
	"list", "show", "set", "add", "rm", "enable", "disable", "get",
	"rename", "block", "unblock", "blocked", "hide", "unhide", "start", "mode",
	"reboot", "log", "backup", "users", "online", "status", "maintenance", "time", "remote",
}

// TestHelpWalk runs --help on every command and lints the tree: help makes
// no request, siblings have distinct names, every command has a Short, and
// every leaf is named from the vocabulary.
func TestHelpWalk(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	root := newRootCmd(&app{getenv: func(string) string { return "" }})
	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		if c.Short == "" {
			t.Errorf("%q has no Short", strings.Join(path, " "))
		}
		seen := map[string]bool{}
		for _, sub := range c.Commands() {
			for _, n := range append([]string{sub.Name()}, sub.Aliases...) {
				if seen[n] {
					t.Errorf("duplicate command %q under %q", n, c.CommandPath())
				}
				seen[n] = true
			}
		}
		if !c.HasSubCommands() && c.Annotations["group"] == "" && len(path) > 0 && !slices.Contains(leafNames, c.Name()) {
			t.Errorf("leaf %q is not in the verb/action vocabulary", c.CommandPath())
		}
		res := runCLI(t, r, append(slices.Clone(path), "--help")...)
		if res.Code != 0 || !strings.Contains(res.Stdout, "Usage:") {
			t.Errorf("%q --help: exit %d, stderr %q", strings.Join(path, " "), res.Code, res.Stderr)
		}
		for _, sub := range c.Commands() {
			walk(sub, append(slices.Clone(path), sub.Name()))
		}
	}
	walk(root, nil)
	if n := len(r.Calls()) + r.Logins(); n != 0 {
		t.Fatalf("--help made %d requests", n)
	}
}

func TestHostPrecedence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		env  map[string]string
		args []string
		want string
	}{
		{nil, nil, "192.168.0.1"},
		{map[string]string{"TENDA_HOST": "env.lan"}, nil, "env.lan"},
		{map[string]string{"TENDA_HOST": "env.lan"}, []string{"--host", "flag.lan"}, "flag.lan"},
	}
	for _, tt := range tests {
		res := runCLIWith(t, nil, cliEnv{Env: tt.env}, append(tt.args, "--help")...)
		if res.Code != 0 || res.app.host != tt.want {
			t.Fatalf("env %v args %v: host = %q (exit %d), want %q", tt.env, tt.args, res.app.host, res.Code, tt.want)
		}
	}
}

func TestOutputFlagValidated(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "-o", "bogus", "status")
	if res.Code != 1 || !strings.Contains(res.Stderr, "--output") || len(r.Calls()) != 0 {
		t.Fatalf("exit %d, stderr %q, calls %d", res.Code, res.Stderr, len(r.Calls()))
	}
}

func TestGroupRejectsUnknownSubcommand(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLI(t, r, "wifi", "bogus")
	if res.Code != 1 || !strings.Contains(res.Stderr, `unknown command "bogus"`) {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	res = runCLI(t, r, "wifi")
	if res.Code != 0 || !strings.Contains(res.Stdout, "Usage:") {
		t.Fatalf("bare group: exit %d, stdout %q", res.Code, res.Stdout)
	}
	if res := runCLI(t, r, "nat", "bogus"); res.Code != 1 || !strings.Contains(res.Stderr, "unknown command") {
		t.Fatalf("nat bogus: exit %d, stderr %q", res.Code, res.Stderr)
	}
}

func TestPasswordFromEnvSkipsPrompt(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	n := 0
	res := runCLIWith(t, r, cliEnv{
		Env: map[string]string{"TENDA_HOST": r.Host(), "TENDA_PASSWORD": tendatest.Password},
		TTY: true, PromptCount: &n,
	}, "status")
	if res.Code != 0 || n != 0 {
		t.Fatalf("exit %d, prompts %d, stderr %q", res.Code, n, res.Stderr)
	}
}

func TestPasswordPromptOncePerProcess(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	// Expire the session between nat add's GET and its POST, forcing a
	// re-login inside one process.
	r.Handle("GetVirtualServerCfg", func(tendatest.Call) tendatest.Response {
		r.Expire()
		return tendatest.JSON(`{"lanIp":"192.168.0.1","lanMask":"255.255.255.0","virtualList":[]}`)
	})
	n := 0
	res := runCLIWith(t, r, cliEnv{
		Env: map[string]string{"TENDA_HOST": r.Host()},
		TTY: true, Password: tendatest.Password, PromptCount: &n,
	}, "nat", "add", "192.168.0.9", "80")
	if res.Code != 0 {
		t.Fatalf("exit %d, stderr %q", res.Code, res.Stderr)
	}
	if n != 1 || r.Logins() != 2 {
		t.Fatalf("prompts = %d, logins = %d; want 1, 2", n, r.Logins())
	}
	if !strings.Contains(res.Stderr, "Router password for "+r.Host()) {
		t.Fatalf("stderr = %q", res.Stderr)
	}
}

func TestPasswordMissingWithoutTerminal(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLIWith(t, r, cliEnv{Env: map[string]string{"TENDA_HOST": r.Host()}}, "status")
	if res.Code != 1 || !strings.Contains(res.Stderr, "TENDA_PASSWORD") || r.Logins() != 0 {
		t.Fatalf("exit %d, stderr %q, logins %d", res.Code, res.Stderr, r.Logins())
	}
}

func TestBadPasswordMessage(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	res := runCLIWith(t, r, cliEnv{Env: map[string]string{"TENDA_HOST": r.Host(), "TENDA_PASSWORD": "wrong"}}, "status")
	if res.Code != 1 || res.Stderr != "tendactl: incorrect router password\n" || res.Stdout != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", res.Code, res.Stdout, res.Stderr)
	}
}
