package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func (m Model) applySort(desc bool) (tea.Model, tea.Cmd) {
	if m.sortIdx < 0 || m.sortIdx >= len(m.headers) {
		m.mode = ModeGrid
		return m, nil
	}
	m = m.push()
	m.spec.Sort = []engine.SortKey{{Col: m.headers[m.sortIdx], Desc: desc}}
	m.mode = ModeGrid
	return m.reload()
}

func (m Model) updateSort(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = ModeGrid
		return m, nil
	case "j", "down":
		if m.sortIdx < len(m.headers)-1 {
			m.sortIdx++
		}
		return m, nil
	case "k", "up":
		if m.sortIdx > 0 {
			m.sortIdx--
		}
		return m, nil
	case "a", "enter":
		return m.applySort(false)
	case "d":
		return m.applySort(true)
	case "x":
		m = m.push()
		m.spec.Sort = nil
		m.mode = ModeGrid
		return m.reload()
	}
	return m, nil
}

func (m Model) sortView() string {
	var b strings.Builder
	b.WriteString("sort: ")
	for i, h := range m.headers {
		if i == m.sortIdx {
			b.WriteString(m.styles.Cursor.Render(" " + h + " "))
		} else {
			b.WriteString(" " + h + " ")
		}
	}
	b.WriteString("   [a] asc  [d] desc  [x] clear  [esc] cancel")
	return b.String()
}
