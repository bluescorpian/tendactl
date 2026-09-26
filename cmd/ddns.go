package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

func init() { register("", newDDNSCmd) }

func newDDNSCmd(a *app) *cobra.Command {
	var showPassword bool
	show := func(cmd *cobra.Command, _ []string) error {
		c, err := a.client()
		if err != nil {
			return err
		}
		d, err := c.DDNS(cmd.Context())
		if err != nil {
			return err
		}
		d.Password = maskSecret(d.Password, showPassword)
		return a.render(cmd, d, func(w io.Writer) error { return ddnsText(w, d) })
	}
	cmd := &cobra.Command{
		Use:   "ddns",
		Short: "Manage Dynamic DNS",
		Long:  `Manage "DDNS" (Advanced Settings).`,
		Args:  cobra.NoArgs,
		RunE:  show,
	}
	cmd.PersistentFlags().BoolVar(&showPassword, "show-password", false, "show the DDNS password instead of masking it")
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the DDNS configuration",
		Args:  cobra.NoArgs,
		RunE:  show,
	})

	var provider, domain, user, password string
	set := &cobra.Command{
		Use:   "set",
		Short: "Set the DDNS provider, domain or credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !changed(cmd, "provider", "domain", "user", "password") {
				return errNothingToChange
			}
			var providerWire string
			if changed(cmd, "provider") {
				var err error
				if providerWire, err = ddnsProviderWire(provider); err != nil {
					return err
				}
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.DDNS, c.SetDDNS, func(d *tenda.DDNS) error {
				if changed(cmd, "provider") {
					d.Provider = providerWire
				}
				if changed(cmd, "domain") {
					d.Domain = domain
				}
				if changed(cmd, "user") {
					d.User = user
				}
				if changed(cmd, "password") {
					d.Password = password
				}
				return nil
			}, "DDNS settings updated")
		},
	}
	set.Flags().StringVar(&provider, "provider", "", "DDNS provider: no-ip.com, dyndns.org, 88ip.cn or oray.com")
	set.Flags().StringVar(&domain, "domain", "", "DDNS domain name (unused by 88ip.cn and oray.com)")
	set.Flags().StringVar(&user, "user", "", "DDNS account user name")
	set.Flags().StringVar(&password, "password", "", "DDNS account password")
	cmd.AddCommand(set)

	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Turn DDNS on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.DDNS, c.SetDDNS, func(d *tenda.DDNS) error { d.Enabled = true; return nil }, "DDNS enabled")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Turn DDNS off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			return update(a, cmd, c.DDNS, c.SetDDNS, func(d *tenda.DDNS) error { d.Enabled = false; return nil }, "DDNS disabled")
		},
	})
	return cmd
}

// ddnsProviderWire accepts the doc's wire values plus the UI's own label for
// dyn.com/dns/, "dyndns.org".
func ddnsProviderWire(s string) (string, error) {
	switch strings.ToLower(s) {
	case "no-ip.com":
		return "no-ip.com", nil
	case "dyndns.org", "dyn.com/dns/":
		return "dyn.com/dns/", nil
	case "88ip.cn":
		return "88ip.cn", nil
	case "oray.com":
		return "oray.com", nil
	default:
		return "", fmt.Errorf("invalid --provider %q: want no-ip.com, dyndns.org, 88ip.cn or oray.com", s)
	}
}

func ddnsProviderLabel(s string) string {
	if s == "dyn.com/dns/" {
		return "dyndns.org"
	}
	return s
}

func ddnsText(w io.Writer, d tenda.DDNS) error {
	return fields(w,
		"Enabled", onOff(d.Enabled),
		"Provider", ddnsProviderLabel(d.Provider),
		"User", d.User,
		"Password", d.Password,
		"Domain", d.Domain,
		"Connected", onOff(d.Connected),
	)
}
