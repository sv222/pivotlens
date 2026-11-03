package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

const (
	noField  = "(none)"
	addValue = "add value"
	doneText = "done"
)

var pivotStages = []string{"rows", "columns", "aggregate", "values", "next"}

type pivotPanel struct {
	headers  []string
	numeric  map[string]bool
	spec     engine.QuerySpec
	run      func(engine.QuerySpec) tea.Cmd
	stage    int
	idx      int
	picks    [3]string
	measures []engine.Measure
}

func newPivotPanel(headers []string, numeric map[string]bool, spec engine.QuerySpec, run func(engine.QuerySpec) tea.Cmd) *pivotPanel {
	return &pivotPanel{headers: headers, numeric: numeric, spec: spec, run: run}
}

func (p *pivotPanel) options() []string {
	switch p.stage {
	case 0:
		return p.headers
	case 1:
		out := []string{noField}
		for _, h := range p.headers {
			if h != p.picks[0] {
				out = append(out, h)
			}
		}
		return out
	case 2:
		return engine.PivotAggs
	case 4:
		return []string{doneText, addValue}
	}
	switch p.picks[2] {
	case "count":
		return append([]string{noField}, p.headers...)
	case "sum", "avg":
		var out []string
		for _, h := range p.headers {
			if p.numeric[h] {
				out = append(out, h)
			}
		}
		return out
	}
	return p.headers
}

func (p *pivotPanel) finish() PanelAction {
	pv := engine.PivotSpec{Rows: []string{p.picks[0]}, Measures: p.measures}
	if p.picks[1] != noField {
		pv.On = p.picks[1]
	}
	clone := p.spec.Clone()
	clone.Sort = nil
	clone.Pivot = &pv
	return PanelAction{Close: true, Cmd: p.run(clone)}
}

func (p *pivotPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	opts := p.options()
	switch key.String() {
	case "esc", "q":
		switch {
		case p.stage == 0:
			return p, PanelAction{Close: true}
		case p.stage == 4:
			p.measures = p.measures[:len(p.measures)-1]
			p.stage = 3
		case p.stage == 2 && len(p.measures) > 0:
			p.stage = 4
		default:
			p.stage--
		}
		p.idx = 0
	case "j", "down", "l", "right":
		if p.idx < len(opts)-1 {
			p.idx++
		}
	case "k", "up", "h", "left":
		if p.idx > 0 {
			p.idx--
		}
	case "enter":
		if p.idx >= len(opts) {
			return p, PanelAction{Close: true}
		}
		switch {
		case p.stage == 4 && opts[p.idx] == doneText:
			return p, p.finish()
		case p.stage == 4:
			p.stage = 2
		case p.stage == 3:
			m := engine.Measure{Agg: p.picks[2]}
			if opts[p.idx] != noField {
				m.Col = opts[p.idx]
			}
			p.measures = append(p.measures, m)
			p.stage = 4
		default:
			p.picks[p.stage] = opts[p.idx]
			p.stage++
		}
		p.idx = 0
	}
	return p, PanelAction{}
}

func (p *pivotPanel) View(s Styles, w int) string {
	var b strings.Builder
	b.WriteString("pivot " + pivotStages[p.stage] + ": ")
	if len(p.measures) > 0 {
		ls := make([]string, len(p.measures))
		for i, m := range p.measures {
			ls[i] = m.Label()
		}
		b.WriteString("[" + strings.Join(ls, ", ") + "] ")
	}
	opts := p.options()
	if len(opts) == 0 {
		b.WriteString(" no numeric fields ")
	}
	for i, o := range opts {
		if i == p.idx {
			b.WriteString(s.Cursor.Render(" " + o + " "))
		} else {
			b.WriteString(" " + o + " ")
		}
	}
	b.WriteString("   [j/k] pick  [enter] next  [esc] back")
	return b.String()
}
