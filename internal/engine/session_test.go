package engine

import (
	"context"
	"path/filepath"
	"testing"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func openFixture(t *testing.T, name string) *Session {
	t.Helper()
	s, err := Open(context.Background(), fixture(name))
	if err != nil {
		t.Fatalf("Open(%s): %v", name, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenCSVSchema(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	cols := s.Columns()
	if len(cols) != 4 {
		t.Fatalf("got %d columns, want 4", len(cols))
	}
	want := []string{"region", "status", "amount", "created_at"}
	for i, w := range want {
		if cols[i].Name != w {
			t.Errorf("column %d = %q, want %q", i, cols[i].Name, w)
		}
	}
	if cols[2].Type == "" {
		t.Error("column type must not be empty")
	}
}

func TestOpenParquetSchema(t *testing.T) {
	s := openFixture(t, "tiny.parquet")
	if len(s.Columns()) != 4 {
		t.Fatalf("got %d columns, want 4", len(s.Columns()))
	}
}

func TestOpenNDJSONSchema(t *testing.T) {
	s := openFixture(t, "tiny.ndjson")
	if len(s.Columns()) != 3 {
		t.Fatalf("got %d columns, want 3", len(s.Columns()))
	}
}

func TestOpenMissingFile(t *testing.T) {
	if _, err := Open(context.Background(), fixture("nope.csv")); err == nil {
		t.Error("missing file must be an error")
	}
}
