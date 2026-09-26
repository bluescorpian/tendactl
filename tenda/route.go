package tenda

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
)

// routeMaxUserRoutes is the UI's client-side limit.
const routeMaxUserRoutes = 10

var (
	ErrRouteFull    = fmt.Errorf("the router allows at most %d user-added static routes", routeMaxUserRoutes)
	ErrRouteNoMatch = errors.New("no user-added static route with that network and mask")
)

// Route is one row of "Static Route" (Advanced Settings): GetStaticRouteCfg
// / SetStaticRouteCfg. System is true for a route the router itself added;
// system routes are read-only and are never part of a set's payload.
type Route struct {
	Network   string `json:"network"`
	Mask      string `json:"mask"`
	Gateway   string `json:"gateway"`
	Interface string `json:"interface"`
	System    bool   `json:"system"`
	Active    bool   `json:"active"` // false: this route is currently overridden and not in effect
}

// RouteConfig is the whole "Static Route" page.
type RouteConfig struct {
	LANIP      string  `json:"lanIp"`
	LANMask    string  `json:"lanMask"`
	WANMask    string  `json:"wanMask"`
	WANGateway string  `json:"wanGateway"`
	Routes     []Route `json:"routes"`
}

type routeEntryWire struct {
	Network     string `json:"network"`
	Mask        string `json:"mask"`
	Gateway     string `json:"gateway"`
	Ifname      string `json:"ifname"`
	OperateType string `json:"operateType"`
	Effective   string `json:"effective"`
}

type routeWire struct {
	LanIp      string           `json:"lanIp"`
	LanMask    string           `json:"lanMask"`
	WanMask    string           `json:"wanMask"`
	WanGateway string           `json:"wanGateway"`
	RouteList  []routeEntryWire `json:"routeList"`
}

// Routes reads GetStaticRouteCfg.
func (c *Client) Routes(ctx context.Context) (RouteConfig, error) {
	var w routeWire
	if err := c.get(ctx, "GetStaticRouteCfg", nil, &w); err != nil {
		return RouteConfig{}, err
	}
	cfg := RouteConfig{LANIP: w.LanIp, LANMask: w.LanMask, WANMask: w.WanMask, WANGateway: w.WanGateway, Routes: make([]Route, 0, len(w.RouteList))}
	for _, e := range w.RouteList {
		cfg.Routes = append(cfg.Routes, Route{
			Network: e.Network, Mask: e.Mask, Gateway: e.Gateway, Interface: e.Ifname,
			System: e.OperateType != "1", Active: e.Effective == "1",
		})
	}
	return cfg, nil
}

// SetRoutes replaces the whole user-added route list (SetStaticRouteCfg).
// System routes are never sent, matching the doc: only operateType=="1"
// rows are ever part of this payload.
func (c *Client) SetRoutes(ctx context.Context, routes []Route) error {
	rows := make([][]string, len(routes))
	for i, r := range routes {
		rows[i] = []string{r.Network, r.Mask, r.Gateway, r.Interface}
	}
	list, err := TildeComma.Encode(rows)
	if err != nil {
		return err
	}
	return c.set(ctx, "SetStaticRouteCfg", url.Values{"list": {list}})
}

// AddRoute appends r after the UI's one confirmed check, at most 10
// user-added routes. r.Interface defaults to "WAN1" and r.Gateway to
// "0.0.0.0" when empty, matching the UI. Nothing is posted when a check
// fails.
//
// The doc's UI also describes a client-side duplicate/overlapping-network
// check (checkIpInSameSegment in static_route.js), but that function has a
// loop-index bug that leaves it a no-op in the live UI, so tendactl does not
// replicate it.
func (c *Client) AddRoute(ctx context.Context, r Route) error {
	network := net.ParseIP(r.Network).To4()
	if network == nil {
		return fmt.Errorf("invalid network address %q", r.Network)
	}
	mask := net.ParseIP(r.Mask).To4()
	if mask == nil {
		return fmt.Errorf("invalid subnet mask %q", r.Mask)
	}
	// The #network/#mask blur handler in js/static_route.js always rewrites
	// the network field to mask&network before getSubmitData() reads it, so
	// the UI can never submit a network with host bits set; match that here
	// rather than sending r.Network as given.
	r.Network = maskIPv4(network, mask).String()
	if r.Gateway == "" {
		r.Gateway = "0.0.0.0"
	}
	if net.ParseIP(r.Gateway).To4() == nil {
		return fmt.Errorf("invalid gateway address %q", r.Gateway)
	}
	if r.Interface == "" {
		r.Interface = "WAN1"
	}
	cfg, err := c.Routes(ctx)
	if err != nil {
		return err
	}
	user := routeUserRoutes(cfg.Routes)
	if len(user) >= routeMaxUserRoutes {
		return ErrRouteFull
	}
	return c.SetRoutes(ctx, append(user, r))
}

// RemoveRoute deletes the user-added route to network/mask. It returns
// ErrRouteNoMatch, having posted nothing, when there is none; a system
// route with the same network/mask is not removable and does not match.
func (c *Client) RemoveRoute(ctx context.Context, network, mask string) error {
	cfg, err := c.Routes(ctx)
	if err != nil {
		return err
	}
	user := routeUserRoutes(cfg.Routes)
	kept := make([]Route, 0, len(user))
	for _, r := range user {
		if r.Network != network || r.Mask != mask {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(user) {
		return fmt.Errorf("%w (%s/%s)", ErrRouteNoMatch, network, mask)
	}
	return c.SetRoutes(ctx, kept)
}

func routeUserRoutes(routes []Route) []Route {
	user := make([]Route, 0, len(routes))
	for _, r := range routes {
		if !r.System {
			user = append(user, r)
		}
	}
	return user
}

// maskIPv4 ANDs ip and mask octet by octet, as the blur handler in
// js/static_route.js does with the entered network/mask.
func maskIPv4(ip, mask net.IP) net.IP {
	out := make(net.IP, net.IPv4len)
	for i := range out {
		out[i] = ip[i] & mask[i]
	}
	return out
}
