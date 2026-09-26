package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func init() { register("", newAPICmd) }

func newAPICmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Call any router endpoint directly",
		Long: `Send a raw request to any endpoint in docs/router-api.md, including the ones
tendactl has no command for (WAN, AP mode, firmware upgrade, ...).

<Endpoint> is a goform name (GetDMZCfg, goform/GetDMZCfg), a cgi-bin path
(cgi-bin/DownloadCfg/RouterCfm.cfg) or cloudv2 with its module and opt
(cloudv2?module=wansta&opt=query, or cloudv2 module=wansta opt=query).
Enter control characters with shell quoting, e.g. $'a\rb'.

JSON replies are pretty-printed; other replies are written raw, so
  tendactl api get cgi-bin/DownloadCfg/RouterCfm.cfg > backup.cfg
works. -o is ignored. Endpoints on the doc's "Never call casually" list
still need --yes.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(&cobra.Command{
		Use:     "get <Endpoint> [key=value...]",
		Short:   "GET an endpoint; pairs go in the query string",
		Example: "  tendactl api get GetDMZCfg\n  tendactl api get cloudv2 module=wansta opt=query",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := apiPairs(args[1:])
			if err != nil {
				return err
			}
			return apiRun(a, cmd, http.MethodGet, args[0], q, nil)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set <Endpoint> key=value...",
		Short: "POST a form to an endpoint and check its errCode",
		Example: "  tendactl api set SetDMZCfg dmzEn=1 dmzIp=192.168.0.100\n" +
			"  tendactl api set cloudv2 module=manage opt=setbasic enable=0",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			form, err := apiPairs(args[1:])
			if err != nil {
				return err
			}
			var q url.Values
			if apiIsCloud(args[0]) {
				q = url.Values{}
				for _, k := range []string{"module", "opt"} {
					if v, ok := form[k]; ok {
						q[k] = v
						delete(form, k)
					}
				}
			}
			return apiRun(a, cmd, http.MethodPost, args[0], q, form)
		},
	})
	return cmd
}

func apiPairs(args []string) (url.Values, error) {
	v := url.Values{}
	for _, p := range args {
		k, val, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid pair %q: want key=value", p)
		}
		v.Add(k, val)
	}
	return v, nil
}

// apiIsCloud reports whether endpoint names goform/cloudv2.
func apiIsCloud(endpoint string) bool {
	s, _, _ := strings.Cut(endpoint, "?")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "/"), "goform/")
	return strings.EqualFold(s, "cloudv2")
}

func apiRun(a *app, cmd *cobra.Command, method, endpoint string, query, form url.Values) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	resp, err := c.Raw(cmd.Context(), method, endpoint, query, form)
	if resp != nil {
		if resp.Status >= 300 && resp.Status < 400 {
			fmt.Fprintf(cmd.ErrOrStderr(), "HTTP %d Location: %s\n", resp.Status, resp.Header.Get("Location"))
		}
		out := cmd.OutOrStdout()
		if resp.IsJSON() {
			var buf bytes.Buffer
			if json.Indent(&buf, bytes.TrimSpace(resp.Body), "", "  ") == nil {
				buf.WriteByte('\n')
				if _, werr := buf.WriteTo(out); werr != nil {
					return werr
				}
				return err
			}
		}
		if _, werr := out.Write(resp.Body); werr != nil {
			return werr
		}
	}
	return err
}
