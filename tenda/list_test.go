package tenda

import (
	"errors"
	"slices"
	"testing"
)

func TestListFormatEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		f    ListFormat
		rows [][]string
		want string
	}{
		{"nat", TildeComma, [][]string{{"192.168.0.5", "80", "80", "1"}, {"192.168.0.6", "443", "8443", "0"}}, "192.168.0.5,80,80,1~192.168.0.6,443,8443,0"},
		{"route single", TildeComma, [][]string{{"10.0.0.0", "255.0.0.0", "192.168.0.2", "br0"}}, "10.0.0.0,255.0.0.0,192.168.0.2,br0"},
		{"pptp trailing empties", TildeSemi, [][]string{{"u", "p", "1", "0", "", "", ""}, {"v", "q", "0", "0", "", "", ""}}, "u;p;1;0;;;~v;q;0;0;;;"},
		{"line cr", LineCR, [][]string{{"Phone", "AA:BB:CC:DD:EE:FF", "256", "128"}, {"", "AA:BB:CC:DD:EE:00", "0", "0"}}, "Phone\rAA:BB:CC:DD:EE:FF\r256\r128\n\rAA:BB:CC:DD:EE:00\r0\r0"},
		{"no rows", TildeComma, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.f.Encode(tt.rows)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Encode = %q, want %q", got, tt.want)
			}
			back := tt.f.Decode(got)
			if !slices.EqualFunc(back, tt.rows, slices.Equal) {
				t.Fatalf("Decode = %q, want %q", back, tt.rows)
			}
		})
	}
}

func TestListFormatDecodeEmpty(t *testing.T) {
	if got := LineCR.Decode(""); got != nil {
		t.Fatalf("Decode(\"\") = %q, want nil", got)
	}
}

func TestListFormatSeparatorError(t *testing.T) {
	tests := []struct {
		f    ListFormat
		cell string
	}{
		{TildeComma, "a~b"}, {TildeComma, "a,b"},
		{TildeSemi, "a;b"}, {TildeSemi, "a~b"},
		{LineCR, "a\rb"}, {LineCR, "a\nb"},
	}
	for _, tt := range tests {
		_, err := tt.f.Encode([][]string{{"ok", "ok"}, {"ok", tt.cell}})
		var se *SeparatorError
		if !errors.As(err, &se) {
			t.Fatalf("Encode(%q) err = %v, want *SeparatorError", tt.cell, err)
		}
		if se.Row != 1 || se.Col != 1 || se.Value != tt.cell {
			t.Fatalf("SeparatorError = %+v", se)
		}
	}
}

func FuzzListFormatRoundTrip(f *testing.F) {
	f.Add("a", "b", "c", "")
	f.Add("u", "p", "", "")
	f.Add("x~y", ",", "\r", "\n")
	f.Fuzz(func(t *testing.T, a, b, c, d string) {
		for _, lf := range []ListFormat{TildeComma, TildeSemi, LineCR} {
			rows := [][]string{{a, b}, {c, d}}
			s, err := lf.Encode(rows)
			if err != nil {
				continue
			}
			if got := lf.Decode(s); !slices.EqualFunc(got, rows, slices.Equal) {
				t.Fatalf("%q: Decode(Encode(%q)) = %q", lf, rows, got)
			}
		}
	})
}
