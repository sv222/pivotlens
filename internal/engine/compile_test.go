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

func TestBaseWhereEmpty(t *testing.T) {
	if got := demoSpec().baseWhere(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestBaseWhereDropNull(t *testing.T) {
	q := demoSpec()
	q.DropNull = []string{"region", "amount"}
	want := ` WHERE "region" IS NOT NULL AND "amount" IS NOT NULL`
	if got := q.baseWhere(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFilterWhereEmpty(t *testing.T) {
	if got := demoSpec().filterWhere(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFilterWhereOrChain(t *testing.T) {
	q := demoSpec()
	q.Filter = "US"
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%US%' ESCAPE '\'` +
		` OR coalesce(CAST("amount" AS VARCHAR), '') ILIKE '%US%' ESCAPE '\')`
	if got := q.filterWhere(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestFilterWhereEscapesWildcards(t *testing.T) {
	q := demoSpec()
	q.Cols[1].Hidden = true
	q.Filter = "50%"
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%50\%%' ESCAPE '\')`
	if got := q.filterWhere(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestFilterWhereEscapesQuote(t *testing.T) {
	q := demoSpec()
	q.Cols[1].Hidden = true
	q.Filter = "it's"
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%it''s%' ESCAPE '\')`
	if got := q.filterWhere(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

const demoPrefix = `WITH base AS (SELECT "region" AS "region", "amount" AS "amount" FROM src), ` +
	`filtered AS (SELECT * FROM base) `

func TestPageSQLDefault(t *testing.T) {
	got, err := demoSpec().PageSQL(100, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `SELECT * FROM filtered LIMIT 100 OFFSET 0`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestPageSQLSorted(t *testing.T) {
	q := demoSpec()
	q.Sort = []SortKey{{Col: "amount", Desc: true}}
	got, err := q.PageSQL(50, 200)
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `SELECT * FROM filtered ORDER BY "amount" DESC LIMIT 50 OFFSET 200`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestCountSQL(t *testing.T) {
	got, err := demoSpec().CountSQL()
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `SELECT count(*) FROM filtered`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestWidthsSQL(t *testing.T) {
	got, err := demoSpec().WidthsSQL(1000)
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `SELECT max(length(CAST("region" AS VARCHAR))), ` +
		`max(length(CAST("amount" AS VARCHAR))) FROM (SELECT * FROM filtered LIMIT 1000)`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestPivotSQL(t *testing.T) {
	q := demoSpec()
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "amount", Agg: "sum", AggCol: "amount"}
	got, err := q.PageSQL(100, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `PIVOT filtered ON "amount" USING sum("amount") GROUP BY "region" LIMIT 100 OFFSET 0`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestPivotRequiresRowsAndOn(t *testing.T) {
	q := demoSpec()
	q.Pivot = &PivotSpec{On: "amount", Agg: "count"}
	if _, err := q.PageSQL(1, 0); err == nil {
		t.Error("pivot without row groups must fail")
	}
	q.Pivot = &PivotSpec{Rows: []string{"region"}, Agg: "count"}
	if _, err := q.PageSQL(1, 0); err == nil {
		t.Error("pivot without a pivot column must fail")
	}
}

func TestWidthsSQLRejectsPivot(t *testing.T) {
	q := demoSpec()
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "amount", Agg: "count"}
	if _, err := q.WidthsSQL(10); err == nil {
		t.Error("WidthsSQL must refuse a pivot spec")
	}
}

func TestExportSQL(t *testing.T) {
	got, err := demoSpec().ExportSQL("out.parquet", "parquet")
	if err != nil {
		t.Fatal(err)
	}
	want := `COPY (` + demoPrefix + `SELECT * FROM filtered) TO 'out.parquet' (FORMAT parquet)`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	got, err = demoSpec().ExportSQL("out.csv", "csv")
	if err != nil {
		t.Fatal(err)
	}
	want = `COPY (` + demoPrefix + `SELECT * FROM filtered) TO 'out.csv' (FORMAT csv, HEADER)`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if _, err := demoSpec().ExportSQL("x.xlsx", "xlsx"); err == nil {
		t.Error("unknown export format must fail")
	}
}
