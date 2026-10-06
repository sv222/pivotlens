package ui

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type exportPanel struct {
	spec    engine.QuerySpec
	input   textinput.Model
	run     func(engine.QuerySpec, string) tea.Cmd
	confirm bool
	errText string
}

func newExportPanel(spec engine.QuerySpec, run func(engine.QuerySpec, string) tea.Cmd) *exportPanel {
	ti := textinput.New()
	ti.Prompt = "export to: "
	ti.Focus()
	return &exportPanel{spec: spec, input: ti, run: run}
}

func exportFormat(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		return "csv", true
	case ".parquet":
		return "parquet", true
	}
	return "", false
}

func (p *exportPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	if p.confirm {
		switch key.String() {
		case "y":
			return p, PanelAction{Cmd: p.run(p.spec, p.input.Value())}
		case "n", "esc":
			p.confirm = false
		}
		return p, PanelAction{}
	}
	switch key.String() {
	case "esc", "q":
		return p, PanelAction{Close: true}
	case "enter":
		path := p.input.Value()
		if _, ok := exportFormat(path); !ok {
			p.errText = "supported extensions: .csv, .parquet"
			return p, PanelAction{}
		}
		p.errText = ""
		if _, err := os.Stat(path); err == nil {
			p.confirm = true
			return p, PanelAction{}
		}
		return p, PanelAction{Cmd: p.run(p.spec, path)}
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, PanelAction{Cmd: cmd}
}

func (p *exportPanel) View(s Styles, w int) string {
	if p.confirm {
		return "overwrite? [y] yes  [n] no"
	}
	line := p.input.View()
	if p.errText != "" {
		line += "  " + s.ErrorBar.Render(p.errText)
	}
	return line
}
