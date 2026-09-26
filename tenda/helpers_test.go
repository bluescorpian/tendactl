package tenda

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

// newTestClient returns a strict-decoding Client for r with the fake's
// password and a session file in a temp dir.
func newTestClient(t *testing.T, r *tendatest.Router, opts ...Option) *Client {
	t.Helper()
	base := []Option{
		WithStrictDecode(),
		WithPassword(func(context.Context) (string, error) { return tendatest.Password, nil }),
		WithSessionFile(filepath.Join(t.TempDir(), "s")),
		WithHTTPClient(r.HTTPClient()),
	}
	c, err := New(r.Host(), append(base, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func allowAll(context.Context, Hazard) error { return nil }
