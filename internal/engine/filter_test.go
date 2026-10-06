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
