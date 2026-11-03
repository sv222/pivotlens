package engine

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var ErrPivotStats = errors.New("pivot stats must be computed from results")

type ColumnStats struct {
	Rows     int64
	Nulls    int64
	Distinct int64
	Min      string
	Max      string
	Avg      string
	Sum      string
	Numeric  bool
}

func numericType(t string) bool {
	return strings.Contains(t, "INT") ||
		strings.HasPrefix(t, "DECIMAL") ||
		strings.Contains(t, "FLOAT") ||
		t == "DOUBLE" || t == "REAL"
}

func (s *Session) NumericFields(q QuerySpec) map[string]bool {
	out := make(map[string]bool)
	for _, c := range q.Visible() {
		t := c.Cast
		if t == "" {
			t = s.colType(c.Name)
		}
		out[c.Display()] = numericType(strings.ToUpper(t))
	}
	return out
}

func (q QuerySpec) StatsSQL(col string, numeric bool) (string, error) {
	if q.Pivot != nil {
		return "", ErrPivotStats
	}
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	c := qi(col)
	sel := "count(*), count(*) - count(" + c + "), count(DISTINCT " + c + "), " +
		"CAST(min(" + c + ") AS VARCHAR), CAST(max(" + c + ") AS VARCHAR)"
	if numeric {
		sel += ", CAST(avg(" + c + ") AS VARCHAR), CAST(sum(" + c + ") AS VARCHAR)"
	}
	return pre + "SELECT " + sel + " FROM filtered", nil
}

func (s *Session) colType(col string) string {
	for _, c := range s.cols {
		if c.Name == col {
			return c.Type
		}
	}
	return ""
}

func (s *Session) Stats(ctx context.Context, q QuerySpec, col string) (ColumnStats, error) {
	numeric := numericType(s.colType(col))
	text, err := q.StatsSQL(col, numeric)
	if err != nil {
		return ColumnStats{}, err
	}
	n := 5
	if numeric {
		n = 7
	}
	raw := make([]sql.NullString, n)
	ptrs := make([]any, n)
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	if err := s.db.QueryRowContext(ctx, text).Scan(ptrs...); err != nil {
		return ColumnStats{}, err
	}
	val := func(i int) string {
		if raw[i].Valid {
			return raw[i].String
		}
		return ""
	}
	st := ColumnStats{Numeric: numeric}
	if err := scanInt(raw[0], &st.Rows); err != nil {
		return ColumnStats{}, err
	}
	if err := scanInt(raw[1], &st.Nulls); err != nil {
		return ColumnStats{}, err
	}
	if err := scanInt(raw[2], &st.Distinct); err != nil {
		return ColumnStats{}, err
	}
	st.Min, st.Max = val(3), val(4)
	if numeric {
		st.Avg, st.Sum = val(5), val(6)
	}
	return st, nil
}

func scanInt(v sql.NullString, dst *int64) error {
	if !v.Valid {
		return nil
	}
	var n int64
	for _, r := range v.String {
		if r < '0' || r > '9' {
			return nil
		}
		n = n*10 + int64(r-'0')
	}
	*dst = n
	return nil
}
