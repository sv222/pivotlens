package engine

import (
	"math/big"
	"testing"
	"time"
)

func TestFormatValue(t *testing.T) {
	ts := time.Date(2026, 9, 1, 14, 2, 11, 0, time.UTC)
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "NULL"},
		{"string", "hello", "hello"},
		{"bytes", []byte("raw"), "raw"},
		{"bool", true, "true"},
		{"float", 1.5, "1.5"},
		{"float-int", 12.0, "12"},
		{"int64", int64(42), "42"},
		{"bigint", big.NewInt(9007199254740993), "9007199254740993"},
		{"timestamp", ts, "2026-09-01 14:02:11"},
		{"date", day, "2026-09-01"},
	}
	for _, c := range cases {
		if got := FormatValue(c.in); got != c.want {
			t.Errorf("%s: FormatValue(%v) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestFormatValueScrubsControlChars(t *testing.T) {
	cases := map[string]string{
		"a\nb":   "a·b",
		"a\tb":   "a·b",
		"a\r\nb": "a··b",
		"clean":  "clean",
	}
	for in, want := range cases {
		if got := FormatValue(in); got != want {
			t.Errorf("FormatValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatValueKeepsUnicode(t *testing.T) {
	if got := FormatValue("Ünicode ✓"); got != "Ünicode ✓" {
		t.Errorf("got %q", got)
	}
}
