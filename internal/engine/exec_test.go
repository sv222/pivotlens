package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchPageReturnsRowsAndColumns(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	rows, cols, err := s.FetchPage(context.Background(), q, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 4 {
		t.Fatalf("got %d columns, want 4", len(cols))
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0][0] != "US-EAST" {
		t.Errorf("first cell = %q, want US-EAST", rows[0][0])
	}
}

func TestFetchPageOffsetIsStable(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	ctx := context.Background()
	all, _, err := s.FetchPage(ctx, q, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := range all {
		one, _, err := s.FetchPage(ctx, q, 1, i)
		if err != nil {
			t.Fatal(err)
		}
		if len(one) != 1 || one[0][0] != all[i][0] {
			t.Fatalf("row %d unstable across offsets: %v vs %v", i, one, all[i])
		}
	}
}

func TestCount(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	n, err := s.Count(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("count = %d, want 5", n)
	}
}

func TestCountWithFilter(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Filter = "EU-CENT"
	n, err := s.Count(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("filtered count = %d, want 2", n)
	}
}

func TestSortChangesOrder(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Sort = []SortKey{{Col: "amount", Desc: true}}
	rows, _, err := s.FetchPage(context.Background(), q, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][0] != "US-WEST" {
		t.Errorf("top row by amount DESC = %q, want US-WEST", rows[0][0])
	}
}

func TestWidths(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	w, err := s.Widths(context.Background(), q, WidthSample)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) != 4 {
		t.Fatalf("got %d widths, want 4", len(w))
	}
	if w[0] != 8 {
		t.Errorf("region width = %d, want 8", w[0])
	}
}

func TestPivotExecutes(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Agg: "sum", AggCol: "amount"}
	rows, cols, err := s.FetchPage(context.Background(), q, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) < 2 {
		t.Fatalf("pivot produced %d columns", len(cols))
	}
	if len(rows) != 4 {
		t.Fatalf("pivot produced %d rows, want 4", len(rows))
	}
}

func TestWidthsForPivot(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Agg: "count"}
	w, err := s.Widths(context.Background(), q, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(w) == 0 {
		t.Fatal("pivot widths must be measured from results")
	}
	for i, n := range w {
		if n <= 0 {
			t.Errorf("width %d is %d, want > 0", i, n)
		}
	}
}

func TestEdgeCaseValues(t *testing.T) {
	s := openFixture(t, "edge.csv")
	q := NewSpec(s.Columns())
	rows, _, err := s.FetchPage(context.Background(), q, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for r, row := range rows {
		for c, cell := range row {
			for _, ch := range cell {
				if isCtrl(ch) {
					t.Fatalf("row %d col %d contains a control character: %q", r, c, cell)
				}
			}
		}
	}
	if rows[1][1] != NullText {
		t.Errorf("empty quoted field should be NULL, got %q", rows[1][1])
	}
}

func TestExport(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Filter = "EU-CENT"
	out := filepath.Join(t.TempDir(), "out.parquet")
	if err := s.Export(context.Background(), q, out, "parquet"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Error("exported file is empty")
	}
}
