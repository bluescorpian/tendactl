package tenda

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// WiFiChannel is "Channel & Bandwidth" (WiFi Settings): WifiRadioGet /
// WifiRadioSet.
type WiFiChannel struct {
	Band24 RadioBand `json:"band24"`
	Band5  RadioBand `json:"band5"`
}

// RadioBand is one radio's mode, channel and bandwidth. Country and Channels
// are read-only.
type RadioBand struct {
	Mode     string `json:"mode"`    // 2.4 GHz: "bgn", "bg" or "n only"; 5 GHz: "ac" or "ac only"
	Channel  int    `json:"channel"` // 0 = Auto
	Width    string `json:"width"`   // "20", "40", "80" (5 GHz only) or "auto"
	Country  string `json:"country"`
	Channels []int  `json:"channels"` // valid channel numbers for Width, Auto excluded
}

type wifiChannelWire struct {
	AdvMode            string           `json:"adv_mode"`
	AdvChannel         string           `json:"adv_channel"`
	AdvBand            string           `json:"adv_band"`
	AdvExtendChannel   string           `json:"adv_extend_channel"` // not surfaced as a control; unused
	AdvCountry         string           `json:"adv_country"`
	Channel            []int            `json:"channel"`
	AdvMode5g          string           `json:"adv_mode_5g"`
	AdvChannel5g       string           `json:"adv_channel_5g"`
	AdvBand5g          string           `json:"adv_band_5g"`
	AdvExtendChannel5g string           `json:"adv_extend_channel_5g"` // unused
	AdvCountry5g       string           `json:"adv_country_5g"`
	Channel5g          map[string][]int `json:"channel_5g"`
}

// WiFiChannel reads WifiRadioGet.
func (c *Client) WiFiChannel(ctx context.Context) (WiFiChannel, error) {
	var w wifiChannelWire
	if err := c.get(ctx, "WifiRadioGet", nil, &w); err != nil {
		return WiFiChannel{}, err
	}
	ch24, err := strconv.Atoi(w.AdvChannel)
	if err != nil {
		return WiFiChannel{}, &DecodeError{Endpoint: "WifiRadioGet", Err: fmt.Errorf("adv_channel: %w", err)}
	}
	ch5, err := strconv.Atoi(w.AdvChannel5g)
	if err != nil {
		return WiFiChannel{}, &DecodeError{Endpoint: "WifiRadioGet", Err: fmt.Errorf("adv_channel_5g: %w", err)}
	}
	return WiFiChannel{
		Band24: RadioBand{Mode: w.AdvMode, Channel: ch24, Width: w.AdvBand, Country: w.AdvCountry, Channels: dropAutoPlaceholder(w.Channel)},
		Band5:  RadioBand{Mode: w.AdvMode5g, Channel: ch5, Width: w.AdvBand5g, Country: w.AdvCountry5g, Channels: dropAutoPlaceholder(wifi5gChannels(w.Channel5g, w.AdvBand5g))},
	}, nil
}

// dropAutoPlaceholder removes index 0, the "Auto" placeholder the doc
// describes for both channel lists.
func dropAutoPlaceholder(list []int) []int {
	if len(list) == 0 {
		return nil
	}
	return list[1:]
}

// wifi5gChannels picks channel_5g's list for width, or, when width is
// "auto", the widest list the router reports (80 > 40 > 20), per the doc's
// "option list offered depends on which keys exist in channel_5g" note.
func wifi5gChannels(m map[string][]int, width string) []int {
	if list, ok := m[width]; ok {
		return list
	}
	for _, w := range []string{"80", "40", "20"} {
		if list, ok := m[w]; ok {
			return list
		}
	}
	return nil
}

// SetWiFiChannel sends the full UI form: both bands' mode, channel and
// bandwidth are always sent together, even when only one band changed.
func (c *Client) SetWiFiChannel(ctx context.Context, ch WiFiChannel) error {
	form := url.Values{
		"adv_mode":       {ch.Band24.Mode},
		"adv_channel":    {strconv.Itoa(ch.Band24.Channel)},
		"adv_band":       {ch.Band24.Width},
		"adv_mode_5g":    {ch.Band5.Mode},
		"adv_channel_5g": {strconv.Itoa(ch.Band5.Channel)},
		"adv_band_5g":    {ch.Band5.Width},
	}
	return c.set(ctx, "WifiRadioSet", form)
}
