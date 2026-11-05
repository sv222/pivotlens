package ui

import (
	"context"
	"path/filepath"
	"slices"
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
		m.panel = newSortPanel(m.headers, m.colCursor, m.spec.Clone())
		return m, nil
	case "/":
		if m.spec.Pivot != nil {
			m.errText = "filters are unavailable on a pivot view"
			return m, nil
		}
		m.panel = newFilterPanel(m.spec.Clone())
		return m, nil
	case "p":
		if m.spec.Pivot != nil {
			m.errText = "already a pivot view, press u to go back"
			return m, nil
		}
		if len(m.headers) == 0 {
			return m, nil
		}
		m.pivotReq++
		id := m.pivotReq
		m.panel = newPivotPanel(m.headers, m.sess.NumericFields(m.spec), m.spec.Clone(), func(q engine.QuerySpec) tea.Cmd {
			return pivotCmd(m.ctx, m.sess, q, id)
		})
		return m, nil
	case "f", "F":
		if m.spec.Pivot != nil {
			m.errText = "filters are unavailable on a pivot view"
			return m, nil
		}
		if m.colCursor >= len(m.headers) {
			m.errText = "column index out of range"
			return m, nil
		}
		if msg.String() == "f" {
			col := m.headers[m.colCursor]
			ct := m.colType(col)
			if ct == "" {
				ct = "VARCHAR"
			}
			m.panel = newFilterTypedPanel(m.spec.Clone(), col, ct)
		} else {
			m.panel = newFiltersPanel(m.spec.Clone())
		}
		return m, nil
	case "c":
		if m.spec.Pivot != nil {
			m.errText = "columns are unavailable on a pivot view"
			return m, nil
		}
		m.panel = newColumnsPanel(m.spec.Clone())
		return m, nil
	case "e":
		m.exportReq++
		id := m.exportReq
		spec := m.spec.Clone()
		m.panel = newExportPanel(spec, func(q engine.QuerySpec, path string) tea.Cmd {
			format, ok := exportFormat(path)
			if !ok {
				return nil
			}
			return exportCmd(context.Background(), m.sess, q, path, format, id)
		})
		return m, nil
	case "w", "o":
		m.viewReq++
		id := m.viewReq
		if msg.String() == "w" {
			spec := m.spec.Clone()
			m.panel = newViewPanel(true, func(path string) tea.Cmd { return saveViewCmd(spec, path, id) })
		} else {
			m.panel = newViewPanel(false, func(path string) tea.Cmd { return loadViewCmd(m.ctx, m.sess, path, id) })
		}
		return m, nil
	case "i":
		if m.spec.Pivot != nil {
			m.errText = "stats are unavailable on a pivot view"
			return m, nil
		}
		if m.colCursor >= len(m.headers) {
			m.errText = "column index out of range"
			return m, nil
		}
		col := m.headers[m.colCursor]
		m.statsReq++
		m.panel = newStatsPanel(col)
		return m, statsCmd(m.ctx, m.sess, m.spec, m.metaName(col), m.statsReq)
	case "t":
		return m.repivot(func(p *engine.PivotSpec) { p.Totals = !p.Totals })
	case "%":
		if m.spec.Pivot != nil {
			for _, v := range m.spec.Pivot.Measures {
				if !slices.Contains(engine.PercentAggs, v.Agg) {
					m.errText = "percent works with count and sum only"
					return m, nil
				}
			}
		}
		return m.repivot(func(p *engine.PivotSpec) {
			modes := append([]string{""}, engine.PivotPercents...)
			p.Percent = modes[(slices.Index(modes, p.Percent)+1)%len(modes)]
		})
	case "u":
		return m.undo()
	}
	return m, nil
}

func (m Model) repivot(change func(*engine.PivotSpec)) (tea.Model, tea.Cmd) {
	if m.spec.Pivot == nil {
		m.errText = "not a pivot view, press p to build one"
		return m, nil
	}
	q := m.spec.Clone()
	change(q.Pivot)
	cols := q.Pivot.Columns()
	q.Sort = slices.DeleteFunc(q.Sort, func(k engine.SortKey) bool { return !slices.Contains(cols, k.Col) })
	m.pivotReq++
	return m, pivotCmd(m.ctx, m.sess, q, m.pivotReq)
}

func (m Model) metaName(display string) string {
	for _, c := range m.spec.Cols {
		if c.Display() == display {
			return c.Name
		}
	}
	return display
}

func (m Model) colType(display string) string {
	name := display
	for _, c := range m.spec.Cols {
		if c.Display() == display {
			name = c.Name
			break
		}
	}
	for _, c := range m.sess.Columns() {
		if c.Name == name {
			return c.Type
		}
	}
	return ""
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
	lo := m.blockOff
	hi := m.blockOff + len(m.rows) - bh
	if hi < lo {
		hi = lo
	}
	if top < lo {
		top = lo
	}
	if top > hi {
		top = hi
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
	title := "pivotlens │ " + filepath.ToSlash(m.sess.Path) + " │ " + rows
	return m.styles.Title.Render(pad(title, m.w))
}

func (m Model) renderStatus() string {
	if m.errText != "" {
		msg := m.errText
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		return m.styles.ErrorBar.Render(pad("error: "+msg, m.w))
	}
	parts := []string{"row " + strconv.Itoa(m.cursor+1)}
	if m.spec.Pivot != nil {
		parts = append(parts, "pivot: "+m.spec.Pivot.String())
	}
	if m.spec.Filter != "" {
		parts = append(parts, "filter: "+m.spec.Filter)
	}
	for _, f := range m.spec.Filters {
		parts = append(parts, f.String())
	}
	if len(m.spec.Sort) > 0 {
		d := "asc"
		if m.spec.Sort[0].Desc {
			d = "desc"
		}
		parts = append(parts, "sort: "+m.spec.Sort[0].Col+" "+d)
	}
	if m.notice != "" {
		parts = append(parts, m.notice)
	}
	if m.spec.Pivot != nil {
		parts = append(parts, "[t] totals  [%] percent  [s] sort  [u] back  [q] quit")
	} else {
		parts = append(parts, "[/] filter  [s] sort  [u] undo  [q] quit")
	}
	return m.styles.Status.Render(pad(strings.Join(parts, "  │  "), m.w))
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
	if m.panel != nil {
		b.WriteString("\n" + m.panel.View(m.styles, m.w))
	}
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
