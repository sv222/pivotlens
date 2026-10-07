package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type statsPanel struct {
	col   string
	stats engine.ColumnStats
	done  bool
	err   string
}

func newStatsPanel(col string) *statsPanel {
	return &statsPanel{col: col}
}

func (p *statsPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	switch msg := msg.(type) {
	case statsMsg:
		p.done = true
		if msg.err != nil {
			p.err = msg.err.Error()
		} else {
			p.stats = msg.stats
		}
		return p, PanelAction{}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return p, PanelAction{Close: true}
		}
	}
	return p, PanelAction{}
}

func (p *statsPanel) View(s Styles, w int) string {
	if p.err != "" {
		return s.ErrorBar.Render(p.err)
	}
	if !p.done {
		return "measuring " + p.col + "..."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s: rows %d  nulls %d  distinct %d  min %s  max %s",
		p.col, p.stats.Rows, p.stats.Nulls, p.stats.Distinct, p.stats.Min, p.stats.Max)
	if p.stats.Numeric {
		fmt.Fprintf(&b, "  avg %s  sum %s", p.stats.Avg, p.stats.Sum)
	}
	b.WriteString("   [esc] close")
	return b.String()
}
