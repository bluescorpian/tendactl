// Package tendatest is a fake Tenda AC10 router for tests. It serves the
// real GET responses in tenda/testdata, simulates login and session expiry,
// and records every authenticated request.
//
// It deliberately does not import package tenda: its forbidden-endpoint list
// is written independently of tenda's hazard table, so a mistake in one is
// caught by the other.
package tendatest

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Password is the admin password the fake accepts.
const Password = "tendatest-pw"

// Fixed bodies served by the download endpoints.
var (
	ConfigBytes = []byte("tendatest RouterCfm.cfg\x00\x01\x02")
	SyslogBytes = []byte("tendatest syslog.tar\x00\x03\x04")
)

// Call is one recorded request. Endpoint is the key used by the Router's
// methods: "SetDMZCfg", "cloudv2?module=m&opt=o" or
// "cgi-bin/DownloadCfg/RouterCfm.cfg".
type Call struct {
	Method, Endpoint, Path string
	Query, Form            url.Values
	RawBody                string
	Header                 http.Header
}

// Response is what a Handle func returns.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// JSON is a 200 response with body s.
func JSON(s string) Response { return Response{Status: http.StatusOK, Body: []byte(s)} }

type expiryMode int

const (
	expireRedirect expiryMode = iota
	expireDoctype
)

// Router is the fake. Its methods are safe for concurrent use.
type Router struct {
	*httptest.Server

	t    testing.TB
	fsys fs.FS

	mu          sync.Mutex
	tokens      map[string]bool
	mode        expiryMode
	always      bool
	handlers    map[string]func(Call) Response
	allowed     map[string]bool
	unsupported map[string]bool
	calls       []Call
	logins      int
	expired     int
}

// Option configures a Router.
type Option func(*Router)

// WithFixtures serves GET responses from fsys instead of tenda/testdata.
func WithFixtures(fsys fs.FS) Option { return func(r *Router) { r.fsys = fsys } }

// New starts a fake router, closed when the test ends.
func New(t testing.TB, opts ...Option) *Router {
	t.Helper()
	r := &Router{
		t:           t,
		tokens:      map[string]bool{},
		handlers:    map[string]func(Call) Response{},
		allowed:     map[string]bool{},
		unsupported: map[string]bool{},
	}
	for _, o := range opts {
		o(r)
	}
	if r.fsys == nil {
		_, file, _, _ := runtime.Caller(0)
		r.fsys = os.DirFS(filepath.Join(filepath.Dir(file), "..", "testdata"))
	}
	r.Server = httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(r.Close)
	return r
}

// Host is the fake's "127.0.0.1:port".
func (r *Router) Host() string { return strings.TrimPrefix(r.URL, "http://") }

// HTTPClient returns a client whose transport refuses every host but the
// fake's, so a test cannot reach a real router.
func (r *Router) HTTPClient() *http.Client {
	return &http.Client{
		Transport: lockedTransport{host: r.Host(), rt: r.Server.Client().Transport},
		Timeout:   10 * time.Second,
	}
}

type lockedTransport struct {
	host string
	rt   http.RoundTripper
}

func (l lockedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != l.host {
		return nil, fmt.Errorf("tendatest: refusing request to %s (only %s is allowed)", req.URL.Host, l.host)
	}
	return l.rt.RoundTrip(req)
}

// Expire invalidates every session; the next request answers 302 /login.html.
func (r *Router) Expire() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = map[string]bool{}
	r.mode = expireRedirect
}

// ExpireDoctype is Expire, but expired requests get a 200 login page.
func (r *Router) ExpireDoctype() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = map[string]bool{}
	r.mode = expireDoctype
}

// AlwaysExpired makes login succeed while every protected request expires.
func (r *Router) AlwaysExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.always = true
}

// Handle answers endpoint with h.
func (r *Router) Handle(endpoint string, h func(Call) Response) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[endpoint] = h
}

// Reply answers endpoint with a fixed status and body.
func (r *Router) Reply(endpoint string, status int, body string) {
	r.Handle(endpoint, func(Call) Response { return Response{Status: status, Body: []byte(body)} })
}

// ErrCode answers endpoint with {"errCode":code} ({"err_code":code} for
// cloudv2). code is marshalled as given, so 2 and "2" differ on the wire.
func (r *Router) ErrCode(endpoint string, code any) {
	field := "errCode"
	if strings.HasPrefix(endpoint, "cloudv2") {
		field = "err_code"
	}
	b, err := json.Marshal(map[string]any{field: code})
	if err != nil {
		r.t.Fatal(err)
	}
	r.Reply(endpoint, http.StatusOK, string(b))
}

// Unsupported answers endpoint with the router's "Form X is not defined" page.
func (r *Router) Unsupported(endpoint string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unsupported[endpoint] = true
}

// Allow permits requests to endpoints on the forbidden list.
func (r *Router) Allow(endpoints ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range endpoints {
		r.allowed[strings.ToLower(e)] = true
	}
}

// Calls returns every authenticated request, GETs included; login requests
// and expired requests are not recorded.
func (r *Router) Calls() []Call {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Call(nil), r.calls...)
}

// CallsTo returns the recorded calls to endpoint.
func (r *Router) CallsTo(endpoint string) []Call {
	var out []Call
	for _, c := range r.Calls() {
		if c.Endpoint == endpoint {
			out = append(out, c)
		}
	}
	return out
}

// LastCall returns the last call to endpoint, failing the test if none.
func (r *Router) LastCall(t testing.TB, endpoint string) Call {
	t.Helper()
	calls := r.CallsTo(endpoint)
	if len(calls) == 0 {
		t.Fatalf("tendatest: no call to %s", endpoint)
	}
	return calls[len(calls)-1]
}

// Logins counts POST /login/Auth attempts.
func (r *Router) Logins() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.logins
}

// Expired counts protected requests answered with an expiry response.
func (r *Router) Expired() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.expired
}

// forbidden is docs/router-api.md "Never call casually", written
// independently of tenda/hazard.go on purpose. Keys are lower-case.
var forbidden = map[string]bool{
	"systoolreboot":             true,
	"systoolrestoreset":         true,
	"setapmodecfg":              true,
	"wifiextraset":              true,
	"setiptvcfg":                true,
	"wanparametersetting":       true,
	"fast_setting_internet_set": true,
	"advsetmacmtuwan":           true,
	"advsetlanip":               true,
	"systoolchangepwd":          true,
	"cgi-bin/uploadcfg":         true,
	"cgi-bin/upgrade":           true,
	"cloudv2?module=olupgrade&opt=queryupgrade": true,
}

// nativeForm maps the native form endpoints to the page they redirect to.
var nativeForm = map[string]string{
	"systoolreboot":     "/system_reboot.html",
	"systoolrestoreset": "/system_reboot.html",
	"systoolchangepwd":  "/system_password.html",
	"cgi-bin/uploadcfg": "/system_backup.html",
	"cgi-bin/upgrade":   "/system_upgrade.html",
}

func endpointKey(req *http.Request) (string, bool) {
	p := req.URL.Path
	switch {
	case strings.HasPrefix(p, "/goform/"):
		name := strings.TrimPrefix(p, "/goform/")
		if name == "cloudv2" {
			q := req.URL.Query()
			return fmt.Sprintf("cloudv2?module=%s&opt=%s", q.Get("module"), q.Get("opt")), true
		}
		return name, true
	case strings.HasPrefix(p, "/cgi-bin/"):
		return strings.TrimPrefix(p, "/"), true
	}
	return "", false
}

func (r *Router) serve(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	switch {
	case req.URL.Path == "/login.html":
		io.WriteString(w, "<!DOCTYPE html><html><body>tendatest login</body></html>")
		return
	case req.URL.Path == "/login/Auth":
		r.login(w, body)
		return
	}
	key, ok := endpointKey(req)
	if !ok {
		http.NotFound(w, req)
		return
	}

	r.mu.Lock()
	lower := strings.ToLower(key)
	if forbidden[lower] && !r.allowed[lower] {
		r.t.Errorf("tendatest: forbidden endpoint %s %s called without Allow", req.Method, key)
	}
	cookie, _ := req.Cookie("password")
	if r.always || cookie == nil || !r.tokens[cookie.Value] {
		r.expired++
		mode := r.mode
		r.mu.Unlock()
		if mode == expireDoctype {
			io.WriteString(w, "<!DOCTYPE html><html><body>tendatest login</body></html>")
			return
		}
		http.Redirect(w, req, "/login.html", http.StatusFound)
		return
	}
	c := Call{Method: req.Method, Endpoint: key, Path: req.URL.Path, Query: req.URL.Query(), RawBody: string(body), Header: req.Header.Clone()}
	if form, err := url.ParseQuery(string(body)); err == nil {
		c.Form = form
	}
	r.calls = append(r.calls, c)
	h := r.handlers[key]
	unsupported := r.unsupported[key]
	r.mu.Unlock()

	switch {
	case h != nil:
		resp := h(c)
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		if resp.Status == 0 {
			resp.Status = http.StatusOK
		}
		w.WriteHeader(resp.Status)
		w.Write(resp.Body)
	case unsupported:
		r.writeUnsupported(w, key)
	case nativeForm[lower] != "":
		http.Redirect(w, req, nativeForm[lower], http.StatusFound)
	case lower == "cgi-bin/downloadcfg/routercfm.cfg":
		w.Write(ConfigBytes)
	case lower == "cgi-bin/downloadlog/syslog.tar":
		w.Write(SyslogBytes)
	case req.Method == http.MethodGet:
		r.serveFixture(w, key)
	case strings.HasPrefix(key, "cloudv2"):
		io.WriteString(w, `{"err_code":0}`)
	default:
		io.WriteString(w, `{"errCode":0}`)
	}
}

func (r *Router) login(w http.ResponseWriter, body []byte) {
	form, _ := url.ParseQuery(string(body))
	sum := md5.Sum([]byte(Password))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logins++
	if form.Get("username") != "admin" || form.Get("password") != hex.EncodeToString(sum[:]) {
		io.WriteString(w, "1")
		return
	}
	b := make([]byte, 16)
	rand.Read(b)
	tok := hex.EncodeToString(b)
	r.tokens[tok] = true
	http.SetCookie(w, &http.Cookie{Name: "password", Value: tok, Path: "/"})
	w.Header().Set("Location", "/main.html")
	w.WriteHeader(http.StatusFound)
}

func (r *Router) serveFixture(w http.ResponseWriter, key string) {
	name := key + ".json"
	if strings.HasPrefix(key, "cloudv2?") {
		q, _ := url.ParseQuery(strings.TrimPrefix(key, "cloudv2?"))
		name = fmt.Sprintf("cloudv2_module_%s_opt_%s.json", q.Get("module"), q.Get("opt"))
	}
	b, err := fs.ReadFile(r.fsys, name)
	if err != nil {
		r.t.Errorf("tendatest: no fixture for %s", key)
		r.writeUnsupported(w, key)
		return
	}
	w.Write(b)
}

func (r *Router) writeUnsupported(w http.ResponseWriter, key string) {
	page, err := fs.ReadFile(r.fsys, "unsupported_form.html")
	if err != nil {
		page = []byte("<html><body><p>Form GetUsbCfg is not defined</p></body></html>")
	}
	name := key
	if i := strings.IndexByte(name, '?'); i >= 0 {
		name = name[:i]
	}
	w.Write([]byte(strings.ReplaceAll(string(page), "GetUsbCfg", name)))
}
