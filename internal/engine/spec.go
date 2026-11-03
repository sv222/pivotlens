package engine

import "strings"

type ColumnMeta struct {
	Name string
	Type string
}

type ColSpec struct {
	Name   string
	Alias  string
	Cast   string
	Hidden bool
	Trim   bool
	NullAs string
}

func (c ColSpec) Display() string {
	if c.Alias != "" {
		return c.Alias
	}
	return c.Name
}

type SortKey struct {
	Col  string
	Desc bool
}

type Measure struct {
	Agg string
	Col string
}

type PivotSpec struct {
	Rows     []string
	On       string
	Measures []Measure
	Totals   bool
	Percent  string
	Keys     []string
	Null     bool
}

type QuerySpec struct {
	Cols     []ColSpec
	Filter   string
	Filters  []FilterSpec
	DropNull []string
	Sort     []SortKey
	Pivot    *PivotSpec
}

func NewSpec(cols []ColumnMeta) QuerySpec {
	cs := make([]ColSpec, len(cols))
	for i, c := range cols {
		cs[i] = ColSpec{Name: c.Name}
	}
	return QuerySpec{Cols: cs}
}

func (q QuerySpec) Visible() []ColSpec {
	out := make([]ColSpec, 0, len(q.Cols))
	for _, c := range q.Cols {
		if !c.Hidden {
			out = append(out, c)
		}
	}
	return out
}

func (q QuerySpec) Clone() QuerySpec {
	n := q
	n.Cols = append([]ColSpec(nil), q.Cols...)
	n.DropNull = append([]string(nil), q.DropNull...)
	n.Filters = append([]FilterSpec(nil), q.Filters...)
	n.Sort = append([]SortKey(nil), q.Sort...)
	if q.Pivot != nil {
		p := *q.Pivot
		p.Rows = append([]string(nil), q.Pivot.Rows...)
		p.Measures = append([]Measure(nil), q.Pivot.Measures...)
		p.Keys = append([]string(nil), q.Pivot.Keys...)
		n.Pivot = &p
	}
	return n
}

type Op int

const (
	OpEq Op = iota
	OpNe
	OpGt
	OpGe
	OpLt
	OpLe
	OpContains
	OpStarts
	OpEnds
	OpNull
	OpNotNull
)

func (o Op) String() string {
	switch o {
	case OpEq:
		return "="
	case OpNe:
		return "!="
	case OpGt:
		return ">"
	case OpGe:
		return ">="
	case OpLt:
		return "<"
	case OpLe:
		return "<="
	case OpContains:
		return "contains"
	case OpStarts:
		return "starts with"
	case OpEnds:
		return "ends with"
	case OpNull:
		return "is null"
	case OpNotNull:
		return "is not null"
	default:
		return "?"
	}
}

type FilterSpec struct {
	Col     string
	Op      Op
	Value   string
	ColType string
}

func (f FilterSpec) String() string {
	if f.Op == OpNull || f.Op == OpNotNull {
		return f.Col + " " + f.Op.String()
	}
	return f.Col + " " + f.Op.String() + " " + f.Value
}

func OpsFor(t string) []Op {
	if strings.HasPrefix(t, "VARCHAR") || t == "TEXT" || t == "STRING" {
		return []Op{OpEq, OpNe, OpContains, OpStarts, OpEnds, OpNull, OpNotNull}
	}
	return []Op{OpEq, OpNe, OpGt, OpGe, OpLt, OpLe, OpNull, OpNotNull}
}
