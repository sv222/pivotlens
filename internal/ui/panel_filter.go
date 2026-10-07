package ui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type filterPanel struct {
	spec   engine.QuerySpec
	input  textinput.Model
	pushed bool
}

func newFilterPanel(spec engine.QuerySpec) *filterPanel {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.SetValue(spec.Filter)
	ti.Focus()
	return &filterPanel{spec: spec, input: ti}
}

func (p *filterPanel) action(final bool) PanelAction {
	v := p.input.Value()
	if v == p.spec.Filter {
		if final {
			return PanelAction{Close: true}
		}
		return PanelAction{}
	}
	clone := p.spec.Clone()
	clone.Filter = v
	a := PanelAction{Spec: &clone, Push: !p.pushed}
	if final {
		a.Close = true
	} else {
		a.Debounce = true
	}
	p.pushed = true
	return a
}

func (p *filterPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	switch key.String() {
	case "esc":
		p.input.SetValue(p.spec.Filter)
		return p, PanelAction{Close: true}
	case "enter":
		return p, p.action(true)
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	a := p.action(false)
	a.Cmd = cmd
	return p, a
}

func (p *filterPanel) View(s Styles, w int) string {
	return p.input.View()
}
