// Package tenda is a client for the Tenda AC10 router's undocumented web API.
// docs/router-api.md is the reference for every endpoint used here.
//
// The package never prompts, prints or reads the environment (except
// XDG_RUNTIME_DIR in DefaultSessionPath): credentials and confirmation come
// from the caller through PasswordFunc and ConfirmFunc.
package tenda

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// PasswordFunc returns the plaintext admin password. The Client calls it
// lazily, only when a login actually happens, and at most once.
type PasswordFunc func(ctx context.Context) (string, error)

// ConfirmFunc decides whether a hazardous request may proceed. A nil error
// means proceed.
type ConfirmFunc func(ctx context.Context, h Hazard) error

// Client talks to one router. It is safe for concurrent use.
type Client struct {
	base        *url.URL
	hc          *http.Client
	password    PasswordFunc
	sessionPath string
	confirm     ConfirmFunc
	strict      bool

	mu     sync.Mutex // guards token, loaded and hash; serialises (re-)login
	token  string
	loaded bool
	hash   string
}

// Option configures a Client.
type Option func(*Client)

// WithPassword sets the password source used on login.
func WithPassword(f PasswordFunc) Option { return func(c *Client) { c.password = f } }

// WithSessionFile sets where the session token is persisted. The default is
// DefaultSessionPath(host); "" keeps the token in memory only.
func WithSessionFile(path string) Option { return func(c *Client) { c.sessionPath = path } }

// WithHTTPClient sets the underlying HTTP client. The Client works on a copy
// that never follows redirects, since a redirect to /login.html is how the
// router signals an expired session.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		cp := *hc
		c.hc = &cp
	}
}

// WithConfirm sets the hook consulted before any request on the hazard list.
// Without it every hazardous request is refused.
func WithConfirm(f ConfirmFunc) Option { return func(c *Client) { c.confirm = f } }

// WithStrictDecode makes unknown JSON fields a *DecodeError. Tests use it to
// prove a wire struct describes its fixture completely.
func WithStrictDecode() Option { return func(c *Client) { c.strict = true } }

// New returns a Client for host, which may be "192.168.0.1", "host:port" or
// "http://host:port[/]". New does no I/O.
func New(host string, opts ...Option) (*Client, error) {
	u, err := parseHost(host)
	if err != nil {
		return nil, err
	}
	c := &Client{
		base:        u,
		hc:          &http.Client{Timeout: 15 * time.Second},
		sessionPath: DefaultSessionPath(u.Host),
	}
	for _, o := range opts {
		o(c)
	}
	c.hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c, nil
}

func parseHost(host string) (*url.URL, error) {
	s := strings.TrimSpace(host)
	if s == "" {
		return nil, errors.New("router host is empty")
	}
	raw := s
	if !strings.Contains(s, "://") {
		if strings.ContainsAny(s, "/?#@") {
			return nil, fmt.Errorf("invalid router host %q: want host, host:port or http://host:port", host)
		}
		raw = "http://" + s
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid router host %q: %w", host, err)
	}
	switch {
	case u.Scheme != "http":
		return nil, fmt.Errorf("invalid router host %q: only http is supported", host)
	case u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/"):
		return nil, fmt.Errorf("invalid router host %q: want host, host:port or http://host:port", host)
	}
	return &url.URL{Scheme: "http", Host: u.Host, Path: "/"}, nil
}

// Host returns the router's host[:port].
func (c *Client) Host() string { return c.base.Host }

// kind selects how a response is judged.
type kind int

const (
	kindJSON     kind = iota // XHR endpoints answering JSON
	kindSubmit               // native form posts answering a redirect or page
	kindDownload             // cgi-bin file downloads
)

type call struct {
	method   string
	endpoint string
	query    url.Values
	form     url.Values // nil sends no body
	before   url.Values // previous values for Before-aware hazard rules
	kind     kind
}

type expiry int

const (
	notExpired expiry = iota
	expiredRedirect
	expiredDoctype
)

var unsupportedRE = regexp.MustCompile(`Form (\S+) is not defined`)

// do runs one request through the hazard gate, session handling and response
// classification.
func (c *Client) do(ctx context.Context, r call) (*RawResponse, error) {
	path, key, inline, err := normalizeEndpoint(r.endpoint)
	if err != nil {
		return nil, err
	}
	query := mergeValues(inline, r.query)

	h, hazardous := Classify(Request{Method: r.method, Endpoint: key, Query: query, Form: r.form, Before: r.before})
	if hazardous {
		if c.confirm == nil {
			return nil, &HazardError{Hazard: h, Err: ErrNotConfirmed}
		}
		if err := c.confirm(ctx, h); err != nil {
			return nil, &HazardError{Hazard: h, Err: err}
		}
	}

	tok, err := c.currentToken(ctx)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		resp, exp, err := c.send(ctx, r, path, key, query, tok)
		if err != nil {
			return nil, err
		}
		if exp == notExpired {
			return resp, nil
		}
		// A DOCTYPE body does not prove the handler did not run, so a
		// hazardous request (a reboot, say) is never replayed after one.
		if attempt > 0 || (exp == expiredDoctype && hazardous) {
			return nil, fmt.Errorf("%s: %w", key, ErrSessionLost)
		}
		if tok, err = c.relogin(ctx, tok); err != nil {
			return nil, err
		}
	}
}

func (c *Client) send(ctx context.Context, r call, path, key string, query url.Values, tok string) (*RawResponse, expiry, error) {
	u := *c.base
	u.Path = path
	u.RawQuery = query.Encode()
	var body io.Reader
	if r.form != nil {
		body = strings.NewReader(r.form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, r.method, u.String(), body)
	if err != nil {
		return nil, notExpired, err
	}
	setHeaders(req)
	if r.form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}
	req.Header.Set("Cookie", "password="+tok)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, notExpired, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, notExpired, err
	}
	raw := &RawResponse{Status: resp.StatusCode, Header: resp.Header, Body: b, endpoint: key}

	trimmed := bytes.TrimSpace(b)
	if bytes.HasPrefix(trimmed, []byte("<")) {
		if m := unsupportedRE.FindSubmatch(trimmed); m != nil {
			return nil, notExpired, &UnsupportedError{Endpoint: key, Form: string(m[1])}
		}
	}
	isRedirect := resp.StatusCode >= 300 && resp.StatusCode < 400
	if isRedirect && strings.Contains(resp.Header.Get("Location"), "login.html") {
		return nil, expiredRedirect, nil
	}
	if r.kind != kindSubmit && hasPrefixFold(trimmed, "<!DOCTYPE") {
		return nil, expiredDoctype, nil
	}
	switch {
	case isRedirect && r.kind == kindSubmit:
		return raw, notExpired, nil
	case isRedirect, resp.StatusCode >= 400:
		return nil, notExpired, &HTTPError{Endpoint: key, Status: resp.StatusCode, Snippet: snippet(b)}
	}
	return raw, notExpired, nil
}

func setHeaders(req *http.Request) {
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "text/plain, */*; q=0.01")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
}

// currentToken returns the session token, loading it from the session file
// on first use and logging in when there is none.
func (c *Client) currentToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.loaded {
		c.loaded = true
		if c.sessionPath != "" {
			c.token = loadSession(c.sessionPath)
		}
	}
	if c.token == "" {
		if err := c.loginLocked(ctx); err != nil {
			return "", err
		}
	}
	return c.token, nil
}

// relogin replaces the token a request found expired. Concurrent callers that
// saw the same stale token share one login.
func (c *Client) relogin(ctx context.Context, stale string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != stale && c.token != "" {
		return c.token, nil
	}
	if err := c.loginLocked(ctx); err != nil {
		return "", err
	}
	return c.token, nil
}

// loginLocked performs the UI's login: GET /login.html, then POST
// /login/Auth with the md5 of the password. Success is a non-empty
// "password" cookie with a body other than "1". c.mu must be held.
func (c *Client) loginLocked(ctx context.Context) error {
	if c.hash == "" {
		if c.password == nil {
			return ErrNoPassword
		}
		pw, err := c.password(ctx)
		if err != nil {
			return err
		}
		if pw == "" {
			return ErrNoPassword
		}
		sum := md5.Sum([]byte(pw))
		c.hash = hex.EncodeToString(sum[:])
	}

	u := *c.base
	u.Path = "/login.html"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	setHeaders(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	u.Path = "/login/Auth"
	form := "username=admin&password=" + c.hash
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form))
	if err != nil {
		return err
	}
	setHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err = c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(bytes.TrimSpace(b)) == "1" {
		return ErrBadPassword
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "password" && ck.Value != "" {
			c.token = ck.Value
			if c.sessionPath != "" {
				// A session file that cannot be written only costs a login
				// next time; it must not fail the command.
				_ = saveSession(c.sessionPath, c.token)
			}
			return nil
		}
	}
	return &HTTPError{Endpoint: "login", Status: resp.StatusCode, Snippet: "no session cookie"}
}

// normalizeEndpoint maps "GetDMZCfg", "goform/GetDMZCfg" and
// "/goform/GetDMZCfg" to path /goform/GetDMZCfg with key GetDMZCfg, and
// "cgi-bin/X" to /cgi-bin/X with key cgi-bin/X. An inline query is split off.
func normalizeEndpoint(ep string) (path, key string, query url.Values, err error) {
	s := strings.TrimSpace(ep)
	if i := strings.IndexByte(s, '?'); i >= 0 {
		query, err = url.ParseQuery(s[i+1:])
		if err != nil {
			return "", "", nil, fmt.Errorf("endpoint %q: %w", ep, err)
		}
		s = s[:i]
	}
	s = strings.TrimPrefix(s, "/")
	switch {
	case strings.HasPrefix(s, "cgi-bin/") && len(s) > len("cgi-bin/"):
		return "/" + s, s, query, nil
	case strings.HasPrefix(s, "goform/"):
		s = strings.TrimPrefix(s, "goform/")
	}
	if s == "" || strings.Contains(s, "/") {
		return "", "", nil, fmt.Errorf("invalid endpoint %q: want Name, goform/Name or cgi-bin/path", ep)
	}
	return "/goform/" + s, s, query, nil
}

func mergeValues(a, b url.Values) url.Values {
	out := url.Values{}
	for k, v := range a {
		out[k] = append(out[k], v...)
	}
	for k, v := range b {
		out[k] = append(out[k], v...)
	}
	return out
}

func hasPrefixFold(b []byte, prefix string) bool {
	return len(b) >= len(prefix) && strings.EqualFold(string(b[:len(prefix)]), prefix)
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}
