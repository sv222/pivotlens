package engine

import (
	"context"
	"database/sql"
	"errors"
)

const WidthSample = 1000

func (s *Session) FetchPage(ctx context.Context, q QuerySpec, limit, offset int) ([][]string, []string, error) {
	text, err := q.PageSQL(limit, offset)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.db.QueryContext(ctx, text)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	out := make([][]string, 0, limit)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		rec := make([]string, len(cols))
		for i, v := range vals {
			rec[i] = FormatValue(v)
		}
		out = append(out, rec)
	}
	return out, cols, rows.Err()
}

func (s *Session) Count(ctx context.Context, q QuerySpec) (int64, error) {
	text, err := q.CountSQL()
	if err != nil {
		return 0, err
	}
	var n int64
	if err := s.db.QueryRowContext(ctx, text).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Session) Widths(ctx context.Context, q QuerySpec, sample int) ([]int, error) {
	text, err := q.WidthsSQL(sample)
	if errors.Is(err, ErrPivotWidths) {
		return s.measureWidths(ctx, q, sample)
	}
	if err != nil {
		return nil, err
	}
	n := len(q.Visible())
	raw := make([]sql.NullInt64, n)
	ptrs := make([]any, n)
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	if err := s.db.QueryRowContext(ctx, text).Scan(ptrs...); err != nil {
		return nil, err
	}
	w := make([]int, n)
	for i, v := range raw {
		if v.Valid {
			w[i] = int(v.Int64)
		}
	}
	return w, nil
}

func (s *Session) measureWidths(ctx context.Context, q QuerySpec, sample int) ([]int, error) {
	rows, cols, err := s.FetchPage(ctx, q, sample, 0)
	if err != nil {
		return nil, err
	}
	w := make([]int, len(cols))
	for i, c := range cols {
		w[i] = len([]rune(c))
	}
	for _, r := range rows {
		for i, v := range r {
			if n := len([]rune(v)); n > w[i] {
				w[i] = n
			}
		}
	}
	return w, nil
}

func (s *Session) Export(ctx context.Context, q QuerySpec, path, format string) error {
	text, err := q.ExportSQL(path, format)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, text)
	return err
}
