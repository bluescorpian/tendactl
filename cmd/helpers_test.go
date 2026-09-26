package cmd

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

var updateGolden = flag.Bool("update", false, "rewrite cmd/testdata/*.golden")

// cliEnv is the process environment a test run sees.
type cliEnv struct {
	Env         map[string]string
	TTY         bool
	Password    string // returned by the password prompt
	PromptCount *int
}

type cliResult struct {
	Stdout, Stderr string
	Code           int
	app            *app
}

// runCLI runs tendactl against r with TENDA_HOST and TENDA_PASSWORD set and
// no terminal.
func runCLI(t *testing.T, r *tendatest.Router, args ...string) cliResult {
	t.Helper()
	return runCLIWith(t, r, cliEnv{Env: map[string]string{"TENDA_HOST": r.Host(), "TENDA_PASSWORD": tendatest.Password}}, args...)
}

// runCLIWith runs tendactl on a fresh app whose seams are fakes; no process
// state is touched, so tests may run in parallel. r may be nil for runs that
// must not reach a router.
func runCLIWith(t *testing.T, r *tendatest.Router, e cliEnv, args ...string) cliResult {
	t.Helper()
	a := &app{
		getenv:     func(k string) string { return e.Env[k] },
		sessionDir: t.TempDir(),
		isTerminal: func() bool { return e.TTY },
		readPass: func() (string, error) {
			if e.PromptCount != nil {
				*e.PromptCount++
			}
			return e.Password, nil
		},
	}
	if r != nil {
		a.httpClient = r.HTTPClient()
	} else {
		a.httpClient = tendatest.New(t).HTTPClient() // refuses every other host
	}
	var out, errOut bytes.Buffer
	root := newRootCmd(a)
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(""))
	code := exitCode(root.ExecuteContext(context.Background()), &errOut)
	return cliResult{Stdout: out.String(), Stderr: errOut.String(), Code: code, app: a}
}

// mustRun fails the test unless the run exits 0.
func mustRun(t *testing.T, r *tendatest.Router, args ...string) cliResult {
	t.Helper()
	res := runCLI(t, r, args...)
	if res.Code != 0 {
		t.Fatalf("tendactl %s: exit %d\nstderr: %s", strings.Join(args, " "), res.Code, res.Stderr)
	}
	return res
}

// golden compares got with cmd/testdata/<name>.golden; -update rewrites it.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./cmd -update)", err)
	}
	if got != string(want) {
		t.Fatalf("%s mismatch:\ngot\n%s\nwant\n%s", path, got, want)
	}
}
