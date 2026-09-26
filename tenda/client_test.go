package tenda

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestNewHost(t *testing.T) {
	good := map[string]string{
		"192.168.0.1":            "192.168.0.1",
		"router.lan:8080":        "router.lan:8080",
		"http://192.168.0.1":     "192.168.0.1",
		"http://192.168.0.1:81/": "192.168.0.1:81",
		" 10.0.0.1 ":             "10.0.0.1",
	}
	for in, want := range good {
		c, err := New(in)
		if err != nil {
			t.Fatalf("New(%q): %v", in, err)
		}
		if c.Host() != want {
			t.Fatalf("New(%q).Host() = %q, want %q", in, c.Host(), want)
		}
	}
	for _, bad := range []string{"", "https://192.168.0.1", "ftp://x", "http://192.168.0.1/admin", "192.168.0.1/x", "http://", "user@host", "http://h?x=1"} {
		if _, err := New(bad); err == nil {
			t.Fatalf("New(%q): want error", bad)
		}
	}
}

func TestLoginOnceAndHeaders(t *testing.T) {
	r := tendatest.New(t)
	session := filepath.Join(t.TempDir(), "s")
	c := newTestClient(t, r, WithSessionFile(session))
	ctx := context.Background()
	if _, err := c.RouterStatus(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.NAT(ctx); err != nil {
		t.Fatal(err)
	}
	if r.Logins() != 1 {
		t.Fatalf("logins = %d, want 1", r.Logins())
	}
	call := r.LastCall(t, "GetRouterStatus")
	if call.Header.Get("X-Requested-With") != "XMLHttpRequest" {
		t.Fatalf("headers = %v", call.Header)
	}
	ck, err := (&http.Request{Header: call.Header}).Cookie("password")
	if err != nil || ck.Value == "" {
		t.Fatalf("cookie = %v, %v", ck, err)
	}

	st, err := os.Stat(session)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("session mode = %v", st.Mode().Perm())
	}
	if b, _ := os.ReadFile(session); string(b) != ck.Value {
		t.Fatalf("session file = %q, want %q", b, ck.Value)
	}

	// A second client on the same file reuses the session without a password.
	c2, err := New(r.Host(), WithHTTPClient(r.HTTPClient()), WithSessionFile(session),
		WithPassword(func(context.Context) (string, error) { t.Fatal("password requested"); return "", nil }))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.RouterStatus(ctx); err != nil {
		t.Fatal(err)
	}
	if r.Logins() != 1 {
		t.Fatalf("logins = %d, want 1", r.Logins())
	}
}

func TestStaleStoredToken(t *testing.T) {
	r := tendatest.New(t)
	session := filepath.Join(t.TempDir(), "s")
	if err := os.WriteFile(session, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := newTestClient(t, r, WithSessionFile(session))
	if _, err := c.RouterStatus(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.Logins() != 1 {
		t.Fatalf("logins = %d, want 1", r.Logins())
	}
	if b, _ := os.ReadFile(session); string(b) == "stale" {
		t.Fatal("session file not updated")
	}
}

func TestRelogin(t *testing.T) {
	for _, tt := range []struct {
		name   string
		expire func(*tendatest.Router)
	}{
		{"redirect", (*tendatest.Router).Expire},
		{"doctype", (*tendatest.Router).ExpireDoctype},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := tendatest.New(t)
			session := filepath.Join(t.TempDir(), "s")
			c := newTestClient(t, r, WithSessionFile(session))
			ctx := context.Background()
			if _, err := c.RouterStatus(ctx); err != nil {
				t.Fatal(err)
			}
			old, _ := os.ReadFile(session)

			tt.expire(r)
			if _, err := c.RouterStatus(ctx); err != nil {
				t.Fatal(err)
			}
			if r.Logins() != 2 {
				t.Fatalf("logins = %d, want 2", r.Logins())
			}
			if now, _ := os.ReadFile(session); string(now) == string(old) {
				t.Fatal("session file not updated after re-login")
			}

			tt.expire(r)
			form := url.Values{"dmzEn": {"1"}, "dmzIp": {"192.168.0.100"}}
			if err := c.set(ctx, "SetDMZCfg", form); err != nil {
				t.Fatal(err)
			}
			if got := r.LastCall(t, "SetDMZCfg").RawBody; got != form.Encode() {
				t.Fatalf("replayed body = %q, want %q", got, form.Encode())
			}
		})
	}
}

func TestAlwaysExpired(t *testing.T) {
	r := tendatest.New(t)
	r.AlwaysExpired()
	c := newTestClient(t, r)
	_, err := c.RouterStatus(context.Background())
	if !errors.Is(err, ErrSessionLost) {
		t.Fatalf("err = %v, want ErrSessionLost", err)
	}
	if r.Logins() != 2 || r.Expired() != 2 {
		t.Fatalf("logins = %d, expired = %d; want 2, 2", r.Logins(), r.Expired())
	}
}

func TestBadAndMissingPassword(t *testing.T) {
	r := tendatest.New(t)
	session := filepath.Join(t.TempDir(), "s")
	c := newTestClient(t, r, WithSessionFile(session), WithPassword(func(context.Context) (string, error) { return "wrong", nil }))
	if _, err := c.RouterStatus(context.Background()); !errors.Is(err, ErrBadPassword) {
		t.Fatalf("err = %v, want ErrBadPassword", err)
	}
	if _, err := os.Stat(session); !os.IsNotExist(err) {
		t.Fatalf("session file written after a failed login: %v", err)
	}

	c = newTestClient(t, r, WithPassword(func(context.Context) (string, error) { return "", nil }))
	if _, err := c.RouterStatus(context.Background()); !errors.Is(err, ErrNoPassword) {
		t.Fatalf("err = %v, want ErrNoPassword", err)
	}
	c = newTestClient(t, r, WithPassword(nil))
	if _, err := c.RouterStatus(context.Background()); !errors.Is(err, ErrNoPassword) {
		t.Fatalf("err = %v, want ErrNoPassword", err)
	}
}

func TestConcurrentReloginIsSingleFlight(t *testing.T) {
	r := tendatest.New(t)
	c := newTestClient(t, r)
	ctx := context.Background()
	if _, err := c.RouterStatus(ctx); err != nil {
		t.Fatal(err)
	}
	r.Expire()
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.RouterStatus(ctx)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if r.Logins() != 2 {
		t.Fatalf("logins = %d, want 2 (one initial, one re-login)", r.Logins())
	}
}

func TestDoctypeInsideJSONIsNotExpiry(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("getOnlineList", http.StatusOK, `[{"blackNum":0,"macFilterType":"black","localhostIP":"192.168.0.2","isWirelessConnect":"false","localhostName":"<!DOCTYPE html>","localhostMac":"02:00:00:00:00:02"}]`)
	c := newTestClient(t, r)
	l, err := c.OnlineList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if l.Host.Name != "<!DOCTYPE html>" || r.Logins() != 1 {
		t.Fatalf("host = %+v, logins = %d", l.Host, r.Logins())
	}
}
