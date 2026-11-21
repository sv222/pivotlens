package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func freqRows(t *testing.T, s *Session, q QuerySpec) [][]string {
	t.Helper()
	rows, cols, err := s.FetchPage(context.Background(), q, 5000, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cols, q.Freq.Columns()) {
		t.Fatalf("columns %v, spec says %v", cols, q.Freq.Columns())
	}
	return rows
}

func TestFreqCounts(t *testing.T) {
	complete := [][]string{
		{"COMPLETED", "3", "60", "60"},
		{"FAILED", "1", "20", "80"},
		{"PENDING", "1", "20", "100"},
	}
	cases := []struct {
		file string
		col  string
		want [][]string
	}{
		{"tiny.csv", "status", complete},
		{"tiny.parquet", "status", complete},
		{"tiny.ndjson", "region", [][]string{
			{"AP-SOUTH", "1", "33.33", "33.33"},
			{"EU-CENT", "1", "33.33", "66.67"},
			{"US-EAST", "1", "33.33", "100"},
		}},
	}
	for _, c := range cases {
		t.Run(c.file+"/"+c.col, func(t *testing.T) {
			s := openFixture(t, c.file)
			q := NewSpec(s.Columns())
			q.Freq = &FreqSpec{Col: c.col, Top: DefaultTop}
			if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, c.want) {
				t.Errorf("rows = %v, want %v", rows, c.want)
			}
		})
	}
}

func TestFreqNullsSortBeforeTextOnTies(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "region", Top: DefaultTop}
	want := [][]string{
		{"US", "3", "60", "60"},
		{FreqNullText, "1", "20", "80"},
		{"EU", "1", "20", "100"},
	}
	if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestFreqEmptyString(t *testing.T) {
	body := "{\"v\":\"x\"}\n{\"v\":\"\"}\n{\"v\":null}\n{\"v\":\"x\"}\n"
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.ndjson")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s := openInput(t, Input{Paths: []string{p}})
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "v", Top: DefaultTop}
	want := [][]string{
		{"x", "2", "50", "50"},
		{FreqEmptyText, "1", "25", "75"},
		{FreqNullText, "1", "25", "100"},
	}
	if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func twelveValues(t *testing.T) *Session {
	t.Helper()
	var b strings.Builder
	b.WriteString("v\n")
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "v%02d\n", i)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "many.csv")
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return openInput(t, Input{Paths: []string{p}})
}

func TestFreqTopShowsOtherRow(t *testing.T) {
	s := twelveValues(t)
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "v", Top: 5}
	want := [][]string{
		{"v00", "1", "8.33", "8.33"},
		{"v01", "1", "8.33", "16.67"},
		{"v02", "1", "8.33", "25"},
		{"v03", "1", "8.33", "33.33"},
		{"v04", "1", "8.33", "41.67"},
		{FreqOtherText, "7", "58.33", "100"},
	}
	if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
	n, err := s.Count(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Errorf("count = %d, want 6 rows", n)
	}
}

func TestFreqTopZeroMeansAll(t *testing.T) {
	s := twelveValues(t)
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "v", Top: 0}
	rows := freqRows(t, s, q)
	if len(rows) != 12 || rows[0][0] != "v00" || rows[11][0] != "v11" {
		t.Errorf("rows = %v, want v00 to v11", rows)
	}
	if len(rows[0]) != 4 {
		t.Errorf("row width = %d, want 4", len(rows[0]))
	}
}

func TestFreqRespectsFilters(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Filters = []FilterSpec{{Col: "status", Op: OpEq, Value: "COMPLETED", ColType: "VARCHAR"}}
	q.Freq = &FreqSpec{Col: "region", Top: DefaultTop}
	want := [][]string{
		{"AP-SOUTH", "1", "33.33", "33.33"},
		{"EU-CENT", "1", "33.33", "66.67"},
		{"US-EAST", "1", "33.33", "100"},
	}
	if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestFreqMultiFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.csv")
	b := filepath.Join(dir, "b.csv")
	if err := os.WriteFile(a, []byte("region\nUS\nUS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("region\nEU\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := openInput(t, Input{Paths: []string{a, b}})
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "region", Top: DefaultTop}
	want := [][]string{
		{"US", "2", "50", "50"},
		{FreqNullText, "1", "25", "75"},
		{"EU", "1", "25", "100"},
	}
	if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestFreqUserSortKeepsCumulativePerRow(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	q := NewSpec(s.Columns())
	q.Sort = []SortKey{{Col: "value"}}
	q.Freq = &FreqSpec{Col: "region", Top: DefaultTop}
	want := [][]string{
		{FreqNullText, "1", "20", "80"},
		{"EU", "1", "20", "100"},
		{"US", "3", "60", "60"},
	}
	if rows := freqRows(t, s, q); !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestFreqStats(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "status", Top: DefaultTop}
	st, err := s.FreqStats(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 5 || st.Distinct != 3 {
		t.Errorf("stats = %+v, want total 5 distinct 3", st)
	}
}

func TestFreqCountMatchesResultRows(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Freq = &FreqSpec{Col: "status", Top: DefaultTop}
	n, err := s.Count(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
}

func TestResolveFreq(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	if err := q.ResolveFreq("region"); err != nil {
		t.Errorf("region must resolve: %v", err)
	}
	err := q.ResolveFreq("nope")
	if err == nil || !strings.Contains(err.Error(), `unknown column "nope"`) {
		t.Fatalf("err = %v, want unknown column message", err)
	}
	if !strings.Contains(err.Error(), "region") {
		t.Errorf("err = %v, want the error to list the columns", err)
	}
}

func TestFreqStatsSQLRefusedWithoutSpec(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	if _, err := s.FreqStats(context.Background(), q); err == nil {
		t.Error("freq stats without a spec must be refused")
	}
	q.Freq = &FreqSpec{Col: "region", Top: DefaultTop}
	if _, err := q.StatsSQL("region", false); !errors.Is(err, ErrPivotStats) {
		t.Errorf("stats on a value counts view = %v, want ErrPivotStats", err)
	}
}
