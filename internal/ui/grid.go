package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func (m Model) bodyHeight() int {
	h := m.h - 4
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) gotoRow(r int) (tea.Model, tea.Cmd) {
	if r < 0 {
		r = 0
	}
	if m.counted && int64(r) >= m.total {
		r = int(m.total) - 1
		if r < 0 {
			r = 0
		}
	}
	m.cursor = r
	b := blockFor(r)
	if b == m.blockOff {
		return m, nil
	}
	m.blockOff = b
	m.pageReq++
	return m, fetchPageCmd(m.ctx, m.sess, m.spec, m.pageReq, PageSize, b)
}

func (m Model) syncColStart() Model {
	if m.colCursor < m.colStart {
		m.colStart = m.colCursor
		return m
	}
	for {
		n := Window(m.widths, m.colStart, m.w)
		if n == 0 || m.colCursor < m.colStart+n || m.colStart >= len(m.widths)-1 {
			return m
		}
		m.colStart++
	}
}

func (m Model) updateGrid(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		return m.gotoRow(m.cursor + 1)
	case "k", "up":
		return m.gotoRow(m.cursor - 1)
	case "ctrl+d":
		return m.gotoRow(m.cursor + m.bodyHeight())
	case "ctrl+u":
		return m.gotoRow(m.cursor - m.bodyHeight())
	case "g":
		return m.gotoRow(0)
	case "G":
		if m.counted {
			return m.gotoRow(int(m.total) - 1)
		}
		return m, nil
	case "h", "left":
		if m.colCursor > 0 {
			m.colCursor--
			m = m.syncColStart()
		}
		return m, nil
	case "l", "right":
		if m.colCursor < len(m.headers)-1 {
			m.colCursor++
			m = m.syncColStart()
		}
		return m, nil
	case "s":
		m.mode = ModeSort
		m.sortIdx = m.colCursor
		return m, nil
	case "/":
		m.mode = ModeFilter
		m.filter.SetValue(m.spec.Filter)
		m.filter.Focus()
		return m, nil
	case "u":
		return m.undo()
	}
	return m, nil
}

func pad(s string, w int) string {
	n := lipgloss.Width(s)
	if n == w {
		return s
	}
	if n < w {
		return s + strings.Repeat(" ", w-n)
	}
	if w <= 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > w-1 {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + strings.Repeat(" ", w-1-used) + "…"
}

func (m Model) topRow() int {
	bh := m.bodyHeight()
	top := m.cursor - bh/2
	if top < 0 {
		top = 0
	}
	return top
}

func (m Model) renderHeader() string {
	n := Window(m.widths, m.colStart, m.w)
	cells := make([]string, 0, n)
	for i := m.colStart; i < m.colStart+n; i++ {
		cells = append(cells, pad(m.headers[i], m.widths[i]))
	}
	return m.styles.Header.Render(strings.Join(cells, " │ "))
}

func (m Model) renderBody() string {
	n := Window(m.widths, m.colStart, m.w)
	bh := m.bodyHeight()
	top := m.topRow()
	lines := make([]string, 0, bh)
	for r := top; r < top+bh; r++ {
		idx := r - m.blockOff
		if idx < 0 || idx >= len(m.rows) {
			lines = append(lines, "")
			continue
		}
		row := m.rows[idx]
		cells := make([]string, 0, n)
		for i := m.colStart; i < m.colStart+n && i < len(row); i++ {
			c := pad(row[i], m.widths[i])
			if row[i] == engine.NullText {
				c = m.styles.Null.Render(c)
			}
			cells = append(cells, c)
		}
		line := strings.Join(cells, " │ ")
		if r == m.cursor {
			line = m.styles.Cursor.Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderTitle() string {
	rows := "counting…"
	if m.counted {
		rows = strconv.FormatInt(m.total, 10) + " rows"
	}
	return m.styles.Title.Render("pivotlens │ " + m.sess.Path + " │ " + rows)
}

func (m Model) renderStatus() string {
	if m.errText != "" {
		return m.styles.ErrorBar.Render("error: " + m.errText)
	}
	parts := []string{"row " + strconv.Itoa(m.cursor+1)}
	if m.spec.Filter != "" {
		parts = append(parts, "filter: "+m.spec.Filter)
	}
	if len(m.spec.Sort) > 0 {
		d := "asc"
		if m.spec.Sort[0].Desc {
			d = "desc"
		}
		parts = append(parts, "sort: "+m.spec.Sort[0].Col+" "+d)
	}
	parts = append(parts, "[/] filter  [s] sort  [u] undo  [q] quit")
	return m.styles.Status.Render(strings.Join(parts, "  │  "))
}

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString(m.renderTitle())
	b.WriteString("\n")
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderBody())
	b.WriteString("\n")
	b.WriteString(m.renderStatus())
	switch m.mode {
	case ModeFilter:
		b.WriteString("\n" + m.filter.View())
	case ModeSort:
		b.WriteString("\n" + m.sortView())
	}
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
