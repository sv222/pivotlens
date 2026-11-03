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

type pivotCol struct {
	name  string
	key   string
	keyed bool
	null  bool
	m     int
}

func (m Measure) Label() string {
	if m.Col == "" {
		return m.Agg
	}
	return m.Agg + "(" + m.Col + ")"
}

func (p PivotSpec) layout() []pivotCol {
	seen := make(map[string]bool)
	for _, r := range p.Rows {
		seen[strings.ToLower(r)] = true
	}
	unique := func(base string) string {
		name := base
		for i := 1; seen[strings.ToLower(name)]; i++ {
			name = base + "_" + itoa(i)
		}
		seen[strings.ToLower(name)] = true
		return name
	}
	var out []pivotCol
	if p.On == "" {
		for i, m := range p.Measures {
			out = append(out, pivotCol{name: unique(m.Label()), m: i})
		}
		return out
	}
	add := func(key string, null bool) {
		base := key
		switch {
		case null:
			base = NullText
		case key == "":
			base = EmptyText
		}
		for i, m := range p.Measures {
			name := base
			if len(p.Measures) > 1 {
				name += " " + m.Label()
			}
			out = append(out, pivotCol{name: unique(name), key: key, keyed: !null, null: null, m: i})
		}
	}
	for _, k := range p.Keys {
		add(k, false)
	}
	if p.Null {
		add("", true)
	}
	return out
}

func (p PivotSpec) Columns() []string {
	out := append([]string(nil), p.Rows...)
	for _, c := range p.layout() {
		out = append(out, c.name)
	}
	return out
}

func (p PivotSpec) String() string {
	s := strings.Join(p.Rows, ", ")
	if p.On != "" {
		s += " × " + p.On
	}
	ls := make([]string, len(p.Measures))
	for i, m := range p.Measures {
		ls[i] = m.Label()
	}
	return s + " " + strings.Join(ls, ", ")
}

func parseMeasure(item, agg string) Measure {
	if i := strings.LastIndex(item, ":"); i >= 0 && slices.Contains(PivotAggs, strings.ToLower(item[i+1:])) {
		col := strings.TrimSpace(item[:i])
		if col == "*" {
			col = ""
		}
		return Measure{Agg: strings.ToLower(item[i+1:]), Col: col}
	}
	return Measure{Agg: agg, Col: item}
}

func ParsePivot(s string) (PivotSpec, error) {
	var p PivotSpec
	agg := "count"
	var items []string
	last := ""
	for _, part := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(part, "=")
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if !ok && last == "values" && k != "" {
			items = append(items, k)
			continue
		}
		if !ok || v == "" {
			return p, fmt.Errorf("pivot: bad part %q, want key=value", part)
		}
		last = k
		switch k {
		case "rows":
			p.Rows = []string{v}
		case "cols":
			p.On = v
		case "values":
			items = append(items, v)
		case "agg":
			agg = strings.ToLower(v)
		default:
			return p, fmt.Errorf("pivot: unknown key %q, want rows, cols, values or agg", k)
		}
	}
	if len(p.Rows) == 0 {
		return p, errors.New("pivot: rows is required")
	}
	for _, it := range items {
		p.Measures = append(p.Measures, parseMeasure(it, agg))
	}
	if len(p.Measures) == 0 {
		p.Measures = []Measure{{Agg: agg}}
	}
	for _, m := range p.Measures {
		if !slices.Contains(PivotAggs, m.Agg) {
			return p, fmt.Errorf("pivot: agg %q, want one of %s", m.Agg, strings.Join(PivotAggs, ", "))
		}
		if m.Agg != "count" && m.Col == "" {
			return p, fmt.Errorf("pivot: agg %s needs values", m.Agg)
		}
	}
	return p, nil
}

func (q QuerySpec) pivotBody() (string, error) {
	p := *q.Pivot
	if len(p.Rows) == 0 {
		return "", errors.New("pivot needs a row field")
	}
	if len(p.Measures) == 0 {
		return "", errors.New("pivot needs a value")
	}
	aggs := make([]string, len(p.Measures))
	for i, m := range p.Measures {
		e, err := aggExpr(m.Agg, m.Col)
		if err != nil {
			return "", err
		}
		aggs[i] = e
	}
	rs := make([]string, len(p.Rows))
	for i, r := range p.Rows {
		rs[i] = qi(r)
	}
	rows := strings.Join(rs, ", ")
	sel := []string{rows}
	on := qi(p.On)
	for _, c := range p.layout() {
		e := aggs[c.m]
		switch {
		case c.keyed:
			e += " FILTER (WHERE CAST(" + on + " AS VARCHAR) = " + ql(c.key) + ")"
		case c.null:
			e += " FILTER (WHERE " + on + " IS NULL)"
		}
		sel = append(sel, e+" AS "+qi(c.name))
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
	cols := append([]string(nil), p.Rows...)
	cols = append(cols, p.On)
	for _, m := range p.Measures {
		cols = append(cols, m.Col)
	}
	for _, c := range cols {
		if c != "" && !slices.Contains(names, c) {
			return q, fmt.Errorf("unknown column %q", c)
		}
	}
	p.Keys, p.Null = nil, false
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
