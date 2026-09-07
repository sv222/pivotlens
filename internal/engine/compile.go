package engine

import (
	"errors"
	"strconv"
	"strings"
)

var ErrNoColumns = errors.New("no visible columns")

func colExpr(c ColSpec) (string, error) {
	e := qi(c.Name)
	if c.Trim {
		e = "trim(" + e + ")"
	}
	if c.NullAs != "" {
		e = "coalesce(" + e + ", " + ql(c.NullAs) + ")"
	}
	if c.Cast != "" {
		if err := checkCast(c.Cast); err != nil {
			return "", err
		}
		e = "CAST(" + e + " AS " + c.Cast + ")"
	}
	return e, nil
}

func (q QuerySpec) selectList() (string, error) {
	vis := q.Visible()
	if len(vis) == 0 {
		return "", ErrNoColumns
	}
	parts := make([]string, len(vis))
	for i, c := range vis {
		e, err := colExpr(c)
		if err != nil {
			return "", err
		}
		parts[i] = e + " AS " + qi(c.Display())
	}
	return strings.Join(parts, ", "), nil
}

func (q QuerySpec) baseWhere() string {
	if len(q.DropNull) == 0 {
		return ""
	}
	preds := make([]string, len(q.DropNull))
	for i, c := range q.DropNull {
		preds[i] = qi(c) + " IS NOT NULL"
	}
	return " WHERE " + strings.Join(preds, " AND ")
}

func (q QuerySpec) filterWhere() string {
	if q.Filter == "" {
		return ""
	}
	vis := q.Visible()
	if len(vis) == 0 {
		return ""
	}
	pat := ql("%" + likeEscape(q.Filter) + "%")
	preds := make([]string, len(vis))
	for i, c := range vis {
		preds[i] = "coalesce(CAST(" + qi(c.Display()) + " AS VARCHAR), '') ILIKE " + pat + ` ESCAPE '\'`
	}
	return " WHERE (" + strings.Join(preds, " OR ") + ")"
}

var ErrPivotWidths = errors.New("pivot widths must be measured from results")

func (q QuerySpec) prefix() (string, error) {
	sel, err := q.selectList()
	if err != nil {
		return "", err
	}
	return "WITH base AS (SELECT " + sel + " FROM src" + q.baseWhere() + "), " +
		"filtered AS (SELECT * FROM base" + q.filterWhere() + ") ", nil
}

func (q QuerySpec) orderBy() string {
	if len(q.Sort) == 0 {
		return ""
	}
	parts := make([]string, len(q.Sort))
	for i, k := range q.Sort {
		dir := " ASC"
		if k.Desc {
			dir = " DESC"
		}
		parts[i] = qi(k.Col) + dir
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

func (q QuerySpec) body() (string, error) {
	if q.Pivot == nil {
		return "SELECT * FROM filtered" + q.orderBy(), nil
	}
	p := *q.Pivot
	if len(p.Rows) == 0 {
		return "", errors.New("pivot needs at least one row group")
	}
	if p.On == "" {
		return "", errors.New("pivot needs a pivot column")
	}
	agg, err := aggExpr(p.Agg, p.AggCol)
	if err != nil {
		return "", err
	}
	rows := make([]string, len(p.Rows))
	for i, r := range p.Rows {
		rows[i] = qi(r)
	}
	return "PIVOT filtered ON " + qi(p.On) + " USING " + agg +
		" GROUP BY " + strings.Join(rows, ", "), nil
}

func (q QuerySpec) PageSQL(limit, offset int) (string, error) {
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	b, err := q.body()
	if err != nil {
		return "", err
	}
	return pre + b + " LIMIT " + itoa(limit) + " OFFSET " + itoa(offset), nil
}

func (q QuerySpec) CountSQL() (string, error) {
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	if q.Pivot == nil {
		return pre + "SELECT count(*) FROM filtered", nil
	}
	b, err := q.body()
	if err != nil {
		return "", err
	}
	return pre + "SELECT count(*) FROM (" + b + ")", nil
}

func (q QuerySpec) WidthsSQL(sample int) (string, error) {
	if q.Pivot != nil {
		return "", ErrPivotWidths
	}
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	vis := q.Visible()
	parts := make([]string, len(vis))
	for i, c := range vis {
		parts[i] = "max(length(CAST(" + qi(c.Display()) + " AS VARCHAR)))"
	}
	return pre + "SELECT " + strings.Join(parts, ", ") +
		" FROM (SELECT * FROM filtered LIMIT " + itoa(sample) + ")", nil
}

func (q QuerySpec) ExportSQL(path, format string) (string, error) {
	var opts string
	switch strings.ToLower(format) {
	case "parquet":
		opts = "(FORMAT parquet)"
	case "csv":
		opts = "(FORMAT csv, HEADER)"
	default:
		return "", errors.New("unsupported export format " + format)
	}
	pre, err := q.prefix()
	if err != nil {
		return "", err
	}
	b, err := q.body()
	if err != nil {
		return "", err
	}
	return "COPY (" + pre + b + ") TO " + ql(path) + " " + opts, nil
}

func itoa(n int) string { return strconv.Itoa(n) }
