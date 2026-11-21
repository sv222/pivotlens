package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// DefaultTop is how many values a value counts view shows unless the top
// is set to something else. 0 means all values.
const DefaultTop = 1000

const (
	FreqNullText  = "(null)"
	FreqEmptyText = "(empty)"
	FreqOtherText = "(other)"
)

const freqProjection = `"value", "count", "percent", "cum_percent"`

// FreqSpec asks for the value counts of one column over the filtered rows.
// The view is transient: SaveView drops it, while drill-down filters made
// from it are kept like any filter.
type FreqSpec struct {
	Col string `json:"col"`
	Top int    `json:"top,omitempty"`
}

// Columns returns the headers of the value counts result. The TUI adds a
// bar column for display only; it never appears in the SQL result.
func (f FreqSpec) Columns() []string {
	return []string{"value", "count", "percent", "cum_percent"}
}

// ResolveFreq checks col against the visible columns and names them when
// it is missing.
func (q QuerySpec) ResolveFreq(col string) error {
	names := displayNamesOf(q)
	if slices.Contains(names, col) {
		return nil
	}
	return fmt.Errorf("unknown column %q, have %s", col, strings.Join(names, ", "))
}

func freqValueExpr(col string) string {
	c := qi(col)
	return "CASE WHEN " + c + " IS NULL THEN " + ql(FreqNullText) +
		" WHEN CAST(" + c + " AS VARCHAR) = '' THEN " + ql(FreqEmptyText) +
		" ELSE CAST(" + c + " AS VARCHAR) END"
}

// freqOrder builds the outer ORDER BY of the value counts result. Sort
// keys that name no column of the result are ignored, and the (other) row
// stays last.
func (q QuerySpec) freqOrder(withOther bool) string {
	var parts []string
	for _, k := range q.Sort {
		if !slices.Contains((*q.Freq).Columns(), k.Col) {
			continue
		}
		dir := " ASC"
		if k.Desc {
			dir = " DESC"
		}
		parts = append(parts, qi(k.Col)+dir)
	}
	if len(parts) == 0 {
		parts = append(parts, `"count" DESC`, `"value" ASC`)
	}
	if withOther {
		return `"other" ASC, ` + strings.Join(parts, ", ")
	}
	return strings.Join(parts, ", ")
}

// freqBody continues the CTE list opened by prefix with the value counts
// query. projection selects the result columns, or count(*) for the row
// count of the result.
func (q QuerySpec) freqBody(projection string) (string, error) {
	f := *q.Freq
	if f.Col == "" {
		return "", errors.New("value counts need a column")
	}
	if f.Top < 0 {
		return "", errors.New("value counts top must be 0 or greater")
	}
	c := qi(f.Col)
	v := freqValueExpr(f.Col)
	shown := "SELECT " + v + ` AS "value", count(*) AS "count" FROM filtered GROUP BY ` + c +
		" ORDER BY count(*) DESC, " + v + " ASC"
	if f.Top > 0 {
		shown += " LIMIT " + itoa(f.Top)
	}
	tot := `SELECT count(*) AS "total", count(DISTINCT ` + c + `) AS "distinct_count" FROM filtered`
	ranked := `SELECT s."value", s."count",` +
		` round(100.0 * s."count" / nullif(t."total", 0), 2) AS "percent",` +
		` round(100.0 * sum(s."count") OVER (ORDER BY s."count" DESC, s."value" ASC) / nullif(t."total", 0), 2) AS "cum_percent",` +
		` 0 AS "other" FROM shown s CROSS JOIN tot t`
	src := ranked
	if f.Top > 0 {
		rest := `t."total" - (SELECT coalesce(sum("count"), 0) FROM shown)`
		src = ranked + " UNION ALL SELECT " + ql(FreqOtherText) + `, ` + rest + `,` +
			` round(100.0 * (` + rest + `) / nullif(t."total", 0), 2), 100.0, 1` +
			` FROM tot t WHERE t."distinct_count" > ` + itoa(f.Top)
	}
	out := `, shown AS (` + shown + `), tot AS (` + tot + `) SELECT ` + projection +
		` FROM (` + src + `) u`
	if projection == freqProjection {
		out += ` ORDER BY ` + q.freqOrder(f.Top > 0)
	}
	return out, nil
}

// FreqStatsSQL builds the companion query of a value counts view: the
// filtered row total and the distinct count of the column.
func (q QuerySpec) FreqStatsSQL() (string, error) {
	if q.Freq == nil {
		return "", errors.New("no value counts in progress")
	}
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	return pre + "SELECT count(*), count(DISTINCT " + qi(q.Freq.Col) + ") FROM filtered", nil
}

// FreqStats holds the filtered row total and the distinct count that the
// status bar of a value counts view reports.
type FreqStats struct {
	Total    int64
	Distinct int64
}

func (s *Session) FreqStats(ctx context.Context, q QuerySpec) (FreqStats, error) {
	text, err := q.FreqStatsSQL()
	if err != nil {
		return FreqStats{}, err
	}
	var st FreqStats
	if err := s.db.QueryRowContext(ctx, text).Scan(&st.Total, &st.Distinct); err != nil {
		return FreqStats{}, err
	}
	return st, nil
}
