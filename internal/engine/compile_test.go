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
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "sum", Col: "amount"}}}
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
	got, err := demoSpec().filterWhere()
	if got != "" || err != nil {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFilterWhereOrChain(t *testing.T) {
	q := demoSpec()
	q.Filter = "US"
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%US%' ESCAPE '\'` +
		` OR coalesce(CAST("amount" AS VARCHAR), '') ILIKE '%US%' ESCAPE '\')`
	got, err := q.filterWhere()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestFilterWhereEscapesWildcards(t *testing.T) {
	q := demoSpec()
	q.Cols[1].Hidden = true
	q.Filter = "50%"
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%50\%%' ESCAPE '\')`
	got, err := q.filterWhere()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestFilterWhereEscapesQuote(t *testing.T) {
	q := demoSpec()
	q.Cols[1].Hidden = true
	q.Filter = "it's"
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%it''s%' ESCAPE '\')`
	got, err := q.filterWhere()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
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
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "amount", Measures: []Measure{{Agg: "sum", Col: "amount"}}, Keys: []string{"1", "it's"}, Null: true}
	got, err := q.PageSQL(100, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `SELECT p."region", p."1", p."it's", p."NULL" FROM (SELECT "region", ` +
		`sum("amount") FILTER (WHERE CAST("amount" AS VARCHAR) = '1') AS "1", ` +
		`sum("amount") FILTER (WHERE CAST("amount" AS VARCHAR) = 'it''s') AS "it's", ` +
		`sum("amount") FILTER (WHERE "amount" IS NULL) AS "NULL" ` +
		`FROM filtered GROUP BY "region") p ORDER BY p."region" LIMIT 100 OFFSET 0`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestPivotSQLWithoutColumnField(t *testing.T) {
	q := demoSpec()
	q.Pivot = &PivotSpec{Rows: []string{"region"}, Measures: []Measure{{Agg: "count"}}}
	q.Sort = []SortKey{{Col: "count", Desc: true}}
	got, err := q.PageSQL(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := demoPrefix + `SELECT p."region", p."count" FROM (SELECT "region", count(*) AS "count" ` +
		`FROM filtered GROUP BY "region") p ORDER BY p."count" DESC LIMIT 10 OFFSET 0`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestPivotRequiresRows(t *testing.T) {
	q := demoSpec()
	q.Pivot = &PivotSpec{On: "amount", Measures: []Measure{{Agg: "count"}}}
	if _, err := q.PageSQL(1, 0); err == nil {
		t.Error("pivot without a row field must fail")
	}
}

func TestWidthsSQLRejectsPivot(t *testing.T) {
	q := demoSpec()
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "amount", Measures: []Measure{{Agg: "count"}}}
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
	got, err = demoSpec().ExportSQL("out.ndjson", "ndjson")
	if err != nil {
		t.Fatal(err)
	}
	want = `COPY (` + demoPrefix + `SELECT * FROM filtered) TO 'out.ndjson' (FORMAT json)`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if _, err := demoSpec().ExportSQL("x.xlsx", "xlsx"); err == nil {
		t.Error("unknown export format must fail")
	}
}
