package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

// errNeedYes is the confirm hook's refusal; Execute turns it into a hint.
var errNeedYes = errors.New("rerun with --yes")

// errNothingToChange is what a bare `set` returns.
var errNothingToChange = errors.New("nothing to change: pass at least one setting flag")

// client builds the router client once per process.
func (a *app) client() (*tenda.Client, error) {
	if a.c != nil {
		return a.c, nil
	}
	hc := a.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: a.timeout}
	}
	opts := []tenda.Option{
		tenda.WithPassword(a.password),
		tenda.WithConfirm(a.confirm),
		tenda.WithHTTPClient(hc),
	}
	if a.sessionDir != "" {
		opts = append(opts, tenda.WithSessionFile(filepath.Join(a.sessionDir, "session")))
	}
	c, err := tenda.New(a.host, opts...)
	if err != nil {
		return nil, err
	}
	a.c = c
	return c, nil
}

// password supplies the router password when a login actually happens:
// TENDA_PASSWORD, else a prompt on a terminal.
func (a *app) password(context.Context) (string, error) {
	if pw := a.getenv("TENDA_PASSWORD"); pw != "" {
		return pw, nil
	}
	if !a.isTerminal() {
		return "", errors.New("no router password: set TENDA_PASSWORD or run in a terminal")
	}
	w := a.stderr
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintf(w, "Router password for %s: ", a.host)
	pw, err := a.readPass()
	fmt.Fprintln(w)
	return pw, err
}

// confirm allows hazardous requests only with --yes; there is no interactive
// prompt, so scripts and people follow one rule.
func (a *app) confirm(context.Context, tenda.Hazard) error {
	if a.yes {
		return nil
	}
	return errNeedYes
}

// update is the read-modify-write helper for setters: get the current value,
// apply mutate, set it, and report done.
func update[T any](a *app, cmd *cobra.Command,
	get func(context.Context) (T, error), set func(context.Context, T) error,
	mutate func(*T) error, done string) error {
	ctx := cmd.Context()
	v, err := get(ctx)
	if err != nil {
		return err
	}
	if err := mutate(&v); err != nil {
		return err
	}
	if err := set(ctx, v); err != nil {
		return err
	}
	return a.done(cmd, "%s", done)
}

// changed reports whether any of the named flags was given. There is no
// bare, zero-argument form: cmd.LocalNonPersistentFlags() builds a fresh
// *pflag.FlagSet on every call via AddFlag, which shares the underlying
// *pflag.Flag pointers but never runs Parse on itself, so its own Visit
// always reports nothing regardless of what was actually parsed. Every
// caller already names its flags, which reads cmd.Flags().Changed instead
// and doesn't hit that trap.
func changed(cmd *cobra.Command, names ...string) bool {
	for _, n := range names {
		if cmd.Flags().Changed(n) {
			return true
		}
	}
	return false
}
