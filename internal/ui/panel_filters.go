package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

type filtersPanel struct {
	spec engine.QuerySpec
	del  map[int]bool
	idx  int
}

func newFiltersPanel(spec engine.QuerySpec) *filtersPanel {
	return &filtersPanel{spec: spec, del: map[int]bool{}}
}

func (p *filtersPanel) rows() int {
	n := len(p.spec.Filters)
	if p.spec.Filter != "" {
		n++
	}
	return n
}

func (p *filtersPanel) Update(msg tea.Msg) (Panel, PanelAction) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, PanelAction{}
	}
	switch key.String() {
	case "esc":
		return p, PanelAction{Close: true}
	case "j", "down":
		if p.idx < p.rows()-1 {
			p.idx++
		}
	case "k", "up":
		if p.idx > 0 {
			p.idx--
		}
	case "d":
		if p.idx < p.rows() {
			p.del[p.idx] = !p.del[p.idx]
		}
	case "enter":
		if len(p.del) == 0 {
			return p, PanelAction{Close: true}
		}
		clone := p.spec.Clone()
		off := 0
		if p.spec.Filter != "" {
			off = 1
			if p.del[0] {
				clone.Filter = ""
			}
		}
		kept := make([]engine.FilterSpec, 0, len(p.spec.Filters))
		for i, f := range p.spec.Filters {
			if p.del[i+off] {
				continue
			}
			kept = append(kept, f)
		}
		clone.Filters = kept
		return p, PanelAction{Spec: &clone, Push: true, Close: true}
	}
	return p, PanelAction{}
}

func (p *filtersPanel) entry(s Styles, i int, label string) string {
	mark := " "
	if p.del[i] {
		mark = "×"
	}
	if i == p.idx {
		return s.Cursor.Render(mark + label)
	}
	if p.del[i] {
		return s.Null.Render(mark + label)
	}
	return mark + label
}

func (p *filtersPanel) View(s Styles, w int) string {
	var b strings.Builder
	b.WriteString("filters: ")
	i := 0
	if p.spec.Filter != "" {
		b.WriteString(p.entry(s, i, "search: "+p.spec.Filter) + " ")
		i++
	}
	for _, f := range p.spec.Filters {
		b.WriteString(p.entry(s, i, f.String()) + " ")
		i++
	}
	b.WriteString("  [d] delete  [enter] apply  [esc] cancel")
	return b.String()
}
