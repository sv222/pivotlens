package engine

import (
	"fmt"
	"strings"
)

func qi(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func ql(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func likeEscape(s string) string {
	return likeEscaper.Replace(s)
}

var castTypes = map[string]bool{
	"VARCHAR":       true,
	"BOOLEAN":       true,
	"INTEGER":       true,
	"BIGINT":        true,
	"DOUBLE":        true,
	"DATE":          true,
	"TIME":          true,
	"TIMESTAMP":     true,
	"DECIMAL(18,2)": true,
	"DECIMAL(38,9)": true,
}

var aggFuncs = map[string]string{
	"count":          "count(%s)",
	"count_distinct": "count(DISTINCT %s)",
	"sum":            "sum(%s)",
	"avg":            "avg(%s)",
	"min":            "min(%s)",
	"max":            "max(%s)",
	"median":         "median(%s)",
}

func checkCast(t string) error {
	if !castTypes[t] {
		return fmt.Errorf("unsupported cast type %q", t)
	}
	return nil
}

func aggExpr(agg, col string) (string, error) {
	tmpl, ok := aggFuncs[agg]
	if !ok {
		return "", fmt.Errorf("unsupported aggregate %q", agg)
	}
	if col == "" {
		if agg != "count" {
			return "", fmt.Errorf("aggregate %q requires a column", agg)
		}
		return "count(*)", nil
	}
	return fmt.Sprintf(tmpl, qi(col)), nil
}
