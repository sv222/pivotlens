package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func richSpec(s *Session) QuerySpec {
	q := NewSpec(s.Columns())
	q.Cols[1].Hidden = true
	q.Cols[0].Alias = "area"
	q.Filters = []FilterSpec{{Col: "amount", Op: OpGt, Value: "50", ColType: "DECIMAL(18,2)"}, {Col: "area", Op: OpContains, Value: "E"}}
	q.Sort = []SortKey{{Col: "amount", Desc: true}}
	return q
}

func TestViewRoundTrip(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := richSpec(s)
	path := filepath.Join(t.TempDir(), "v.json")
	if err := SaveView(path, q); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadView(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, q) {
		t.Errorf("round trip changed the spec:\n got %+v\nwant %+v", got, q)
	}
	b, _ := os.ReadFile(path)
	for _, want := range []string{`"version": 1`, `"op": ">"`, `"op": "contains"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("view file lacks %s:\n%s", want, b)
		}
	}
}

func TestViewPivotKeysResolvedOnLoad(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	p, err := ParsePivot("rows=region,cols=status,values=amount:sum,totals=true")
	if err != nil {
		t.Fatal(err)
	}
	q := NewSpec(s.Columns())
	q.Pivot = &p
	q, err = s.ResolvePivot(context.Background(), q)
	if err != nil || len(q.Pivot.Keys) == 0 {
		t.Fatalf("keys = %v, err = %v", q.Pivot, err)
	}
	path := filepath.Join(t.TempDir(), "v.json")
	if err := SaveView(path, q); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); strings.Contains(string(b), `"keys"`) {
		t.Errorf("keys must not be saved:\n%s", b)
	}
	got, err := s.LoadView(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, q) {
		t.Errorf("got %+v, want %+v", got.Pivot, q.Pivot)
	}
}

func TestLoadViewRejects(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	dir := t.TempDir()
	cols := `[{"name":"region"},{"name":"status"},{"name":"amount"},{"name":"created_at"}]`
	cases := map[string]struct{ body, want string }{
		"not json":    {`nope`, "bad view"},
		"no version":  {`{"spec":{"cols":` + cols + `}}`, "missing version"},
		"future":      {`{"version":2,"spec":{"cols":` + cols + `}}`, "version 2"},
		"extra field": {`{"version":1,"spec":{"cols":` + cols + `,"bogus":1}}`, "bogus"},
		"bad op":      {`{"version":1,"spec":{"cols":` + cols + `,"filters":[{"col":"amount","op":"~"}]}}`, "operator"},
		"unknown col": {`{"version":1,"spec":{"cols":[{"name":"zzz"}]}}`, `unknown column "zzz"`},
		"few cols":    {`{"version":1,"spec":{"cols":[{"name":"region"}]}}`, "1 columns"},
		"bad filter":  {`{"version":1,"spec":{"cols":` + cols + `,"filters":[{"col":"nope","op":"="}]}}`, "does not fit"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(dir, strings.ReplaceAll(name, " ", "_")+".json")
			if err := os.WriteFile(p, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := s.LoadView(context.Background(), p)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
	if _, err := s.LoadView(context.Background(), filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file must be an error")
	}
}

func TestSaveViewKeepsCallerSpec(t *testing.T) {
	q := QuerySpec{Pivot: &PivotSpec{Keys: []string{"a"}, Null: true}}
	if err := SaveView(filepath.Join(t.TempDir(), "v.json"), q); err != nil {
		t.Fatal(err)
	}
	if len(q.Pivot.Keys) != 1 || !q.Pivot.Null {
		t.Error("SaveView must not modify its argument")
	}
}
