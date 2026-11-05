package ui

import (
	"os"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type viewPanel struct {
	input   textinput.Model
	save    bool
	run     func(string) tea.Cmd
	confirm bool
}

func newViewPanel(save bool, run func(string) tea.Cmd) *viewPanel {
	ti := textinput.New()
	ti.Prompt = "open view: "
	if save {
		ti.Prompt = "save view to: "
	}
	ti.Focus()
	return &viewPanel{input: ti, save: save, run: run}
}

func (p *viewPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	if p.confirm {
		switch key.String() {
		case "y":
			return p, PanelAction{Cmd: p.run(p.input.Value())}
		case "n", "esc":
			p.confirm = false
		}
		return p, PanelAction{}
	}
	switch key.String() {
	case "esc":
		return p, PanelAction{Close: true}
	case "enter":
		path := p.input.Value()
		if path == "" {
			return p, PanelAction{}
		}
		if _, err := os.Stat(path); err == nil && p.save {
			p.confirm = true
			return p, PanelAction{}
		}
		return p, PanelAction{Cmd: p.run(path)}
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, PanelAction{Cmd: cmd}
}

func (p *viewPanel) View(s Styles, w int) string {
	if p.confirm {
		return "overwrite? [y] yes  [n] no"
	}
	return p.input.View()
}
