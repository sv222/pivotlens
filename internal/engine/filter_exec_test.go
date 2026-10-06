package engine

import (
	"context"
	"testing"
)

func metaType(t *testing.T, s *Session, name string) string {
	t.Helper()
	for _, c := range s.Columns() {
		if c.Name == name {
			return c.Type
		}
	}
	t.Fatalf("column %q not found", name)
	return ""
}

func TestTypedFilterCounts(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	ctx := context.Background()
	q := NewSpec(s.Columns())
	q.Filters = []FilterSpec{{Col: "status", Op: OpEq, Value: "FAILED", ColType: metaType(t, s, "status")}}
	n, err := s.Count(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("status = FAILED: got %d rows, want 1", n)
	}

	q.Filters = []FilterSpec{{Col: "amount", Op: OpGt, Value: "100", ColType: metaType(t, s, "amount")}}
	n, err = s.Count(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("amount > 100: got %d rows, want 3", n)
	}

	q.Filter = "EAST"
	n, err = s.Count(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("EAST and amount > 100: got %d rows, want 1", n)
	}
}

func TestTypedFilterNullPredicate(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	ctx := context.Background()
	q := NewSpec(s.Columns())
	q.Filters = []FilterSpec{{Col: "region", Op: OpNotNull}}
	n, err := s.Count(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("region is not null: got %d rows, want 5", n)
	}
}
