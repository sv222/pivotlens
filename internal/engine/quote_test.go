package engine

import "testing"

func TestQuoteIdentifier(t *testing.T) {
	cases := map[string]string{
		"region":     `"region"`,
		`we"ird`:     `"we""ird"`,
		"a b":        `"a b"`,
		`"; DROP --`: `"""; DROP --"`,
	}
	for in, want := range cases {
		if got := qi(in); got != want {
			t.Errorf("qi(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestQuoteLiteral(t *testing.T) {
	cases := map[string]string{
		"plain":    `'plain'`,
		"it's":     `'it''s'`,
		"'; DROP ": `'''; DROP '`,
	}
	for in, want := range cases {
		if got := ql(in); got != want {
			t.Errorf("ql(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLikeEscape(t *testing.T) {
	if got, want := likeEscape(`50%_x\`), `50\%\_x\\`; got != want {
		t.Errorf("likeEscape = %q, want %q", got, want)
	}
	if got, want := likeEscape("plain"), "plain"; got != want {
		t.Errorf("likeEscape = %q, want %q", got, want)
	}
}

func TestCheckCast(t *testing.T) {
	if err := checkCast("TIMESTAMP"); err != nil {
		t.Errorf("TIMESTAMP should be allowed: %v", err)
	}
	if err := checkCast("VARCHAR); DROP TABLE src; --"); err == nil {
		t.Error("injection payload must be rejected")
	}
}

func TestAggExpr(t *testing.T) {
	cases := []struct {
		agg, col, want string
		wantErr        bool
	}{
		{"count", "", "count(*)", false},
		{"sum", "amount", `sum("amount")`, false},
		{"count_distinct", "id", `count(DISTINCT "id")`, false},
		{"median", `we"ird`, `median("we""ird")`, false},
		{"sum", "", "", true},
		{"evil(); DROP", "x", "", true},
	}
	for _, c := range cases {
		got, err := aggExpr(c.agg, c.col)
		if c.wantErr {
			if err == nil {
				t.Errorf("aggExpr(%q,%q) should fail", c.agg, c.col)
			}
			continue
		}
		if err != nil {
			t.Errorf("aggExpr(%q,%q) unexpected error %v", c.agg, c.col, err)
		}
		if got != c.want {
			t.Errorf("aggExpr(%q,%q) = %q, want %q", c.agg, c.col, got, c.want)
		}
	}
}
