package engine

import (
	"errors"
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
