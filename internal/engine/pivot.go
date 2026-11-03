package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const PivotLimit = 50

var PivotAggs = []string{"count", "sum", "avg", "min", "max"}

const EmptyText = "(empty)"

var ErrPivotLimit = errors.New("too many pivot columns")

func pivotAliases(taken []string, names []string, nullCol bool) ([]string, string) {
	seen := make(map[string]bool, len(taken)+len(names)+1)
	for _, t := range taken {
		seen[strings.ToLower(t)] = true
	}
	unique := func(base string) string {
		name := base
		for i := 1; seen[strings.ToLower(name)]; i++ {
			name = base + "_" + itoa(i)
		}
		seen[strings.ToLower(name)] = true
		return name
	}
	out := make([]string, len(names))
	for i, n := range names {
		if n == "" {
			n = EmptyText
		}
		out[i] = unique(n)
	}
	if !nullCol {
		return out, ""
	}
	return out, unique(NullText)
}

func (p PivotSpec) keys() []string {
	if len(p.Keys) == len(p.Values) {
		return p.Keys
	}
	return p.Values
}

func (p PivotSpec) nullCol() string {
	if p.NullCol == "" {
		return NullText
	}
	return p.NullCol
}

func (p PivotSpec) Label() string {
	if p.AggCol == "" {
		return p.Agg
	}
	return p.Agg + "(" + p.AggCol + ")"
}

func (p PivotSpec) Columns() []string {
	out := append([]string(nil), p.Rows...)
	if p.On == "" {
		return append(out, p.Label())
	}
	out = append(out, p.Values...)
	if p.Null {
		out = append(out, p.nullCol())
	}
	return out
}

func (p PivotSpec) String() string {
	s := strings.Join(p.Rows, ", ")
	if p.On != "" {
		s += " × " + p.On
	}
	return s + " " + p.Label()
}

func ParsePivot(s string) (PivotSpec, error) {
	p := PivotSpec{Agg: "count"}
	for _, part := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(part, "=")
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if !ok || v == "" {
			return p, fmt.Errorf("pivot: bad part %q, want key=value", part)
		}
		switch k {
		case "rows":
			p.Rows = []string{v}
		case "cols":
			p.On = v
		case "values":
			p.AggCol = v
		case "agg":
			p.Agg = strings.ToLower(v)
		default:
			return p, fmt.Errorf("pivot: unknown key %q, want rows, cols, values or agg", k)
		}
	}
	if len(p.Rows) == 0 {
		return p, errors.New("pivot: rows is required")
	}
	if !slices.Contains(PivotAggs, p.Agg) {
		return p, fmt.Errorf("pivot: agg %q, want one of %s", p.Agg, strings.Join(PivotAggs, ", "))
	}
	if p.Agg != "count" && p.AggCol == "" {
		return p, fmt.Errorf("pivot: agg %s needs values", p.Agg)
	}
	return p, nil
}

func (q QuerySpec) pivotBody() (string, error) {
	p := *q.Pivot
	if len(p.Rows) == 0 {
		return "", errors.New("pivot needs a row field")
	}
	agg, err := aggExpr(p.Agg, p.AggCol)
	if err != nil {
		return "", err
	}
	rs := make([]string, len(p.Rows))
	for i, r := range p.Rows {
		rs[i] = qi(r)
	}
	rows := strings.Join(rs, ", ")
	sel := []string{rows}
	if p.On == "" {
		sel = append(sel, agg+" AS "+qi(p.Label()))
	} else {
		on := qi(p.On)
		keys := p.keys()
		for i, v := range p.Values {
			sel = append(sel, agg+" FILTER (WHERE CAST("+on+" AS VARCHAR) = "+ql(keys[i])+") AS "+qi(v))
		}
		if p.Null {
			sel = append(sel, agg+" FILTER (WHERE "+on+" IS NULL) AS "+qi(p.nullCol()))
		}
	}
	order := q.orderBy()
	if order == "" {
		order = " ORDER BY " + rows
	}
	return "SELECT * FROM (SELECT " + strings.Join(sel, ", ") +
		" FROM filtered GROUP BY " + rows + ")" + order, nil
}

func (q QuerySpec) pivotValuesSQL() (string, error) {
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	return pre + "SELECT CAST(v AS VARCHAR) FROM (SELECT DISTINCT " + qi(q.Pivot.On) +
		" AS v FROM filtered) ORDER BY v NULLS LAST LIMIT " + itoa(PivotLimit+1), nil
}

func (s *Session) ResolvePivot(ctx context.Context, q QuerySpec) (QuerySpec, error) {
	if q.Pivot == nil {
		return q, errors.New("no pivot to resolve")
	}
	q = q.Clone()
	p := q.Pivot
	names := displayNamesOf(q)
	for _, c := range append(append([]string(nil), p.Rows...), p.On, p.AggCol) {
		if c != "" && !slices.Contains(names, c) {
			return q, fmt.Errorf("unknown column %q", c)
		}
	}
	p.Keys, p.Values, p.Null, p.NullCol = nil, nil, false, ""
	if p.On != "" {
		text, err := q.pivotValuesSQL()
		if err != nil {
			return q, err
		}
		rows, err := s.db.QueryContext(ctx, text)
		if err != nil {
			return q, err
		}
		defer func() { _ = rows.Close() }()
		n := 0
		for rows.Next() {
			var v sql.NullString
			if err := rows.Scan(&v); err != nil {
				return q, err
			}
			if n++; n > PivotLimit {
				return q, fmt.Errorf("%w: %s has more than %d distinct values, pick a column field with fewer", ErrPivotLimit, p.On, PivotLimit)
			}
			if v.Valid {
				p.Keys = append(p.Keys, v.String)
			} else {
				p.Null = true
			}
		}
		if err := rows.Err(); err != nil {
			return q, err
		}
		p.Values, p.NullCol = pivotAliases(p.Rows, p.Keys, p.Null)
	}
	_, err := q.SQL()
	return q, err
}

func displayNamesOf(q QuerySpec) []string {
	vis := q.Visible()
	out := make([]string, len(vis))
	for i, c := range vis {
		out[i] = c.Display()
	}
	return out
}
