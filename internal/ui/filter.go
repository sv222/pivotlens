package ui

import (
	tea "charm.land/bubbletea/v2"
)

func (m Model) applyFilter() (tea.Model, tea.Cmd) {
	v := m.filter.Value()
	if v == m.spec.Filter {
		return m, nil
	}
	if !m.filterPushed {
		m = m.push()
		m.filterPushed = true
	}
	m.spec.Filter = v
	return m.reload()
}

func (m Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeGrid
		m.filter.SetValue(m.spec.Filter)
		m.filter.Blur()
		return m, nil
	case "enter":
		m.mode = ModeGrid
		m.filter.Blur()
		return m.applyFilter()
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.keySeq++
	return m, tea.Batch(cmd, debounceCmd(m.keySeq))
}
