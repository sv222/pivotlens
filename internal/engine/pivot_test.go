package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func pivotRows(t *testing.T, s *Session, q QuerySpec, p PivotSpec) ([]string, [][]string) {
	t.Helper()
	q.Pivot = &p
	q, err := s.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	rows, cols, err := s.FetchPage(context.Background(), q, 1000, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cols, q.Pivot.Columns()) {
		t.Fatalf("columns %v, spec says %v", cols, q.Pivot.Columns())
	}
	return cols, rows
}

func TestPivotAggregates(t *testing.T) {
	head := []string{"region", "COMPLETED", "FAILED", "PENDING"}
	cases := []struct {
		file string
		agg  string
		col  string
		head []string
		want [][]string
	}{
		{"tiny.csv", "sum", "amount", head, [][]string{
			{"AP-SOUTH", "312.25", "NULL", "NULL"},
			{"EU-CENT", "12.5", "84", "NULL"},
			{"US-EAST", "1420.5", "NULL", "NULL"},
			{"US-WEST", "NULL", "NULL", "9120"},
		}},
		{"tiny.parquet", "count", "", head, [][]string{
			{"AP-SOUTH", "1", "0", "0"},
			{"EU-CENT", "1", "1", "0"},
			{"US-EAST", "1", "0", "0"},
			{"US-WEST", "0", "0", "1"},
		}},
		{"tiny.parquet", "max", "amount", head, [][]string{
			{"AP-SOUTH", "312.25", "NULL", "NULL"},
			{"EU-CENT", "12.5", "84", "NULL"},
			{"US-EAST", "1420.5", "NULL", "NULL"},
			{"US-WEST", "NULL", "NULL", "9120"},
		}},
		{"tiny.ndjson", "avg", "amount", []string{"region", "COMPLETED", "FAILED"}, [][]string{
			{"AP-SOUTH", "312.25", "NULL"},
			{"EU-CENT", "NULL", "84"},
			{"US-EAST", "1420.5", "NULL"},
		}},
		{"tiny.ndjson", "min", "region", []string{"region", "COMPLETED", "FAILED"}, [][]string{
			{"AP-SOUTH", "AP-SOUTH", "NULL"},
			{"EU-CENT", "NULL", "EU-CENT"},
			{"US-EAST", "US-EAST", "NULL"},
		}},
	}
	for _, c := range cases {
		t.Run(c.file+"/"+c.agg, func(t *testing.T) {
			s := openFixture(t, c.file)
			cols, rows := pivotRows(t, s, NewSpec(s.Columns()),
				PivotSpec{Rows: []string{"region"}, On: "status", Agg: c.agg, AggCol: c.col})
			if !reflect.DeepEqual(cols, c.head) {
				t.Errorf("cols = %v, want %v", cols, c.head)
			}
			if !reflect.DeepEqual(rows, c.want) {
				t.Errorf("rows = %v, want %v", rows, c.want)
			}
		})
	}
}

func TestPivotWithoutColumnField(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	cols, rows := pivotRows(t, s, NewSpec(s.Columns()),
		PivotSpec{Rows: []string{"status"}, Agg: "sum", AggCol: "amount"})
	want := [][]string{{"COMPLETED", "1745.25"}, {"FAILED", "84"}, {"PENDING", "9120"}}
	if cols[1] != "sum(amount)" || !reflect.DeepEqual(rows, want) {
		t.Errorf("cols %v rows %v, want %v", cols, rows, want)
	}
}

func TestPivotNulls(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	cols, rows := pivotRows(t, s, NewSpec(s.Columns()),
		PivotSpec{Rows: []string{"region"}, On: "status", Agg: "sum", AggCol: "amount"})
	if want := []string{"region", "FAIL", "OK", "NULL"}; !reflect.DeepEqual(cols, want) {
		t.Fatalf("cols = %v, want %v", cols, want)
	}
	want := [][]string{
		{"EU", "7", "NULL", "NULL"},
		{"US", "NULL", "13", "5"},
		{"NULL", "NULL", "NULL", "NULL"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestPivotSortsByValueColumn(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Sort = []SortKey{{Col: "COMPLETED", Desc: true}}
	_, rows := pivotRows(t, s, q, PivotSpec{Rows: []string{"region"}, On: "status", Agg: "sum", AggCol: "amount"})
	if rows[0][0] != "US-EAST" {
		t.Errorf("first row = %v, want US-EAST", rows[0])
	}
}

func TestPivotRespectsFilters(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Filters = []FilterSpec{{Col: "region", Op: OpEq, Value: "nowhere", ColType: "VARCHAR"}}
	cols, rows := pivotRows(t, s, q, PivotSpec{Rows: []string{"region"}, On: "status", Agg: "count"})
	if len(cols) != 1 || len(rows) != 0 {
		t.Errorf("empty input gave cols %v rows %v", cols, rows)
	}
}

func TestPivotLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString("k,v\n")
	for i := 0; i <= PivotLimit; i++ {
		fmt.Fprintf(&b, "a,%d\n", i)
	}
	path := filepath.Join(t.TempDir(), "wide.csv")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"k"}, On: "v", Agg: "count"}
	if _, err := s.ResolvePivot(context.Background(), q); !errors.Is(err, ErrPivotLimit) {
		t.Fatalf("err = %v, want ErrPivotLimit", err)
	}
	q.Filters = []FilterSpec{{Col: "v", Op: OpLt, Value: itoa(PivotLimit), ColType: "BIGINT"}}
	r, err := s.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Pivot.Values) != PivotLimit || r.Pivot.Values[2] != "2" {
		t.Errorf("values = %v", r.Pivot.Values)
	}
}

func TestResolvePivotUnknownColumn(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"nope"}, Agg: "count"}
	if _, err := s.ResolvePivot(context.Background(), q); err == nil {
		t.Error("unknown row field must fail")
	}
}

func TestPivotExportAndWriteCSV(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Agg: "count"}
	q, err := s.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	want := "region,FAIL,OK,NULL\nEU,1,0,0\nUS,0,2,1\n,0,1,0\n"
	var buf bytes.Buffer
	if err := s.WriteCSV(context.Background(), q, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != want {
		t.Errorf("WriteCSV = %q, want %q", buf.String(), want)
	}
	path := filepath.Join(t.TempDir(), "p.csv")
	if err := s.Export(context.Background(), q, path, "csv"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(got), "\r\n", "\n") != want {
		t.Errorf("export = %q, want %q", got, want)
	}
}

func TestParsePivot(t *testing.T) {
	p, err := ParsePivot("rows=region, cols=status,values=amount,agg=SUM")
	if err != nil {
		t.Fatal(err)
	}
	want := PivotSpec{Rows: []string{"region"}, On: "status", AggCol: "amount", Agg: "sum"}
	if !reflect.DeepEqual(p, want) {
		t.Errorf("got %+v, want %+v", p, want)
	}
	if p, err := ParsePivot("rows=region"); err != nil || p.Agg != "count" {
		t.Errorf("default agg: %+v %v", p, err)
	}
	for _, bad := range []string{"", "cols=status", "rows=a,agg=median,values=b", "rows=a,agg=sum", "rows=a,x=1", "rows"} {
		if _, err := ParsePivot(bad); err == nil {
			t.Errorf("ParsePivot(%q) must fail", bad)
		}
	}
}

func openNDJSON(t *testing.T, body string) *Session {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rows.ndjson")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestPivotEmptyStringColumnValue(t *testing.T) {
	s := openNDJSON(t, `{"region":"EU","status":"","amount":1}
{"region":"EU","status":"ok","amount":2}
{"region":"US","status":"","amount":4}
`)
	cols, rows := pivotRows(t, s, NewSpec(s.Columns()),
		PivotSpec{Rows: []string{"region"}, On: "status", Agg: "sum", AggCol: "amount"})
	if want := []string{"region", EmptyText, "ok"}; !reflect.DeepEqual(cols, want) {
		t.Fatalf("cols = %v, want %v", cols, want)
	}
	want := [][]string{{"EU", "1", "2"}, {"US", "4", "NULL"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestPivotAliases(t *testing.T) {
	names, null := pivotAliases([]string{"region"}, []string{"NULL", "OK", "ok", "region", "", "(empty)"}, true)
	want := []string{"NULL", "OK", "ok_1", "region_1", "(empty)", "(empty)_1"}
	if !reflect.DeepEqual(names, want) || null != "NULL_1" {
		t.Errorf("names = %v null = %q, want %v NULL_1", names, null, want)
	}
	if names, null := pivotAliases(nil, []string{"a"}, false); null != "" || !reflect.DeepEqual(names, []string{"a"}) {
		t.Errorf("names = %v null = %q", names, null)
	}
}

func TestPivotCollidingColumnValues(t *testing.T) {
	s := openNDJSON(t, `{"region":"EU","status":"ok","amount":1}
{"region":"EU","status":"OK","amount":2}
{"region":"US","status":"region","amount":4}
{"region":"US","status":"NULL","amount":8}
{"region":"US","status":null,"amount":16}
{"region":"US","status":"ok","amount":32}
`)
	q := NewSpec(s.Columns())
	q.Sort = []SortKey{{Col: "ok_1", Desc: true}}
	cols, rows := pivotRows(t, s, q,
		PivotSpec{Rows: []string{"region"}, On: "status", Agg: "sum", AggCol: "amount"})
	wantCols := []string{"region", "NULL", "OK", "ok_1", "region_1", "NULL_1"}
	if !reflect.DeepEqual(cols, wantCols) {
		t.Fatalf("cols = %v, want %v", cols, wantCols)
	}
	want := [][]string{
		{"US", "8", "NULL", "32", "4", "16"},
		{"EU", "NULL", "2", "1", "NULL", "NULL"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestPivotExportFormats(t *testing.T) {
	for _, file := range []string{"tiny.csv", "tiny.parquet", "tiny.ndjson"} {
		t.Run(file, func(t *testing.T) {
			s := openFixture(t, file)
			q := NewSpec(s.Columns())
			q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Agg: "count"}
			q, err := s.ResolvePivot(context.Background(), q)
			if err != nil {
				t.Fatal(err)
			}
			_, want := pivotRows(t, s, q, *q.Pivot)
			for _, format := range []string{"parquet", "ndjson"} {
				path := filepath.Join(t.TempDir(), "p."+format)
				if err := s.Export(context.Background(), q, path, format); err != nil {
					t.Fatal(err)
				}
				back := openFixturePath(t, path)
				got, cols, err := back.FetchPage(context.Background(), NewSpec(back.Columns()), 100, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(cols, q.Pivot.Columns()) || !reflect.DeepEqual(got, want) {
					t.Errorf("%s: cols %v rows %v, want %v %v", format, cols, got, q.Pivot.Columns(), want)
				}
			}
		})
	}
}

func openFixturePath(t *testing.T, path string) *Session {
	t.Helper()
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
