package tenda

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Flag is a wire boolean. The router spells them "0"/"1", 0/1,
// "true"/"false" or true/false depending on the endpoint.
type Flag bool

func (f *Flag) UnmarshalJSON(b []byte) error {
	switch string(bytes.TrimSpace(b)) {
	case `"1"`, `1`, `"true"`, `true`:
		*f = true
	case `"0"`, `0`, `"false"`, `false`:
		*f = false
	default:
		return fmt.Errorf("flag: unexpected value %s", b)
	}
	return nil
}

func (f Flag) MarshalJSON() ([]byte, error) { return json.Marshal(bool(f)) }

// flag spells b the way most setters expect it.
func flag(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Number is a wire integer sent either as a JSON number or a numeric string
// (errCode, err_code, schedWifiEnable, ...).
type Number int

func (n *Number) UnmarshalJSON(b []byte) error {
	s := string(bytes.TrimSpace(b))
	if uq, err := strconv.Unquote(s); err == nil {
		s = strings.TrimSpace(uq)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("number: unexpected value %s", b)
	}
	*n = Number(v)
	return nil
}

// Band selects the WiFi radio(s) a command applies to. The zero value is
// BandAll. It implements pflag.Value.
type Band int

const (
	BandAll Band = iota
	Band24
	Band5
)

func (b Band) String() string {
	switch b {
	case Band24:
		return "2.4"
	case Band5:
		return "5"
	}
	return "all"
}

func (b *Band) Set(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "2.4", "2.4ghz", "24":
		*b = Band24
	case "5", "5ghz":
		*b = Band5
	case "all", "both":
		*b = BandAll
	default:
		return fmt.Errorf("invalid band %q: want 2.4, 5 or all", s)
	}
	return nil
}

func (Band) Type() string { return "2.4|5|all" }

func (b Band) Has24() bool { return b != Band5 }
func (b Band) Has5() bool  { return b != Band24 }

// ParseMAC accepts a MAC with ':' or '-' separators in any case and returns
// it upper-case with colons.
func ParseMAC(s string) (string, error) {
	t := strings.TrimSpace(s)
	if len(t) != 17 {
		return "", fmt.Errorf("invalid MAC address %q", s)
	}
	sep := t[2]
	if sep != ':' && sep != '-' {
		return "", fmt.Errorf("invalid MAC address %q", s)
	}
	out := make([]byte, 0, 17)
	for i := 0; i < 17; i++ {
		c := t[i]
		if i%3 == 2 {
			if c != sep {
				return "", fmt.Errorf("invalid MAC address %q", s)
			}
			out = append(out, ':')
			continue
		}
		switch {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'F':
		case c >= 'a' && c <= 'f':
			c -= 'a' - 'A'
		default:
			return "", fmt.Errorf("invalid MAC address %q", s)
		}
		out = append(out, c)
	}
	return string(out), nil
}

// EqualMAC compares two MACs ignoring case and separator style.
func EqualMAC(a, b string) bool {
	pa, errA := ParseMAC(a)
	pb, errB := ParseMAC(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return pa == pb
}

// decodeJSON decodes a reply body, rejecting unknown fields under
// WithStrictDecode.
func (c *Client) decodeJSON(endpoint string, body []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	if c.strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(v); err != nil {
		return &DecodeError{Endpoint: endpoint, Body: body, Err: err}
	}
	return nil
}
