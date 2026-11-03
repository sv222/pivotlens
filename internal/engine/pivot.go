package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const PivotLimit = 50

var PivotAggs = []string{"count", "sum", "avg", "min", "max"}

var PercentAggs = []string{"count", "sum"}

var PivotPercents = []string{"row", "col", "total"}

var percentText = map[string]string{"row": "row", "col": "column", "total": "grand total"}

const (
	EmptyText = "(empty)"
	TotalText = "Total"
)

var ErrPivotLimit = errors.New("too many pivot columns")

type pivotCol struct {
	name  string
	key   string
	keyed bool
	null  bool
	total bool
	m     int
}

func (m Measure) Label() string {
	if m.Col == "" {
		return m.Agg
	}
	return m.Agg + "(" + m.Col + ")"
}

func (p PivotSpec) layout() ([]pivotCol, string) {
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
	add := func(base string, c pivotCol) {
		for i, m := range p.Measures {
			name := base
			if len(p.Measures) > 1 {
				name += " " + m.Label()
			}
			if base == "" {
				name = m.Label()
			}
			c.name, c.m = unique(name), i
			out = append(out, c)
		}
	}
	if p.On == "" {
		add("", pivotCol{})
	}
	for _, k := range p.Keys {
		base := k
		if k == "" {
			base = EmptyText
		}
		add(base, pivotCol{key: k, keyed: true})
	}
	if p.Null {
		add(NullText, pivotCol{null: true})
	}
	add(TotalText, pivotCol{total: true})
	return out, unique("grouping")
}

func (p PivotSpec) shown(c pivotCol) bool {
	return !c.total || p.Totals && p.On != ""
}

func (p PivotSpec) Columns() []string {
	out := append([]string(nil), p.Rows...)
	cs, _ := p.layout()
	for _, c := range cs {
		if p.shown(c) {
			out = append(out, c.name)
		}
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
	s += " " + strings.Join(ls, ", ")
	if p.Totals {
		s += " with totals"
	}
	if p.Percent != "" {
		s += " as % of " + percentText[p.Percent]
	}
	return s
}

func parseMeasure(item, agg string) Measure {
	i := strings.LastIndex(item, ":")
	if a := strings.ToLower(strings.TrimSpace(item[i+1:])); i >= 0 && slices.Contains(PivotAggs, a) {
		col := strings.TrimSpace(item[:i])
		if col == "*" {
			col = ""
		}
		return Measure{Agg: a, Col: col}
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
		case "totals":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return p, fmt.Errorf("pivot: totals %q, want true or false", v)
			}
			p.Totals = b
		case "percent":
			if !slices.Contains(PivotPercents, v) {
				return p, fmt.Errorf("pivot: percent %q, want one of %s", v, strings.Join(PivotPercents, ", "))
			}
			p.Percent = v
		default:
			return p, fmt.Errorf("pivot: unknown key %q, want rows, cols, values, agg, totals or percent", k)
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
	pr := make([]string, len(p.Rows))
	for i, r := range p.Rows {
		rs[i] = qi(r)
		pr[i] = "p." + rs[i]
	}
	rows := strings.Join(rs, ", ")
	out := append([]string(nil), pr...)
	cs, g := p.layout()
	gq := qi(g)
	grand := p.Totals || p.Percent == "col" || p.Percent == "total"
	sel := []string{rows}
	mid := []string{rows}
	tot := make([]string, len(p.Measures))
	for _, c := range cs {
		if c.total {
			tot[c.m] = qi(c.name)
		}
	}
	on := qi(p.On)
	for _, c := range cs {
		n := qi(c.name)
		keep := p.shown(c) || c.total && p.Percent != ""
		if !keep {
			continue
		}
		e := aggs[c.m]
		switch {
		case c.keyed:
			e += " FILTER (WHERE CAST(" + on + " AS VARCHAR) = " + ql(c.key) + ")"
		case c.null:
			e += " FILTER (WHERE " + on + " IS NULL)"
		}
		sel = append(sel, e+" AS "+n)
		den := tot[c.m]
		switch p.Percent {
		case "col":
			den = "max(CASE WHEN " + gq + " <> 0 THEN " + n + " END) OVER ()"
		case "total":
			den = "max(CASE WHEN " + gq + " <> 0 THEN " + tot[c.m] + " END) OVER ()"
		}
		mid = append(mid, "round(100.0 * "+n+" / nullif("+den+", 0), 2) AS "+n)
		if p.shown(c) {
			out = append(out, "p."+n)
		}
	}
	group, where := rows, ""
	var order []string
	if grand {
		group = "GROUPING SETS ((" + rows + "), ())"
		sel = append(sel, "grouping("+rows+") AS "+gq)
		mid = append(mid, gq)
		where = " WHERE p." + gq + " = 0"
	}
	if p.Totals {
		out[0] = "CASE WHEN p." + gq + " <> 0 THEN " + ql(TotalText) + " ELSE CAST(" + pr[0] + " AS VARCHAR) END AS " + rs[0]
		order = append(order, "p."+gq)
		where = ""
	}
	for _, k := range q.Sort {
		dir := " ASC"
		if k.Desc {
			dir = " DESC"
		}
		order = append(order, "p."+qi(k.Col)+dir)
	}
	if len(q.Sort) == 0 {
		order = append(order, pr...)
	}
	from := "SELECT " + strings.Join(sel, ", ") + " FROM filtered GROUP BY " + group
	switch {
	case p.Percent != "" && !slices.Contains(PivotPercents, p.Percent):
		return "", fmt.Errorf("unsupported percent %q", p.Percent)
	case p.Percent != "":
		from = "SELECT " + strings.Join(mid, ", ") + " FROM (" + from + ")"
	}
	return "SELECT " + strings.Join(out, ", ") + " FROM (" + from + ") p" + where +
		" ORDER BY " + strings.Join(order, ", "), nil
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
	if p.Percent != "" {
		num := s.NumericFields(q)
		for _, m := range p.Measures {
			if !slices.Contains(PercentAggs, m.Agg) {
				return q, fmt.Errorf("percent works with %s only, %s is not allowed", strings.Join(PercentAggs, " and "), m.Label())
			}
			if m.Agg != "count" && !num[m.Col] {
				return q, fmt.Errorf("percent needs numeric values, %s is not", m.Label())
			}
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
