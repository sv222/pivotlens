package engine

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const StdinPath = "-"

type Input struct {
	Paths  []string
	Stdin  io.Reader
	Source string
}

// Expand resolves glob patterns that match no file by that exact name.
func Expand(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, errors.New("no input files")
	}
	var out []string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil || p == StdinPath || !strings.ContainsAny(p, "*?[") {
			out = append(out, p)
			continue
		}
		m, err := filepath.Glob(p)
		if err != nil {
			return nil, fmt.Errorf("bad pattern %q: %w", p, err)
		}
		if len(m) == 0 {
			return nil, fmt.Errorf("no files match %q", p)
		}
		out = append(out, m...)
	}
	n := 0
	for _, p := range out {
		if p == StdinPath {
			n++
		}
	}
	if n > 1 {
		return nil, errors.New("stdin can be read only once")
	}
	return out, nil
}

// spool copies stdin to a temp file, since DuckDB scans the source many times.
func spool(r io.Reader) (string, error) {
	if r == nil {
		r = os.Stdin
	}
	br := bufio.NewReader(r)
	head, _ := br.Peek(512)
	head = bytes.TrimLeft(head, " \t\r\n")
	if len(head) == 0 {
		return "", errors.New("stdin is empty")
	}
	ext := ".csv"
	switch {
	case bytes.HasPrefix(head, []byte("PAR1")):
		ext = ".parquet"
	case head[0] == '{' || head[0] == '[':
		ext = ".ndjson"
	}
	f, err := os.CreateTemp("", "pivotlens-stdin-*"+ext)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, br)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return f.Name(), nil
}

func removeAll(paths []string) {
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

// union builds the src query: every scan must have the first scan's schema.
func union(ctx context.Context, db *sql.DB, names, scans []string, source string) (string, error) {
	var first []ColumnMeta
	if len(scans) > 1 || source != "" {
		var err error
		if first, err = describe(ctx, db, "SELECT * FROM "+scans[0]); err != nil {
			return "", fmt.Errorf("%s: %w", names[0], err)
		}
	}
	for _, c := range first {
		if source != "" && strings.EqualFold(c.Name, source) {
			return "", fmt.Errorf("--source-column %q clashes with a column of %s", source, names[0])
		}
	}
	parts := make([]string, len(scans))
	for i, sc := range scans {
		if i > 0 {
			cols, err := describe(ctx, db, "SELECT * FROM "+sc)
			if err != nil {
				return "", fmt.Errorf("%s: %w", names[i], err)
			}
			if d := schemaDiff(first, cols); d != "" {
				return "", fmt.Errorf("schema of %s differs from %s: %s", names[i], names[0], d)
			}
		}
		parts[i] = "SELECT *"
		if source != "" {
			parts[i] += ", " + ql(filepath.ToSlash(names[i])) + " AS " + qi(source)
		}
		parts[i] += " FROM " + sc
	}
	return strings.Join(parts, " UNION ALL "), nil
}

func schemaDiff(want, got []ColumnMeta) string {
	if len(want) != len(got) {
		return fmt.Sprintf("%d columns, want %d", len(got), len(want))
	}
	for i, w := range want {
		if g := got[i]; g != w {
			return fmt.Sprintf("column %d is %s %s, want %s %s", i+1, g.Name, g.Type, w.Name, w.Type)
		}
	}
	return ""
}
