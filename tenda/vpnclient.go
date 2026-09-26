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

// SetVPNClient sends the full UI form (SetPptpClientCfg): all seven fields
// are sent on both enable and disable, matching the UI's own doc'd
// disabled-state example.
//
// prev is the value most recently read. getSubmitData() in
// js/pptp_client.js only takes clientType/domain/userName/password (and, for
// clientType=="pptp", clientMppe/clientMppeOp too) from the live form when
// clientEn=="1"; otherwise every field but clientEn is resent from prev
// unchanged. Even while enabling, clientMppe/clientMppeOp are resent from
// prev when clientType=="l2tp", since MPPE isn't user-configurable there.
func (c *Client) SetVPNClient(ctx context.Context, prev, next VPNClient) error {
	return c.set(ctx, "SetPptpClientCfg", vpnClientForm(prev, next))
}

func vpnClientForm(prev, next VPNClient) url.Values {
	clientType, domain, user, password := next.Type, next.Domain, next.User, next.Password
	mppe, mppeBits := next.MPPE, next.MPPEBits
	switch {
	case !next.Enabled:
		clientType, domain, user, password = prev.Type, prev.Domain, prev.User, prev.Password
		mppe, mppeBits = prev.MPPE, prev.MPPEBits
	case next.Type == "l2tp":
		mppe, mppeBits = prev.MPPE, prev.MPPEBits
	}
	return url.Values{
		"clientEn":     {flag(next.Enabled)},
		"clientType":   {clientType},
		"clientMppe":   {flag(mppe)},
		"clientMppeOp": {strconv.Itoa(mppeBits)},
		"domain":       {domain},
		"userName":     {user},
		"password":     {password},
	}
}
