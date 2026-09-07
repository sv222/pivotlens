package engine

import "testing"

func demoSpec() QuerySpec {
	return NewSpec([]ColumnMeta{
		{Name: "region", Type: "VARCHAR"},
		{Name: "amount", Type: "DOUBLE"},
	})
}

func TestColExpr(t *testing.T) {
	cases := []struct {
		name string
		in   ColSpec
		want string
	}{
		{"plain", ColSpec{Name: "region"}, `"region"`},
		{"trim", ColSpec{Name: "region", Trim: true}, `trim("region")`},
		{"nullas", ColSpec{Name: "region", NullAs: "n/a"}, `coalesce("region", 'n/a')`},
		{"cast", ColSpec{Name: "amount", Cast: "BIGINT"}, `CAST("amount" AS BIGINT)`},
		{"all", ColSpec{Name: "amount", Trim: true, NullAs: "0", Cast: "BIGINT"},
			`CAST(coalesce(trim("amount"), '0') AS BIGINT)`},
	}
	for _, c := range cases {
		got, err := colExpr(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestColExprRejectsBadCast(t *testing.T) {
	if _, err := colExpr(ColSpec{Name: "a", Cast: "INT); DROP TABLE src; --"}); err == nil {
		t.Error("bad cast must be rejected")
	}
}

func TestSelectList(t *testing.T) {
	q := demoSpec()
	got, err := q.selectList()
	if err != nil {
		t.Fatal(err)
	}
	want := `"region" AS "region", "amount" AS "amount"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSelectListUsesAliasAndSkipsHidden(t *testing.T) {
	q := demoSpec()
	q.Cols[0].Alias = "Region"
	q.Cols[1].Hidden = true
	got, err := q.selectList()
	if err != nil {
		t.Fatal(err)
	}
	want := `"region" AS "Region"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSelectListNoVisibleColumns(t *testing.T) {
	q := demoSpec()
	q.Cols[0].Hidden = true
	q.Cols[1].Hidden = true
	if _, err := q.selectList(); err == nil {
		t.Error("all-hidden spec must be an error")
	}
}

func TestCloneIsDeep(t *testing.T) {
	q := demoSpec()
	q.Sort = []SortKey{{Col: "region"}}
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Agg: "sum", AggCol: "amount"}
	c := q.Clone()
	c.Cols[0].Alias = "changed"
	c.Sort[0].Desc = true
	c.Pivot.Rows[0] = "other"
	if q.Cols[0].Alias != "" {
		t.Error("Cols aliased with original")
	}
	if q.Sort[0].Desc {
		t.Error("Sort aliased with original")
	}
	if q.Pivot.Rows[0] != "region" {
		t.Error("Pivot.Rows aliased with original")
	}
}
