package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func openInput(t *testing.T, in Input) *Session {
	t.Helper()
	s, err := OpenInput(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenInputStdin(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		cols int
	}{
		"csv":    {"a,b\n1,x\n2,y\n", 2},
		"ndjson": {"\n {\"a\":1,\"b\":\"x\"}\n{\"a\":2,\"b\":\"y\"}\n", 2},
	} {
		t.Run(name, func(t *testing.T) {
			s := openInput(t, Input{Paths: []string{StdinPath}, Stdin: strings.NewReader(tc.body)})
			if len(s.Columns()) != tc.cols || s.Path != "stdin" {
				t.Fatalf("cols=%d path=%q", len(s.Columns()), s.Path)
			}
			n, err := s.Count(context.Background(), NewSpec(s.Columns()))
			if err != nil || n != 2 {
				t.Fatalf("count=%d err=%v", n, err)
			}
		})
	}
}

func TestOpenInputStdinParquet(t *testing.T) {
	f, err := os.Open(fixture("tiny.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s := openInput(t, Input{Paths: []string{StdinPath}, Stdin: f})
	if len(s.Columns()) != 4 {
		t.Fatalf("got %d columns, want 4", len(s.Columns()))
	}
}

func TestStdinTempRemovedOnClose(t *testing.T) {
	s, err := OpenInput(context.Background(), Input{Paths: []string{StdinPath}, Stdin: strings.NewReader("a\n1\n")})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.tmp) != 1 {
		t.Fatalf("tmp = %v", s.tmp)
	}
	_ = s.Close()
	if _, err := os.Stat(s.tmp[0]); !os.IsNotExist(err) {
		t.Errorf("temp file left behind: %v", err)
	}
}

func TestOpenInputEmptyStdin(t *testing.T) {
	_, err := OpenInput(context.Background(), Input{Paths: []string{StdinPath}, Stdin: strings.NewReader(" \n")})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v, want empty stdin error", err)
	}
}

func TestOpenInputMultiFile(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.csv", "k,v\nx,1\ny,2\n")
	b := writeFile(t, dir, "b.csv", "k,v\nz,3\n")
	s := openInput(t, Input{Paths: []string{a, b}, Source: "file"})
	if s.Path != "2 files" {
		t.Errorf("Path = %q", s.Path)
	}
	cols := s.Columns()
	if len(cols) != 3 || cols[2].Name != "file" {
		t.Fatalf("columns = %v", cols)
	}
	rows, _, err := s.FetchPage(context.Background(), NewSpec(cols), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0][2] != filepath.ToSlash(a) || rows[2][2] != filepath.ToSlash(b) {
		t.Errorf("rows = %v", rows)
	}
}

func TestOpenInputGlob(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "p1.csv", "k\n1\n")
	writeFile(t, dir, "p2.csv", "k\n2\n")
	writeFile(t, dir, "other.csv", "z\n9\n")
	s := openInput(t, Input{Paths: []string{filepath.Join(dir, "p*.csv")}})
	n, err := s.Count(context.Background(), NewSpec(s.Columns()))
	if err != nil || n != 2 || len(s.Columns()) != 1 {
		t.Fatalf("count=%d cols=%v err=%v", n, s.Columns(), err)
	}
}

func TestOpenInputNoMatch(t *testing.T) {
	_, err := OpenInput(context.Background(), Input{Paths: []string{filepath.Join(t.TempDir(), "*.csv")}})
	if err == nil || !strings.Contains(err.Error(), "no files match") {
		t.Errorf("err = %v", err)
	}
}

func TestOpenInputSchemaMismatch(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.csv", "k,v\nx,1\n")
	for name, tc := range map[string]struct{ body, want string }{
		"count": {"k,v,w\nx,1,2\n", "3 columns, want 2"},
		"name":  {"k,w\nx,1\n", "column 2 is w BIGINT, want v BIGINT"},
		"type":  {"k,v\nx,abc\n", "column 2 is v VARCHAR, want v BIGINT"},
	} {
		t.Run(name, func(t *testing.T) {
			b := writeFile(t, dir, name+".csv", tc.body)
			_, err := OpenInput(context.Background(), Input{Paths: []string{a, b}})
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), name+".csv") {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestOpenInputSourceClash(t *testing.T) {
	a := writeFile(t, t.TempDir(), "a.csv", "Source\n1\n")
	_, err := OpenInput(context.Background(), Input{Paths: []string{a}, Source: "source"})
	if err == nil || !strings.Contains(err.Error(), "clashes") {
		t.Errorf("err = %v", err)
	}
}

func TestOpenInputSingleSource(t *testing.T) {
	s := openInput(t, Input{Paths: []string{fixture("tiny.csv")}, Source: "src"})
	if cols := s.Columns(); len(cols) != 5 || cols[4].Name != "src" {
		t.Errorf("columns = %v", cols)
	}
}

func TestExpandStdinTwice(t *testing.T) {
	if _, err := Expand([]string{StdinPath, StdinPath}); err == nil {
		t.Error("two stdin inputs must be an error")
	}
}
