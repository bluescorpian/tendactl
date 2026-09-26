package tenda

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VPNClient is "PPTP/L2TP Client" (VPN tab): GetPptpClientCfg /
// SetPptpClientCfg. This dials out to an upstream VPN server; it is
// unrelated to VPNServer, this router's own PPTP server.
type VPNClient struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"` // "pptp" or "l2tp"
	Domain   string `json:"domain"`
	MPPE     bool   `json:"mppe"`     // PPTP only; ignored for l2tp
	MPPEBits int    `json:"mppeBits"` // 40 or 128, PPTP only
	User     string `json:"user"`
	Password string `json:"password"`

	// Read-only tunnel status, independent of which Type is selected.
	PPTPStatus string `json:"pptpStatus"` // "disconnected", "connected" or "connecting"
	PPTPIP     string `json:"pptpIp"`
	L2TPStatus string `json:"l2tpStatus"`
	L2TPIP     string `json:"l2tpIp"`
}

type vpnClientWire struct {
	ClientEn     Flag   `json:"clientEn"`
	ClientType   string `json:"clientType"`
	Domain       string `json:"domain"`
	ClientMppe   Flag   `json:"clientMppe"`
	ClientMppeOp string `json:"clientMppeOp"`
	ClientWanid  string `json:"clientWanid"`
	UserName     string `json:"userName"`
	Password     string `json:"password"`
	ClientIp     string `json:"clientIp"`
	ClientMask   string `json:"clientMask"`
	PptpStatus   string `json:"pptpStatus"`
	PptpIp       string `json:"pptpIp"`
	L2tpStatus   string `json:"l2tpStatus"`
	L2tpIp       string `json:"l2tpIp"`
	WanConnType  string `json:"wanConnType"`
	WanUser      string `json:"wanUser"`
	WanIp        string `json:"wanIp"`
}

// vpnClientStatusLabel maps the doc's 0-2 tunnel status enum.
func vpnClientStatusLabel(code string) string {
	switch code {
	case "0":
		return "disconnected"
	case "1":
		return "connected"
	case "2":
		return "connecting"
	}
	return code
}

// VPNClient reads GetPptpClientCfg.
func (c *Client) VPNClient(ctx context.Context) (VPNClient, error) {
	var w vpnClientWire
	if err := c.get(ctx, "GetPptpClientCfg", nil, &w); err != nil {
		return VPNClient{}, err
	}
	bits, err := strconv.Atoi(w.ClientMppeOp)
	if err != nil {
		return VPNClient{}, &DecodeError{Endpoint: "GetPptpClientCfg", Err: fmt.Errorf("clientMppeOp: %w", err)}
	}
	return VPNClient{
		Enabled: bool(w.ClientEn), Type: w.ClientType, Domain: w.Domain,
		MPPE: bool(w.ClientMppe), MPPEBits: bits,
		User: w.UserName, Password: w.Password,
		PPTPStatus: vpnClientStatusLabel(w.PptpStatus), PPTPIP: w.PptpIp,
		L2TPStatus: vpnClientStatusLabel(w.L2tpStatus), L2TPIP: w.L2tpIp,
	}, nil
}

// SetVPNClient sends the full UI form (SetPptpClientCfg). The UI sends every
// field on both enable and disable (its own doc'd disabled-state example
// includes clientType/clientMppe/clientMppeOp/domain/userName/password, not
// just clientEn), so this always sends all seven; a caller that only wants
// to flip Enabled gets that behaviour for free through update[T].
func (c *Client) SetVPNClient(ctx context.Context, v VPNClient) error {
	form := url.Values{
		"clientEn":     {flag(v.Enabled)},
		"clientType":   {v.Type},
		"clientMppe":   {flag(v.MPPE)},
		"clientMppeOp": {strconv.Itoa(v.MPPEBits)},
		"domain":       {v.Domain},
		"userName":     {v.User},
		"password":     {v.Password},
	}
	return c.set(ctx, "SetPptpClientCfg", form)
}
