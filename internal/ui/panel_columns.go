package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type columnsPanel struct {
	spec     engine.QuerySpec
	orig     []string
	idx      int
	renaming bool
	input    textinput.Model
	errText  string
}

func newColumnsPanel(spec engine.QuerySpec) *columnsPanel {
	ti := textinput.New()
	ti.Prompt = "rename: "
	orig := make([]string, len(spec.Cols))
	for i, c := range spec.Cols {
		orig[i] = c.Display()
	}
	return &columnsPanel{spec: spec, orig: orig, input: ti}
}

func (p *columnsPanel) dupDisplay() string {
	seen := map[string]bool{}
	for _, c := range p.spec.Cols {
		if c.Hidden {
			continue
		}
		d := c.Display()
		if seen[d] {
			return d
		}
		seen[d] = true
	}
	return ""
}

func (p *columnsPanel) resolve(col string) (int, bool) {
	for j, o := range p.orig {
		if o == col && !p.spec.Cols[j].Hidden {
			return j, true
		}
	}
	for j, o := range p.orig {
		if o == col {
			return j, true
		}
	}
	for j, c := range p.spec.Cols {
		if c.Display() == col && !c.Hidden {
			return j, true
		}
	}
	return 0, false
}

func (p *columnsPanel) commit() PanelAction {
	vis := 0
	for _, c := range p.spec.Cols {
		if !c.Hidden {
			vis++
		}
	}
	if vis == 0 {
		p.errText = engine.ErrNoColumns.Error()
		return PanelAction{}
	}
	if d := p.dupDisplay(); d != "" {
		p.errText = "duplicate column name " + d
		return PanelAction{}
	}
	sort := make([]engine.SortKey, 0, len(p.spec.Sort))
	for _, k := range p.spec.Sort {
		if j, ok := p.resolve(k.Col); ok {
			if p.spec.Cols[j].Hidden {
				continue
			}
			k.Col = p.spec.Cols[j].Display()
		}
		sort = append(sort, k)
	}
	p.spec.Sort = sort
	filters := make([]engine.FilterSpec, 0, len(p.spec.Filters))
	for _, f := range p.spec.Filters {
		if j, ok := p.resolve(f.Col); ok {
			if p.spec.Cols[j].Hidden {
				continue
			}
			f.Col = p.spec.Cols[j].Display()
		}
		filters = append(filters, f)
	}
	p.spec.Filters = filters
	return PanelAction{Spec: &p.spec, Push: true, Close: true}
}

func (p *columnsPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	if p.renaming {
		switch key.String() {
		case "esc":
			p.renaming = false
			p.input.SetValue("")
			return p, PanelAction{}
		case "enter":
			v := p.input.Value()
			if v != "" {
				p.spec.Cols[p.idx].Alias = v
			}
			p.renaming = false
			p.input.SetValue("")
			return p, PanelAction{}
		}
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return p, PanelAction{Cmd: cmd}
	}
	switch key.String() {
	case "esc", "q":
		return p, PanelAction{Close: true}
	case "j", "down":
		if p.idx < len(p.spec.Cols)-1 {
			p.idx++
		}
	case "k", "up":
		if p.idx > 0 {
			p.idx--
		}
	case "space":
		p.spec.Cols[p.idx].Hidden = !p.spec.Cols[p.idx].Hidden
	case "J":
		if p.idx < len(p.spec.Cols)-1 {
			p.spec.Cols[p.idx], p.spec.Cols[p.idx+1] = p.spec.Cols[p.idx+1], p.spec.Cols[p.idx]
			p.idx++
		}
	case "K":
		if p.idx > 0 {
			p.spec.Cols[p.idx], p.spec.Cols[p.idx-1] = p.spec.Cols[p.idx-1], p.spec.Cols[p.idx]
			p.idx--
		}
	case "r":
		p.renaming = true
		p.input.SetValue(p.spec.Cols[p.idx].Alias)
		p.input.Focus()
	case "enter":
		return p, p.commit()
	}
	return p, PanelAction{}
}

func (p *columnsPanel) View(s Styles, w int) string {
	if p.renaming {
		return p.input.View() + "   [enter] commit  [esc] revert"
	}
	var b strings.Builder
	b.WriteString("columns: ")
	for i, c := range p.spec.Cols {
		label := c.Display()
		if c.Alias != "" {
			label = c.Alias + " (" + c.Name + ")"
		}
		if c.Hidden {
			label = s.Null.Render(label)
		}
		if i == p.idx {
			b.WriteString(s.Cursor.Render(" " + label + " "))
		} else {
			b.WriteString(" " + label + " ")
		}
	}
	b.WriteString("   [space] hide  [J/K] move  [r] rename  [enter] apply  [esc] cancel")
	if p.errText != "" {
		b.WriteString("\n" + s.ErrorBar.Render(p.errText))
	}
	return b.String()
}
