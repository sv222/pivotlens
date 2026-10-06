package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type sortPanel struct {
	headers []string
	spec    engine.QuerySpec
	idx     int
}

func newSortPanel(headers []string, idx int, spec engine.QuerySpec) *sortPanel {
	return &sortPanel{headers: headers, spec: spec, idx: idx}
}

func (p *sortPanel) apply(desc bool) PanelAction {
	if p.idx < 0 || p.idx >= len(p.headers) {
		return PanelAction{Close: true}
	}
	clone := p.spec.Clone()
	clone.Sort = []engine.SortKey{{Col: p.headers[p.idx], Desc: desc}}
	return PanelAction{Spec: &clone, Push: true, Close: true}
}

func (p *sortPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	switch key.String() {
	case "esc", "q":
		return p, PanelAction{Close: true}
	case "j", "down":
		if p.idx < len(p.headers)-1 {
			p.idx++
		}
	case "k", "up":
		if p.idx > 0 {
			p.idx--
		}
	case "a", "enter":
		return p, p.apply(false)
	case "d":
		return p, p.apply(true)
	case "x":
		clone := p.spec.Clone()
		clone.Sort = nil
		return p, PanelAction{Spec: &clone, Push: true, Close: true}
	}
	return p, PanelAction{}
}

func (p *sortPanel) View(s Styles, w int) string {
	var b strings.Builder
	b.WriteString("sort: ")
	for i, h := range p.headers {
		if i == p.idx {
			b.WriteString(s.Cursor.Render(" " + h + " "))
		} else {
			b.WriteString(" " + h + " ")
		}
	}
	b.WriteString("   [a] asc  [d] desc  [x] clear  [esc] cancel")
	return b.String()
}
