package engine

import "testing"

func TestOpString(t *testing.T) {
	cases := map[Op]string{
		OpEq: "=", OpNe: "!=", OpGt: ">", OpGe: ">=", OpLt: "<", OpLe: "<=",
		OpContains: "contains", OpStarts: "starts with", OpEnds: "ends with",
		OpNull: "is null", OpNotNull: "is not null",
	}
	for op, want := range cases {
		if got := op.String(); got != want {
			t.Errorf("Op(%d).String() = %q, want %q", op, got, want)
		}
	}
}

func TestFilterSpecString(t *testing.T) {
	f := FilterSpec{Col: "status", Op: OpEq, Value: "FAILED", ColType: "VARCHAR"}
	if got, want := f.String(), "status = FAILED"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	n := FilterSpec{Col: "amount", Op: OpNull}
	if got, want := n.String(), "amount is null"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestOpsFor(t *testing.T) {
	text := OpsFor("VARCHAR")
	has := func(ops []Op, op Op) bool {
		for _, o := range ops {
			if o == op {
				return true
			}
		}
		return false
	}
	if !has(text, OpContains) || has(text, OpGt) {
		t.Errorf("text ops wrong: %v", text)
	}
	num := OpsFor("DOUBLE")
	if !has(num, OpGt) || has(num, OpContains) {
		t.Errorf("numeric ops wrong: %v", num)
	}
	if len(text) != 7 || len(num) != 8 {
		t.Errorf("op counts: text %d, numeric %d", len(text), len(num))
	}
}

func TestCloneCopiesFilters(t *testing.T) {
	q := NewSpec([]ColumnMeta{{Name: "a", Type: "BIGINT"}})
	q.Filters = []FilterSpec{{Col: "a", Op: OpGt, Value: "1", ColType: "BIGINT"}}
	n := q.Clone()
	n.Filters[0].Value = "2"
	if q.Filters[0].Value != "1" {
		t.Error("Clone shares the Filters slice")
	}
}

func TestPredGolden(t *testing.T) {
	cases := []struct {
		name string
		f    FilterSpec
		want string
	}{
		{"eq numeric", FilterSpec{Col: "amount", Op: OpEq, Value: "84", ColType: "DOUBLE"},
			`CAST("amount" AS DOUBLE) = CAST('84' AS DOUBLE)`},
		{"ge date", FilterSpec{Col: "created_at", Op: OpGe, Value: "2026-09-01", ColType: "TIMESTAMP"},
			`CAST("created_at" AS TIMESTAMP) >= CAST('2026-09-01' AS TIMESTAMP)`},
		{"eq empty coltype falls back to varchar", FilterSpec{Col: "status", Op: OpEq, Value: "FAILED"},
			`CAST("status" AS VARCHAR) = CAST('FAILED' AS VARCHAR)`},
		{"contains escaped", FilterSpec{Col: "status", Op: OpContains, Value: `50%\_`},
			`CAST("status" AS VARCHAR) ILIKE '%50\%\\\_%' ESCAPE '\'`},
		{"starts", FilterSpec{Col: "region", Op: OpStarts, Value: "US"},
			`CAST("region" AS VARCHAR) ILIKE 'US%' ESCAPE '\'`},
		{"ends quote escaped", FilterSpec{Col: "name", Op: OpEnds, Value: "o'brien"},
			`CAST("name" AS VARCHAR) ILIKE '%o''brien' ESCAPE '\'`},
		{"null", FilterSpec{Col: "amount", Op: OpNull}, `"amount" IS NULL`},
		{"not null", FilterSpec{Col: "amount", Op: OpNotNull}, `"amount" IS NOT NULL`},
	}
	for _, c := range cases {
		got, err := c.f.pred()
		if err != nil {
			t.Fatalf("%s: pred error: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

func TestPredRejectsBadColType(t *testing.T) {
	f := FilterSpec{Col: "a", Op: OpGt, Value: "1", ColType: `DOUBLE); DROP TABLE src`}
	if _, err := f.pred(); err == nil {
		t.Fatal("bad ColType accepted")
	}
	q := NewSpec([]ColumnMeta{{Name: "a", Type: "DOUBLE"}})
	q.Filters = []FilterSpec{f}
	if _, err := q.PageSQL(10, 0); err == nil {
		t.Fatal("bad ColType not surfaced through PageSQL")
	}
}

func fixtureSpec() QuerySpec {
	return NewSpec([]ColumnMeta{
		{Name: "region", Type: "VARCHAR"},
		{Name: "status", Type: "VARCHAR"},
	})
}

func TestFilterWhereCombinesSubstringAndTyped(t *testing.T) {
	q := fixtureSpec()
	q.Filter = "EAST"
	q.Filters = []FilterSpec{{Col: "status", Op: OpEq, Value: "FAILED", ColType: "VARCHAR"}}
	got, err := q.filterWhere()
	if err != nil {
		t.Fatal(err)
	}
	want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%EAST%' ESCAPE '\' OR coalesce(CAST("status" AS VARCHAR), '') ILIKE '%EAST%' ESCAPE '\') AND CAST("status" AS VARCHAR) = CAST('FAILED' AS VARCHAR)`
	if got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
}

func TestFilterWhereTypedOnlyAndUnchanged(t *testing.T) {
	q := fixtureSpec()
	q.Filters = []FilterSpec{{Col: "status", Op: OpNe, Value: "FAILED", ColType: "VARCHAR"}}
	got, err := q.filterWhere()
	if err != nil {
		t.Fatal(err)
	}
	if want := ` WHERE CAST("status" AS VARCHAR) != CAST('FAILED' AS VARCHAR)`; got != want {
		t.Errorf("typed only:\n got %s\nwant %s", got, want)
	}
	r := fixtureSpec()
	r.Filter = "EAST"
	got, err = r.filterWhere()
	if err != nil {
		t.Fatal(err)
	}
	if want := ` WHERE (coalesce(CAST("region" AS VARCHAR), '') ILIKE '%EAST%' ESCAPE '\' OR coalesce(CAST("status" AS VARCHAR), '') ILIKE '%EAST%' ESCAPE '\')`; got != want {
		t.Errorf("substring only changed shape:\n got %s\nwant %s", got, want)
	}
}

func TestValidType(t *testing.T) {
	for _, ok := range []string{"DOUBLE", "DECIMAL(18,2)", "TIMESTAMP WITH TIME ZONE", "BIGINT"} {
		if !validType(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", `DOUBLE); DROP`, "double", "VARCHAR'"} {
		if validType(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
