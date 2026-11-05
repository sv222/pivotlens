package engine

import (
	"fmt"
	"strings"
)

type ColumnMeta struct {
	Name string
	Type string
}

type ColSpec struct {
	Name   string `json:"name"`
	Alias  string `json:"alias,omitempty"`
	Cast   string `json:"cast,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
	Trim   bool   `json:"trim,omitempty"`
	NullAs string `json:"null_as,omitempty"`
}

func (c ColSpec) Display() string {
	if c.Alias != "" {
		return c.Alias
	}
	return c.Name
}

type SortKey struct {
	Col  string `json:"col"`
	Desc bool   `json:"desc,omitempty"`
}

type Measure struct {
	Agg string `json:"agg"`
	Col string `json:"col,omitempty"`
}

type PivotSpec struct {
	Rows     []string  `json:"rows"`
	On       string    `json:"on,omitempty"`
	Measures []Measure `json:"measures"`
	Totals   bool      `json:"totals,omitempty"`
	Percent  string    `json:"percent,omitempty"`
	Keys     []string  `json:"keys,omitempty"`
	Null     bool      `json:"null,omitempty"`
}

type QuerySpec struct {
	Cols     []ColSpec    `json:"cols"`
	Filter   string       `json:"filter,omitempty"`
	Filters  []FilterSpec `json:"filters,omitempty"`
	DropNull []string     `json:"drop_null,omitempty"`
	Sort     []SortKey    `json:"sort,omitempty"`
	Pivot    *PivotSpec   `json:"pivot,omitempty"`
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
	Col     string `json:"col"`
	Op      Op     `json:"op"`
	Value   string `json:"value,omitempty"`
	ColType string `json:"col_type,omitempty"`
}

func (f FilterSpec) String() string {
	if f.Op == OpNull || f.Op == OpNotNull {
		return f.Col + " " + f.Op.String()
	}
	return f.Col + " " + f.Op.String() + " " + f.Value
}

func (o Op) MarshalText() ([]byte, error) {
	if o < OpEq || o > OpNotNull {
		return nil, fmt.Errorf("unknown filter operator %d", int(o))
	}
	return []byte(o.String()), nil
}

func (o *Op) UnmarshalText(b []byte) error {
	for c := OpEq; c <= OpNotNull; c++ {
		if c.String() == string(b) {
			*o = c
			return nil
		}
	}
	return fmt.Errorf("unknown filter operator %q", b)
}

func OpsFor(t string) []Op {
	if strings.HasPrefix(t, "VARCHAR") || t == "TEXT" || t == "STRING" {
		return []Op{OpEq, OpNe, OpContains, OpStarts, OpEnds, OpNull, OpNotNull}
	}
	return []Op{OpEq, OpNe, OpGt, OpGe, OpLt, OpLe, OpNull, OpNotNull}
}
