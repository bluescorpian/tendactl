package tenda

import (
	"encoding/json"
	"testing"
)

func TestFlagUnmarshal(t *testing.T) {
	for in, want := range map[string]Flag{
		`"1"`: true, `1`: true, `"true"`: true, `true`: true,
		`"0"`: false, `0`: false, `"false"`: false, `false`: false,
	} {
		f := Flag(!want)
		if err := json.Unmarshal([]byte(in), &f); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if f != want {
			t.Fatalf("%s = %v, want %v", in, f, want)
		}
	}
	var f Flag
	for _, bad := range []string{`"2"`, `""`, `null`, `"yes"`} {
		if err := json.Unmarshal([]byte(bad), &f); err == nil {
			t.Fatalf("%s: want error", bad)
		}
	}
	if b, _ := json.Marshal(Flag(true)); string(b) != "true" {
		t.Fatalf("Marshal = %s", b)
	}
}

func TestNumberUnmarshal(t *testing.T) {
	for in, want := range map[string]Number{`0`: 0, `"0"`: 0, `2`: 2, `"2"`: 2, `19`: 19, `-1`: -1} {
		var n Number
		if err := json.Unmarshal([]byte(in), &n); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if n != want {
			t.Fatalf("%s = %d, want %d", in, n, want)
		}
	}
	var n Number
	for _, bad := range []string{`"x"`, `""`, `1.5`, `true`} {
		if err := json.Unmarshal([]byte(bad), &n); err == nil {
			t.Fatalf("%s: want error", bad)
		}
	}
}

func TestBandSet(t *testing.T) {
	for in, want := range map[string]Band{
		"2.4": Band24, "2.4GHz": Band24, "24": Band24,
		"5": Band5, "5ghz": Band5,
		"all": BandAll, "BOTH": BandAll,
	} {
		b := Band(99)
		if err := b.Set(in); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if b != want {
			t.Fatalf("%s = %v, want %v", in, b, want)
		}
	}
	var b Band
	if err := b.Set("6"); err == nil {
		t.Fatal("Set(6): want error")
	}
	if !BandAll.Has24() || !BandAll.Has5() || Band24.Has5() || Band5.Has24() {
		t.Fatal("Has24/Has5 wrong")
	}
	if Band24.String() != "2.4" || Band5.String() != "5" || BandAll.String() != "all" {
		t.Fatal("String wrong")
	}
}

func TestParseMAC(t *testing.T) {
	for in, want := range map[string]string{
		"aa:bb:cc:dd:ee:ff":   "AA:BB:CC:DD:EE:FF",
		"AA-bb-CC-dd-EE-0f":   "AA:BB:CC:DD:EE:0F",
		" 02:00:00:00:00:0f ": "02:00:00:00:00:0F",
	} {
		got, err := ParseMAC(in)
		if err != nil || got != want {
			t.Fatalf("ParseMAC(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "aa:bb:cc:dd:ee", "aa:bb-cc:dd:ee:ff", "gg:bb:cc:dd:ee:ff", "aabb.ccdd.eeff", "aa:bb:cc:dd:ee:ff:00"} {
		if _, err := ParseMAC(bad); err == nil {
			t.Fatalf("ParseMAC(%q): want error", bad)
		}
	}
	if !EqualMAC("aa:bb:cc:dd:ee:ff", "AA-BB-CC-DD-EE-FF") || EqualMAC("aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:00") {
		t.Fatal("EqualMAC wrong")
	}
}
