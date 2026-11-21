package ui

import (
	"maps"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

// toggleMark marks or unmarks the row under the cursor of a value counts
// view for drill-down.
func (m Model) toggleMark() (tea.Model, tea.Cmd) {
	idx := m.cursor - m.blockOff
	if idx < 0 || idx >= len(m.rows) {
		return m, nil
	}
	v := m.rows[idx][0]
	if v == engine.FreqOtherText {
		m.errText = "(other) cannot be selected"
		return m, nil
	}
	if m.marked == nil {
		m.marked = make(map[int]string)
	}
	if _, ok := m.marked[m.cursor]; ok {
		delete(m.marked, m.cursor)
	} else {
		m.marked[m.cursor] = v
	}
	return m, nil
}

// freqDrill closes the value counts view and adds one typed filter for the
// row under the cursor, or for the marked rows.
func (m Model) freqDrill() (tea.Model, tea.Cmd) {
	var vals []string
	if len(m.marked) > 0 {
		for _, k := range slices.Sorted(maps.Keys(m.marked)) {
			if v := m.marked[k]; v != engine.FreqOtherText {
				vals = append(vals, v)
			}
		}
	} else {
		idx := m.cursor - m.blockOff
		if idx < 0 || idx >= len(m.rows) {
			return m, nil
		}
		v := m.rows[idx][0]
		if v == engine.FreqOtherText {
			m.errText = "(other) cannot be selected"
			return m, nil
		}
		vals = []string{v}
	}
	if len(vals) == 0 {
		m.errText = "(other) cannot be selected"
		return m, nil
	}
	f, ok := m.freqFilter(vals)
	if !ok {
		return m, nil
	}
	parent := m.spec
	if len(m.hist) > 0 {
		parent = m.hist[len(m.hist)-1]
	}
	q := parent.Clone()
	q.Freq = nil
	q.Filters = append(q.Filters, f)
	m.spec = q
	m.marked = nil
	return m.reload()
}

func (m Model) freqFilter(vals []string) (engine.FilterSpec, bool) {
	col := m.spec.Freq.Col
	ct := m.colType(col)
	if len(vals) == 1 {
		switch vals[0] {
		case engine.FreqNullText:
			return engine.FilterSpec{Col: col, Op: engine.OpNull}, true
		case engine.FreqEmptyText:
			return engine.FilterSpec{Col: col, Op: engine.OpEq, ColType: "VARCHAR"}, true
		}
		if ct == "" {
			ct = "VARCHAR"
		}
		return engine.FilterSpec{Col: col, Op: engine.OpEq, Value: vals[0], ColType: ct}, true
	}
	var in []string
	for _, v := range vals {
		switch v {
		case engine.FreqNullText, engine.FreqOtherText:
		case engine.FreqEmptyText:
			in = append(in, "")
		default:
			in = append(in, v)
		}
	}
	if len(in) == 0 {
		return engine.FilterSpec{}, false
	}
	if ct == "" {
		ct = "VARCHAR"
	}
	return engine.FilterSpec{Col: col, Op: engine.OpIn, Values: in, ColType: ct}, true
}
