package ui

import (
	"reflect"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type Panel interface {
	Update(tea.Msg) (Panel, PanelAction)
	View(Styles, int) string
}

type PanelAction struct {
	Close    bool
	Spec     *engine.QuerySpec
	Push     bool
	Debounce bool
	Cmd      tea.Cmd
}

func (m Model) applyPanelAction(a PanelAction) (tea.Model, tea.Cmd) {
	if a.Close {
		m.panel = nil
		m.pending = nil
	}
	if a.Spec == nil {
		if a.Cmd != nil {
			return m, a.Cmd
		}
		return m, nil
	}
	if reflect.DeepEqual(m.spec, *a.Spec) {
		return m, nil
	}
	if a.Debounce {
		if a.Push {
			m = m.push()
		}
		m.pending = a.Spec
		m.keySeq++
		return m, debounceCmd(m.keySeq)
	}
	if a.Push {
		m = m.push()
	}
	m.spec = *a.Spec
	m.pending = nil
	return m.reload()
}
