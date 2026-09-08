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
}

func Open(ctx context.Context, path string) (*Session, error) {
	scan, err := ScanExpr(path)
	if err != nil {
		return nil, err
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
	if _, err := db.ExecContext(ctx, "CREATE VIEW src AS SELECT * FROM "+scan); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("read source: %w", err)
	}
	s := &Session{db: db, Path: path}
	if err := s.describe(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Session) describe(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "DESCRIBE src")
	if err != nil {
		return fmt.Errorf("describe: %w", err)
	}
	defer func() { _ = rows.Close() }()
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(names) < 2 {
		return errors.New("unexpected DESCRIBE output")
	}
	for rows.Next() {
		vals := make([]any, len(names))
		ptrs := make([]any, len(names))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		s.cols = append(s.cols, ColumnMeta{
			Name: asString(vals[0]),
			Type: asString(vals[1]),
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(s.cols) == 0 {
		return errors.New("source has no columns")
	}
	return nil
}

func (s *Session) Columns() []ColumnMeta { return s.cols }

func (s *Session) Close() error { return s.db.Close() }
