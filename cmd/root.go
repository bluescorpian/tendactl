// Package cmd is tendactl's command line. Each feature lives in its own file
// and adds its commands from init() through register.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const defaultHost = "192.168.0.1"

// app holds the parsed global flags and the process seams. Execute sets the
// seams to real values; tests set them to fakes.
type app struct {
	getenv     func(string) string
	httpClient *http.Client // nil: a default client using --timeout
	sessionDir string       // "": tenda.DefaultSessionPath
	isTerminal func() bool
	readPass   func() (string, error)

	host    string
	output  string
	yes     bool
	timeout time.Duration

	stderr io.Writer
	c      *tenda.Client
}

// ctor builds one command. Feature files register theirs from init().
type ctor func(a *app) *cobra.Command

// registry maps a parent ("" for the root, or a group name) to its commands.
var registry = map[string][]ctor{}

// groups are the parents a feature may register under besides the root.
var groups = []struct{ use, short string }{
	{"wifi", "WiFi settings"},
	{"system", "System settings"},
	{"vpn", "PPTP/L2TP VPN server and client"},
}

func register(parent string, f ctor) {
	registry[parent] = append(registry[parent], f)
}

func newRootCmd(a *app) *cobra.Command {
	root := &cobra.Command{
		Use:   "tendactl",
		Short: "Manage a Tenda AC10 router over its web API",
		Long: `tendactl manages a Tenda AC10 router through the same web API its admin UI uses.

The password comes from TENDA_PASSWORD, or a prompt when run in a terminal.
The session is cached per host under $XDG_RUNTIME_DIR, so later commands
reuse it without a password until the router expires it.

Requests that can cut connectivity or destroy configuration (reboot, LAN
changes, turning WiFi off, ...) are refused unless --yes is given.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			a.stderr = cmd.ErrOrStderr()
			switch a.output {
			case "table", "json":
				return nil
			}
			return fmt.Errorf("invalid --output %q: want table or json", a.output)
		},
	}
	host := a.getenv("TENDA_HOST")
	if host == "" {
		host = defaultHost
	}
	pf := root.PersistentFlags()
	pf.StringVar(&a.host, "host", host, "router address, host[:port] (env TENDA_HOST)")
	pf.StringVarP(&a.output, "output", "o", "table", "output format: table or json")
	pf.BoolVarP(&a.yes, "yes", "y", false, "allow hazardous requests (reboot, WiFi off, LAN changes, ...)")
	pf.DurationVar(&a.timeout, "timeout", 15*time.Second, "HTTP timeout per request")

	parents := map[string]*cobra.Command{"": root}
	for _, g := range groups {
		gc := group(g.use, g.short)
		root.AddCommand(gc)
		parents[g.use] = gc
	}
	for parent, ctors := range registry {
		p, ok := parents[parent]
		if !ok {
			panic(fmt.Sprintf("tendactl: command registered under unknown parent %q", parent))
		}
		for _, f := range ctors {
			p.AddCommand(f(a))
		}
	}
	return root
}

// group returns a parent command that prints help when run bare and rejects
// unknown subcommands (plain cobra would print help and exit 0 on a typo).
func group(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:         use,
		Short:       short,
		Args:        cobra.ArbitraryArgs,
		Annotations: map[string]string{"group": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
		},
	}
}

// Execute runs tendactl and returns the process exit code.
func Execute() int {
	a := &app{
		getenv:     os.Getenv,
		isTerminal: func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		readPass: func() (string, error) {
			b, err := term.ReadPassword(int(os.Stdin.Fd()))
			return string(b), err
		},
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root := newRootCmd(a)
	return exitCode(root.ExecuteContext(ctx), root.ErrOrStderr())
}

// exitCode reports err on stderr and maps it to the process exit code.
func exitCode(err error, stderr io.Writer) int {
	if err == nil {
		return 0
	}
	var he *tenda.HazardError
	if errors.As(err, &he) && errors.Is(he.Err, errNeedYes) {
		fmt.Fprintf(stderr, "tendactl: refusing %s: %s; rerun with --yes\n", he.Hazard.Endpoint, he.Hazard.Reason)
		return 1
	}
	fmt.Fprintf(stderr, "tendactl: %v\n", err)
	return 1
}
