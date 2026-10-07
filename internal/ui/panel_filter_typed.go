package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type filterTypedPanel struct {
	spec     engine.QuerySpec
	col      string
	colType  string
	existing []engine.FilterSpec
	ops      []engine.Op
	idx      int
	stage    int
	op       engine.Op
	editing  *engine.FilterSpec
	value    textinput.Model
	errText  string
}

func newFilterTypedPanel(spec engine.QuerySpec, col, colType string) *filterTypedPanel {
	p := &filterTypedPanel{spec: spec, col: col, colType: colType, ops: engine.OpsFor(colType)}
	for _, f := range spec.Filters {
		if f.Col == col {
			p.existing = append(p.existing, f)
		}
	}
	ti := textinput.New()
	ti.Prompt = "value: "
	p.value = ti
	return p
}

func (p *filterTypedPanel) rows() int { return len(p.existing) + len(p.ops) }

func (p *filterTypedPanel) apply(v string) PanelAction {
	nf := engine.FilterSpec{Col: p.col, Op: p.op, Value: v, ColType: p.colType}
	clone := p.spec.Clone()
	replaced := false
	if p.editing != nil {
		for i := range clone.Filters {
			if clone.Filters[i] == *p.editing {
				clone.Filters[i] = nf
				replaced = true
			}
		}
	}
	if !replaced {
		clone.Filters = append(clone.Filters, nf)
	}
	return PanelAction{Spec: &clone, Push: true, Close: true}
}

func (p *filterTypedPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	if p.stage == 1 {
		switch key.String() {
		case "esc":
			p.stage = 0
			p.errText = ""
			return p, PanelAction{}
		case "enter":
			v := p.value.Value()
			if v == "" {
				p.errText = "value required"
				return p, PanelAction{}
			}
			return p, p.apply(v)
		}
		var cmd tea.Cmd
		p.value, cmd = p.value.Update(msg)
		return p, PanelAction{Cmd: cmd}
	}
	switch key.String() {
	case "esc":
		return p, PanelAction{Close: true}
	case "j", "down":
		if p.idx < p.rows()-1 {
			p.idx++
		}
	case "k", "up":
		if p.idx > 0 {
			p.idx--
		}
	case "enter":
		if p.idx < len(p.existing) {
			e := p.existing[p.idx]
			p.editing = &p.existing[p.idx]
			p.op = e.Op
			if p.op == engine.OpNull || p.op == engine.OpNotNull {
				return p, p.apply("")
			}
			p.value.SetValue(e.Value)
			p.value.Focus()
			p.stage = 1
			return p, PanelAction{}
		}
		p.op = p.ops[p.idx-len(p.existing)]
		p.editing = nil
		if p.op == engine.OpNull || p.op == engine.OpNotNull {
			return p, p.apply("")
		}
		p.value.SetValue("")
		p.value.Focus()
		p.stage = 1
	}
	return p, PanelAction{}
}

func (p *filterTypedPanel) View(s Styles, w int) string {
	if p.stage == 1 {
		line := p.value.View()
		if p.errText != "" {
			line += "  " + s.ErrorBar.Render(p.errText)
		}
		return line
	}
	var b strings.Builder
	b.WriteString("filter " + p.col + ": ")
	for i, f := range p.existing {
		if i == p.idx {
			b.WriteString(s.Cursor.Render(" " + f.String() + " "))
		} else {
			b.WriteString(" " + f.String() + " ")
		}
	}
	if len(p.existing) > 0 {
		b.WriteString(" │ ")
	}
	for i, o := range p.ops {
		if len(p.existing)+i == p.idx {
			b.WriteString(s.Cursor.Render(" " + o.String() + " "))
		} else {
			b.WriteString(" " + o.String() + " ")
		}
	}
	b.WriteString("   [j/k] pick  [enter] select  [esc] cancel")
	return b.String()
}
