package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bluescorpian/tendactl/tenda"
)

// fixtureClient serves tenda/testdata/<Name>.json behind a stub login.
func fixtureClient(t *testing.T) *tenda.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/login/Auth":
			http.SetCookie(w, &http.Cookie{Name: "password", Value: "tok", Path: "/"})
			http.Redirect(w, r, "/main.html", http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/goform/"):
			http.ServeFile(w, r, filepath.Join("..", "tenda", "testdata", strings.TrimPrefix(r.URL.Path, "/goform/")+".json"))
		}
	}))
	t.Cleanup(srv.Close)
	c, err := tenda.New(srv.URL, tenda.WithSessionFile(""), tenda.WithStrictDecode(),
		tenda.WithPassword(func(context.Context) (string, error) { return "pw", nil }))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestStatusTextMatchesLegacy(t *testing.T) {
	s, err := fixtureClient(t).RouterStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := statusText(&buf, s); err != nil {
		t.Fatal(err)
	}
	if want := readGolden(t, "status_legacy.golden"); buf.String() != want {
		t.Fatalf("statusText =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestClientsTextGolden(t *testing.T) {
	l, err := fixtureClient(t).OnlineList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := clientsText(&buf, l); err != nil {
		t.Fatal(err)
	}
	golden(t, "clients", buf.String())
	sameDataRows(t, readGolden(t, "online_legacy.golden"), buf.String())
}

func TestNATTextGolden(t *testing.T) {
	cfg, err := fixtureClient(t).NAT(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := natText(&buf, cfg.Rules); err != nil {
		t.Fatal(err)
	}
	golden(t, "nat", buf.String())
	sameDataRows(t, readGolden(t, "nat_legacy.golden"), buf.String())
}

func TestTableEmptyLists(t *testing.T) {
	var buf bytes.Buffer
	if err := natText(&buf, nil); err != nil || buf.String() != "No port forwarding rules configured\n" {
		t.Fatalf("natText(nil) = %q, %v", buf.String(), err)
	}
	buf.Reset()
	if err := clientsText(&buf, tenda.OnlineList{Host: tenda.OnlineHost{Name: "h", IP: "1", MAC: "m"}}); err != nil || buf.String() != "h @ 1 (m)\n\nNo connected devices\n" {
		t.Fatalf("clientsText(empty) = %q, %v", buf.String(), err)
	}
}

// sameDataRows checks the new table's rows carry the legacy table's values:
// the lines after the "─" rule, split on whitespace, must match.
func sameDataRows(t *testing.T, legacy, got string) {
	t.Helper()
	rows := func(s string) []string {
		var out []string
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			if strings.HasPrefix(l, "─") {
				for _, d := range lines[i+1:] {
					if f := strings.Fields(d); len(f) > 0 {
						out = append(out, strings.Join(f, " "))
					}
				}
				break
			}
		}
		return out
	}
	a, b := rows(legacy), rows(got)
	if len(a) == 0 || strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Fatalf("data rows differ:\nlegacy %q\nnew    %q", a, b)
	}
}
