package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

const noField = "(none)"

var pivotStages = []string{"rows", "columns", "aggregate", "values"}

type pivotPanel struct {
	headers []string
	spec    engine.QuerySpec
	run     func(engine.QuerySpec) tea.Cmd
	stage   int
	idx     int
	picks   [4]string
}

func newPivotPanel(headers []string, spec engine.QuerySpec, run func(engine.QuerySpec) tea.Cmd) *pivotPanel {
	return &pivotPanel{headers: headers, spec: spec, run: run}
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
	}
	if p.picks[2] == "count" {
		return append([]string{noField}, p.headers...)
	}
	return p.headers
}

func (p *pivotPanel) finish() PanelAction {
	pv := engine.PivotSpec{Rows: []string{p.picks[0]}, Agg: p.picks[2]}
	if p.picks[1] != noField {
		pv.On = p.picks[1]
	}
	if p.picks[3] != noField {
		pv.AggCol = p.picks[3]
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
		if p.stage == 0 {
			return p, PanelAction{Close: true}
		}
		p.stage--
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
		p.picks[p.stage] = opts[p.idx]
		if p.stage == len(pivotStages)-1 {
			return p, p.finish()
		}
		p.stage++
		p.idx = 0
	}
	return p, PanelAction{}
}

func (p *pivotPanel) View(s Styles, w int) string {
	var b strings.Builder
	b.WriteString("pivot " + pivotStages[p.stage] + ": ")
	for i, o := range p.options() {
		if i == p.idx {
			b.WriteString(s.Cursor.Render(" " + o + " "))
		} else {
			b.WriteString(" " + o + " ")
		}
	}
	b.WriteString("   [j/k] pick  [enter] next  [esc] back")
	return b.String()
}
