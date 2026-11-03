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
				PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: c.agg, Col: c.col}}})
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
		PivotSpec{Rows: []string{"status"}, Measures: []Measure{{Agg: "sum", Col: "amount"}}})
	want := [][]string{{"COMPLETED", "1745.25"}, {"FAILED", "84"}, {"PENDING", "9120"}}
	if cols[1] != "sum(amount)" || !reflect.DeepEqual(rows, want) {
		t.Errorf("cols %v rows %v, want %v", cols, rows, want)
	}
}

func TestPivotNulls(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	cols, rows := pivotRows(t, s, NewSpec(s.Columns()),
		PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "sum", Col: "amount"}}})
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
	_, rows := pivotRows(t, s, q, PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "sum", Col: "amount"}}})
	if rows[0][0] != "US-EAST" {
		t.Errorf("first row = %v, want US-EAST", rows[0])
	}
}

func TestPivotRespectsFilters(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Filters = []FilterSpec{{Col: "region", Op: OpEq, Value: "nowhere", ColType: "VARCHAR"}}
	cols, rows := pivotRows(t, s, q, PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "count"}}})
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
	q.Pivot = &PivotSpec{Rows: []string{"k"}, On: "v", Measures: []Measure{{Agg: "count"}}}
	if _, err := s.ResolvePivot(context.Background(), q); !errors.Is(err, ErrPivotLimit) {
		t.Fatalf("err = %v, want ErrPivotLimit", err)
	}
	q.Filters = []FilterSpec{{Col: "v", Op: OpLt, Value: itoa(PivotLimit), ColType: "BIGINT"}}
	r, err := s.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if cols := r.Pivot.Columns(); len(cols) != PivotLimit+1 || cols[3] != "2" {
		t.Errorf("columns = %v", cols)
	}
}

func TestResolvePivotUnknownColumn(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"nope"}, Measures: []Measure{{Agg: "count"}}}
	if _, err := s.ResolvePivot(context.Background(), q); err == nil {
		t.Error("unknown row field must fail")
	}
}

func TestPivotExportAndWriteCSV(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "count"}}}
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
	want := PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "sum", Col: "amount"}}}
	if !reflect.DeepEqual(p, want) {
		t.Errorf("got %+v, want %+v", p, want)
	}
	if p, err := ParsePivot("rows=region"); err != nil || !reflect.DeepEqual(p.Measures, []Measure{{Agg: "count"}}) {
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
		PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "sum", Col: "amount"}}})
	if want := []string{"region", EmptyText, "ok"}; !reflect.DeepEqual(cols, want) {
		t.Fatalf("cols = %v, want %v", cols, want)
	}
	want := [][]string{{"EU", "1", "2"}, {"US", "4", "NULL"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestPivotColumnNames(t *testing.T) {
	p := PivotSpec{Rows: []string{"region"}, On: "s", Measures: []Measure{{Agg: "count"}},
		Keys: []string{"NULL", "OK", "ok", "region", "", "(empty)"}, Null: true}
	want := []string{"region", "NULL", "OK", "ok_1", "region_1", "(empty)", "(empty)_1", "NULL_1"}
	if got := p.Columns(); !reflect.DeepEqual(got, want) {
		t.Errorf("columns = %v, want %v", got, want)
	}
	p = PivotSpec{Rows: []string{"count"}, Measures: []Measure{{Agg: "count"}, {Agg: "sum", Col: "a"}}}
	if got, want := p.Columns(), []string{"count", "count_1", "sum(a)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("columns = %v, want %v", got, want)
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
		PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "sum", Col: "amount"}}})
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
			q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "count"}}}
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

func TestPivotSeveralValues(t *testing.T) {
	ms := []Measure{{Agg: "sum", Col: "amount"}, {Agg: "count"}}
	head := []string{"region", "COMPLETED sum(amount)", "COMPLETED count", "FAILED sum(amount)", "FAILED count"}
	want := map[string][][]string{
		"tiny.csv": {
			{"AP-SOUTH", "312.25", "1", "NULL", "0"},
			{"EU-CENT", "12.5", "1", "84", "1"},
			{"US-EAST", "1420.5", "1", "NULL", "0"},
			{"US-WEST", "NULL", "0", "NULL", "0"},
		},
		"tiny.ndjson": {
			{"AP-SOUTH", "312.25", "1", "NULL", "0"},
			{"EU-CENT", "NULL", "0", "84", "1"},
			{"US-EAST", "1420.5", "1", "NULL", "0"},
		},
	}
	want["tiny.parquet"] = want["tiny.csv"]
	for file, rows := range want {
		t.Run(file, func(t *testing.T) {
			s := openFixture(t, file)
			cols, got := pivotRows(t, s, NewSpec(s.Columns()), PivotSpec{Rows: []string{"region"}, On: "status", Measures: ms})
			if !reflect.DeepEqual(cols[:len(head)], head) {
				t.Errorf("cols = %v, want prefix %v", cols, head)
			}
			for i := range got {
				got[i] = got[i][:len(head)]
			}
			if !reflect.DeepEqual(got, rows) {
				t.Errorf("rows = %v, want %v", got, rows)
			}
		})
	}
}

func TestParsePivotSeveralValues(t *testing.T) {
	p, err := ParsePivot("rows=region,values=amount:sum, amount: AVG,*:count,a:b,cols=status")
	if err != nil {
		t.Fatal(err)
	}
	want := []Measure{{Agg: "sum", Col: "amount"}, {Agg: "avg", Col: "amount"}, {Agg: "count"}, {Agg: "count", Col: "a:b"}}
	if !reflect.DeepEqual(p.Measures, want) || p.On != "status" {
		t.Errorf("got %+v, want measures %v", p, want)
	}
	if p, err := ParsePivot("rows=r,values=x,y,agg=max"); err != nil || !reflect.DeepEqual(p.Measures, []Measure{{Agg: "max", Col: "x"}, {Agg: "max", Col: "y"}}) {
		t.Errorf("agg default: %+v %v", p, err)
	}
	for _, bad := range []string{"rows=r,values=*:sum", "rows=r,x,values=a"} {
		if _, err := ParsePivot(bad); err == nil {
			t.Errorf("ParsePivot(%q) must fail", bad)
		}
	}
}

func TestPivotTotals(t *testing.T) {
	head := []string{"region", "COMPLETED", "FAILED", "PENDING", "Total"}
	cases := []struct {
		file string
		head []string
		want [][]string
	}{
		{"tiny.csv", head, [][]string{
			{"AP-SOUTH", "312.25", "NULL", "NULL", "312.25"},
			{"EU-CENT", "12.5", "84", "NULL", "96.5"},
			{"US-EAST", "1420.5", "NULL", "NULL", "1420.5"},
			{"US-WEST", "NULL", "NULL", "9120", "9120"},
			{"Total", "1745.25", "84", "9120", "10949.25"},
		}},
		{"tiny.parquet", head, nil},
		{"tiny.ndjson", []string{"region", "COMPLETED", "FAILED", "Total"}, [][]string{
			{"AP-SOUTH", "312.25", "NULL", "312.25"},
			{"EU-CENT", "NULL", "84", "84"},
			{"US-EAST", "1420.5", "NULL", "1420.5"},
			{"Total", "1732.75", "84", "1816.75"},
		}},
	}
	cases[1].want = cases[0].want
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			s := openFixture(t, c.file)
			cols, rows := pivotRows(t, s, NewSpec(s.Columns()), PivotSpec{Rows: []string{"region"}, On: "status",
				Measures: []Measure{{Agg: "sum", Col: "amount"}}, Totals: true})
			if !reflect.DeepEqual(cols, c.head) || !reflect.DeepEqual(rows, c.want) {
				t.Errorf("cols %v rows %v, want %v %v", cols, rows, c.head, c.want)
			}
		})
	}
}

func TestPivotTotalsSortedAndExported(t *testing.T) {
	s := openFixture(t, "nulls.csv")
	q := NewSpec(s.Columns())
	q.Sort = []SortKey{{Col: "Total count", Desc: true}, {Col: "region"}}
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Totals: true,
		Measures: []Measure{{Agg: "avg", Col: "amount"}, {Agg: "count"}}}
	q, err := s.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	want := "region,FAIL avg(amount),FAIL count,OK avg(amount),OK count,NULL avg(amount),NULL count,Total avg(amount),Total count\n" +
		"US,,0,6.5,2,5,1,6,3\n" +
		"EU,7,1,,0,,0,7,1\n" +
		",,0,,1,,0,,1\n" +
		"Total,7,1,6.5,3,5,1,6.25,5\n"
	var buf bytes.Buffer
	if err := s.WriteCSV(context.Background(), q, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != want {
		t.Errorf("WriteCSV = %q, want %q", buf.String(), want)
	}
	for _, format := range []string{"csv", "parquet", "ndjson"} {
		path := filepath.Join(t.TempDir(), "t."+format)
		if err := s.Export(context.Background(), q, path, format); err != nil {
			t.Fatal(err)
		}
		back := openFixturePath(t, path)
		rows, _, err := back.FetchPage(context.Background(), NewSpec(back.Columns()), 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 4 || rows[3][0] != "Total" || rows[3][8] != "5" {
			t.Errorf("%s export rows = %v", format, rows)
		}
	}
}

func TestParsePivotTotals(t *testing.T) {
	if p, err := ParsePivot("rows=r,totals=true,percent=col"); err != nil || !p.Totals || p.Percent != "col" {
		t.Errorf("totals: %+v %v", p, err)
	}
	for _, bad := range []string{"rows=r,totals=maybe", "rows=r,percent=half"} {
		if _, err := ParsePivot(bad); err == nil {
			t.Errorf("ParsePivot(%q) must fail", bad)
		}
	}
}

func TestPivotPercent(t *testing.T) {
	cases := []struct {
		pct    string
		totals bool
		want   [][]string
	}{
		{"row", false, [][]string{
			{"AP-SOUTH", "100", "NULL"},
			{"EU-CENT", "NULL", "100"},
			{"US-EAST", "100", "NULL"},
		}},
		{"col", true, [][]string{
			{"AP-SOUTH", "18.02", "NULL", "17.19"},
			{"EU-CENT", "NULL", "100", "4.62"},
			{"US-EAST", "81.98", "NULL", "78.19"},
			{"Total", "100", "100", "100"},
		}},
		{"total", true, [][]string{
			{"AP-SOUTH", "17.19", "NULL", "17.19"},
			{"EU-CENT", "NULL", "4.62", "4.62"},
			{"US-EAST", "78.19", "NULL", "78.19"},
			{"Total", "95.38", "4.62", "100"},
		}},
	}
	for _, c := range cases {
		t.Run(c.pct, func(t *testing.T) {
			s := openFixture(t, "tiny.ndjson")
			_, rows := pivotRows(t, s, NewSpec(s.Columns()), PivotSpec{Rows: []string{"region"}, On: "status",
				Measures: []Measure{{Agg: "sum", Col: "amount"}}, Totals: c.totals, Percent: c.pct})
			if !reflect.DeepEqual(rows, c.want) {
				t.Errorf("rows = %v, want %v", rows, c.want)
			}
		})
	}
}

func TestPivotPercentFixtures(t *testing.T) {
	want := [][]string{
		{"AP-SOUTH", "25", "2.87"},
		{"EU-CENT", "25", "0.12"},
		{"US-EAST", "25", "13.07"},
		{"US-WEST", "25", "83.94"},
	}
	for _, file := range []string{"tiny.csv", "tiny.parquet"} {
		t.Run(file, func(t *testing.T) {
			s := openFixture(t, file)
			q := NewSpec(s.Columns())
			q.Filters = []FilterSpec{{Col: "status", Op: OpNe, Value: "FAILED", ColType: "VARCHAR"}}
			_, rows := pivotRows(t, s, q, PivotSpec{Rows: []string{"region"}, Percent: "col",
				Measures: []Measure{{Agg: "count"}, {Agg: "sum", Col: "amount"}}})
			if !reflect.DeepEqual(rows, want) {
				t.Errorf("rows = %v, want %v", rows, want)
			}
		})
	}
}

func TestPivotPercentNeedsNumbers(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	q.Pivot = &PivotSpec{Rows: []string{"status"}, Percent: "row", Measures: []Measure{{Agg: "max", Col: "region"}}}
	if _, err := s.ResolvePivot(context.Background(), q); err == nil {
		t.Error("percent of a text value must fail")
	}
	q.Pivot = &PivotSpec{Rows: []string{"status"}, Percent: "row", Measures: []Measure{{Agg: "avg", Col: "amount"}}}
	if _, err := s.ResolvePivot(context.Background(), q); err == nil {
		t.Error("percent of avg must fail")
	}
	q.Pivot = &PivotSpec{Rows: []string{"status"}, Percent: "row", Measures: []Measure{{Agg: "max", Col: "amount"}}}
	if _, err := s.ResolvePivot(context.Background(), q); err == nil {
		t.Error("percent of max must fail")
	}
	q.Pivot.Percent = "half"
	if _, err := q.SQL(); err == nil {
		t.Error("unknown percent must fail")
	}
}

func TestPivotPercentZeroTotalIsNull(t *testing.T) {
	s := openNDJSON(t, `{"region":"A","status":"x","amount":5}
{"region":"A","status":"y","amount":-5}
{"region":"B","status":"x","amount":10}
`)
	_, rows := pivotRows(t, s, NewSpec(s.Columns()), PivotSpec{Rows: []string{"region"}, On: "status",
		Measures: []Measure{{Agg: "sum", Col: "amount"}}, Percent: "row"})
	want := [][]string{{"A", "NULL", "NULL"}, {"B", "100", "NULL"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}

func TestPivotPercentSeveralValuesWithColumnField(t *testing.T) {
	s := openNDJSON(t, `{"region":"A","status":"x","amount":10}
{"region":"A","status":"y","amount":30}
{"region":"B","status":"x","amount":30}
{"region":"B","status":"x","amount":30}
`)
	cols, rows := pivotRows(t, s, NewSpec(s.Columns()), PivotSpec{Rows: []string{"region"}, On: "status",
		Measures: []Measure{{Agg: "sum", Col: "amount"}, {Agg: "count"}}, Percent: "col"})
	wantCols := []string{"region", "x sum(amount)", "x count", "y sum(amount)", "y count"}
	if !reflect.DeepEqual(cols, wantCols) {
		t.Fatalf("cols = %v, want %v", cols, wantCols)
	}
	want := [][]string{
		{"A", "14.29", "33.33", "100", "100"},
		{"B", "85.71", "66.67", "NULL", "0"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}
