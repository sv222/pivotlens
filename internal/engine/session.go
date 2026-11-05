package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/marcboeker/go-duckdb/v2"
)

type Session struct {
	db   *sql.DB
	cols []ColumnMeta
	Path string
	tmp  []string
}

func Open(ctx context.Context, path string) (*Session, error) {
	return OpenInput(ctx, Input{Paths: []string{path}})
}

func OpenInput(ctx context.Context, in Input) (_ *Session, err error) {
	paths, err := Expand(in.Paths)
	if err != nil {
		return nil, err
	}
	var spooled []string
	defer func() {
		if err != nil {
			removeAll(spooled)
		}
	}()
	names := append([]string(nil), paths...)
	scans := make([]string, len(paths))
	for i, p := range paths {
		if p == StdinPath {
			if p, err = spool(in.Stdin); err != nil {
				return nil, err
			}
			spooled = append(spooled, p)
			names[i] = "stdin"
		}
		if scans[i], err = ScanExpr(p); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	tmp := filepath.Join(os.TempDir(), "pivotlens")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		_ = db.Close()
		return nil, err
	}
	threads := runtime.NumCPU()
	if threads > 8 {
		threads = 8
	}
	pragmas := []string{
		fmt.Sprintf("SET threads TO %d", threads),
		"SET temp_directory = " + ql(tmp),
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(ctx, p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	src, err := union(ctx, db, names, scans, in.Source)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, "CREATE VIEW src AS "+src); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("read source: %w", err)
	}
	s := &Session{db: db, Path: names[0], tmp: spooled}
	if len(names) > 1 {
		s.Path = fmt.Sprintf("%d files", len(names))
	}
	if s.cols, err = describe(ctx, db, "src"); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func describe(ctx context.Context, db *sql.DB, query string) ([]ColumnMeta, error) {
	rows, err := db.QueryContext(ctx, "DESCRIBE "+query)
	if err != nil {
		return nil, fmt.Errorf("describe: %w", err)
	}
	defer func() { _ = rows.Close() }()
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if len(names) < 2 {
		return nil, errors.New("unexpected DESCRIBE output")
	}
	var cols []ColumnMeta
	for rows.Next() {
		vals := make([]any, len(names))
		ptrs := make([]any, len(names))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		cols = append(cols, ColumnMeta{
			Name: asString(vals[0]),
			Type: asString(vals[1]),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, errors.New("source has no columns")
	}
	return cols, nil
}

func (s *Session) Columns() []ColumnMeta { return s.cols }

func (s *Session) Close() error {
	err := s.db.Close()
	removeAll(s.tmp)
	return err
}
