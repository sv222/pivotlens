package engine

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

type PivotSpec struct {
	Rows   []string
	On     string
	Agg    string
	AggCol string
}

type QuerySpec struct {
	Cols     []ColSpec
	Filter   string
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
	n.Sort = append([]SortKey(nil), q.Sort...)
	if q.Pivot != nil {
		p := *q.Pivot
		p.Rows = append([]string(nil), q.Pivot.Rows...)
		n.Pivot = &p
	}
	return n
}
