package tenda

import (
	"context"
	"net/url"
)

// DDNS is the "DDNS" page (Advanced Settings): GetDDNSCfg / SetDDNSCfg.
type DDNS struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"` // "no-ip.com", "dyn.com/dns/" (UI: "dyndns.org"), "88ip.cn" or "oray.com"
	User     string `json:"user"`
	Password string `json:"password"`
	Domain   string `json:"domain"` // hidden in the UI, and meaningless, when Provider is 88ip.cn or oray.com

	Connected bool `json:"connected"` // read-only
}

type ddnsWire struct {
	DdnsEn     Flag   `json:"ddnsEn"`
	ServerName string `json:"serverName"`
	DdnsUser   string `json:"ddnsUser"`
	DdnsPwd    string `json:"ddnsPwd"`
	DdnsDomain string `json:"ddnsDomain"`
	DdnsStatus string `json:"ddnsStatus"`
}

// DDNS reads GetDDNSCfg.
func (c *Client) DDNS(ctx context.Context) (DDNS, error) {
	var w ddnsWire
	if err := c.get(ctx, "GetDDNSCfg", nil, &w); err != nil {
		return DDNS{}, err
	}
	return DDNS{
		Enabled:   bool(w.DdnsEn),
		Provider:  w.ServerName,
		User:      w.DdnsUser,
		Password:  w.DdnsPwd,
		Domain:    w.DdnsDomain,
		Connected: w.DdnsStatus == "1",
	}, nil
}

// SetDDNS sends the full UI form.
func (c *Client) SetDDNS(ctx context.Context, d DDNS) error {
	form := url.Values{
		"ddnsEn":     {flag(d.Enabled)},
		"serverName": {d.Provider},
		"ddnsUser":   {d.User},
		"ddnsPwd":    {d.Password},
		"ddnsDomain": {d.Domain},
	}
	return c.set(ctx, "SetDDNSCfg", form)
}
