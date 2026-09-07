package ui

import tea "charm.land/bubbletea/v2"

func (m Model) updateSort(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.mode = ModeGrid
	return m, nil
}

func (m Model) sortView() string { return "" }

func (m Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.mode = ModeGrid
	return m, nil
}

func (m Model) applyFilter() (tea.Model, tea.Cmd) { return m, nil }
