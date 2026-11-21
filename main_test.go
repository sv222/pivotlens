package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sv222/pivotlens/internal/engine"
)

func fixture(name string) string {
	return filepath.Join("testdata", name)
}

func runFreqOut(t *testing.T, in engine.Input, col string, top int, out, view, saveView string) string {
	t.Helper()
	var b bytes.Buffer
	if err := runFreq(in, col, top, out, view, saveView, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestFreqCSV(t *testing.T) {
	got := runFreqOut(t, engine.Input{Paths: []string{fixture("tiny.csv")}}, "status", engine.DefaultTop, "", "", "")
	want := "value,count,percent,cum_percent\n" +
		"COMPLETED,3,60,60\n" +
		"FAILED,1,20,80\n" +
		"PENDING,1,20,100\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestFreqTopCutShowsOther(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "many.csv")
	if err := os.WriteFile(p, []byte("v\na\na\nb\nb\nc\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := runFreqOut(t, engine.Input{Paths: []string{p}}, "v", 2, "", "", "")
	want := "value,count,percent,cum_percent\n" +
		"a,2,33.33,33.33\n" +
		"b,2,33.33,66.67\n" +
		"(other),2,33.33,100\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestFreqStdin(t *testing.T) {
	in := engine.Input{Paths: []string{engine.StdinPath}, Stdin: strings.NewReader("v\nx\nx\ny\n")}
	got := runFreqOut(t, in, "v", engine.DefaultTop, "", "", "")
	want := "value,count,percent,cum_percent\nx,2,66.67,66.67\ny,1,33.33,100\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestFreqUnknownColumnListsColumns(t *testing.T) {
	in := engine.Input{Paths: []string{fixture("tiny.csv")}}
	var b bytes.Buffer
	err := runFreq(in, "nope", engine.DefaultTop, "", "", "", &b)
	if err == nil || !strings.Contains(err.Error(), `unknown column "nope"`) {
		t.Fatalf("err = %v, want unknown column message", err)
	}
	if !strings.Contains(err.Error(), "region") {
		t.Errorf("err = %v, want the error to list the columns", err)
	}
}

func TestFreqAppliesViewFilters(t *testing.T) {
	ctx := context.Background()
	sess, err := engine.Open(ctx, fixture("tiny.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()
	q := engine.NewSpec(sess.Columns())
	q.Filters = []engine.FilterSpec{{Col: "status", Op: engine.OpEq, Value: "COMPLETED", ColType: "VARCHAR"}}
	vp := filepath.Join(t.TempDir(), "view.json")
	if err := engine.SaveView(vp, q); err != nil {
		t.Fatal(err)
	}
	got := runFreqOut(t, engine.Input{Paths: []string{fixture("tiny.csv")}}, "region", engine.DefaultTop, "", vp, "")
	want := "value,count,percent,cum_percent\n" +
		"AP-SOUTH,1,33.33,33.33\n" +
		"EU-CENT,1,33.33,66.67\n" +
		"US-EAST,1,33.33,100\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestFreqBadArgs(t *testing.T) {
	in := engine.Input{Paths: []string{fixture("tiny.csv")}}
	cases := []struct {
		name     string
		col      string
		top      int
		out      string
		saveView string
		want     string
	}{
		{"negative top", "status", -1, "", "", "--top must be 0 or greater"},
		{"save view", "status", 0, "", "v.json", "--save-view needs --pivot"},
		{"bad output", "status", 0, "f.xlsx", "", "--output must end in .csv, .parquet or .ndjson"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var b bytes.Buffer
			err := runFreq(in, c.col, c.top, c.out, "", c.saveView, &b)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestFreqExports(t *testing.T) {
	dir := t.TempDir()
	in := engine.Input{Paths: []string{fixture("tiny.csv")}}
	csvOut := runFreqOut(t, in, "status", engine.DefaultTop, filepath.Join(dir, "f.csv"), "", "")
	// DuckDB writes the file and renders doubles with one decimal.
	want := "value,count,percent,cum_percent\nCOMPLETED,3,60.0,60.0\nFAILED,1,20.0,80.0\nPENDING,1,20.0,100.0\n"
	if csvOut != "" {
		t.Error("--output must not write to stdout")
	}
	b, err := os.ReadFile(filepath.Join(dir, "f.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Errorf("csv = %q, want %q", b, want)
	}
	for _, ext := range []string{"parquet", "ndjson"} {
		out := filepath.Join(dir, "f."+ext)
		if got := runFreqOut(t, in, "status", engine.DefaultTop, out, "", ""); got != "" {
			t.Fatalf("%s: --output must not write to stdout, got %q", ext, got)
		}
		s, err := engine.Open(context.Background(), out)
		if err != nil {
			t.Fatalf("%s: %v", ext, err)
		}
		q := engine.NewSpec(s.Columns())
		n, err := s.Count(context.Background(), q)
		if err != nil {
			t.Fatalf("%s: %v", ext, err)
		}
		_ = s.Close()
		if n != 3 {
			t.Errorf("%s holds %d rows, want 3", ext, n)
		}
	}
}
