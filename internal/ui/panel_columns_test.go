package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func columnsModel(t *testing.T) (Model, *columnsPanel) {
	t.Helper()
	m := gridModel(t)
	p := newColumnsPanel(m.spec.Clone())
	m.panel = p
	return m, p
}

func colIdx(p *columnsPanel, name string) int {
	for i, c := range p.spec.Cols {
		if c.Name == name {
			return i
		}
	}
	return -1
}

func TestColumnsToggleHide(t *testing.T) {
	m, p := columnsModel(t)
	p.idx = colIdx(p, "status")
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	got := next.(Model)
	if !got.panel.(*columnsPanel).spec.Cols[p.idx].Hidden {
		t.Error("space must toggle hidden")
	}
}

func TestColumnsReorder(t *testing.T) {
	m, p := columnsModel(t)
	p.idx = 0
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: 'J'})
	got := next.(Model)
	cp := got.panel.(*columnsPanel)
	if cp.spec.Cols[0].Name != "status" || cp.spec.Cols[1].Name != "region" {
		t.Errorf("J did not move region down: %v", names(cp.spec))
	}
	if cp.idx != 1 {
		t.Errorf("cursor = %d, want 1", cp.idx)
	}
}

func names(q engine.QuerySpec) []string {
	out := make([]string, len(q.Cols))
	for i, c := range q.Cols {
		out[i] = c.Name
	}
	return out
}

func TestColumnsRenameAndApply(t *testing.T) {
	m, p := columnsModel(t)
	p.idx = colIdx(p, "status")
	m.panel = p
	next, _ := m.Update(press("r"))
	got := next.(Model)
	cp := got.panel.(*columnsPanel)
	if !cp.renaming {
		t.Fatal("r must open the rename input")
	}
	cp.input.SetValue("st")
	m.panel = cp
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	cp = next.(Model).panel.(*columnsPanel)
	if cp.renaming {
		t.Fatal("enter must commit the rename")
	}
	m.panel = cp
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = next.(Model)
	if got.panel != nil {
		t.Fatal("enter must close the panel")
	}
	if got.spec.Cols[colIdx(p, "status")].Alias != "st" {
		t.Errorf("alias not applied: %+v", got.spec.Cols)
	}
	if got.headers[1] != "st" {
		t.Errorf("headers = %v", got.headers)
	}
}

func TestColumnsRenameEscReverts(t *testing.T) {
	m, p := columnsModel(t)
	p.idx = colIdx(p, "status")
	m.panel = p
	next, _ := m.Update(press("r"))
	cp := next.(Model).panel.(*columnsPanel)
	cp.input.SetValue("st")
	m.panel = cp
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	cp = next.(Model).panel.(*columnsPanel)
	if cp.renaming {
		t.Fatal("esc must leave rename mode")
	}
	if cp.spec.Cols[colIdx(p, "status")].Alias != "" {
		t.Error("esc must revert the pending alias")
	}
}

func TestColumnsDuplicateDisplayRefused(t *testing.T) {
	m, p := columnsModel(t)
	i := colIdx(p, "status")
	p.spec.Cols[i].Alias = "region"
	p.idx = i
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.panel == nil {
		t.Fatal("duplicate display name must be refused")
	}
	if !strings.Contains(got.panel.(*columnsPanel).errText, "region") {
		t.Errorf("error must name the collision: %q", got.panel.(*columnsPanel).errText)
	}
}

func TestColumnsAllHiddenRefused(t *testing.T) {
	m, p := columnsModel(t)
	for i := range p.spec.Cols {
		p.spec.Cols[i].Hidden = true
	}
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.panel == nil {
		t.Fatal("all-hidden must be refused")
	}
	if got.panel.(*columnsPanel).errText == "" {
		t.Error("refusal must set an error message")
	}
}

func TestColumnsEscDiscards(t *testing.T) {
	m, p := columnsModel(t)
	p.spec.Cols[0].Hidden = true
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got := next.(Model)
	if got.panel != nil {
		t.Error("esc must close the panel")
	}
	if got.spec.Cols[0].Hidden {
		t.Error("esc must discard pending edits")
	}
}

func TestColumnsRenameReconcilesSortAndFilters(t *testing.T) {
	m, p := columnsModel(t)
	i := colIdx(p, "status")
	p.spec.Sort = []engine.SortKey{{Col: "status"}}
	p.spec.Filters = []engine.FilterSpec{{Col: "status", Op: engine.OpEq, Value: "FAILED"}}
	p.idx = i
	m.panel = p
	next, _ := m.Update(press("r"))
	cp := next.(Model).panel.(*columnsPanel)
	cp.input.SetValue("st")
	m.panel = cp
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.panel = next.(Model).panel
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.spec.Sort[0].Col != "st" {
		t.Errorf("sort not reconciled: %+v", got.spec.Sort)
	}
	if got.spec.Filters[0].Col != "st" {
		t.Errorf("filter not reconciled: %+v", got.spec.Filters)
	}
}

func TestColumnsHideDropsSortAndFilter(t *testing.T) {
	m, p := columnsModel(t)
	i := colIdx(p, "status")
	p.spec.Sort = []engine.SortKey{{Col: "status"}, {Col: "region"}}
	p.spec.Filters = []engine.FilterSpec{{Col: "status", Op: engine.OpEq, Value: "FAILED"}}
	p.idx = i
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	m.panel = next.(Model).panel
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if len(got.spec.Sort) != 1 || got.spec.Sort[0].Col != "region" {
		t.Errorf("hidden column's sort key not dropped: %+v", got.spec.Sort)
	}
	if len(got.spec.Filters) != 0 {
		t.Errorf("hidden column's filter not dropped: %+v", got.spec.Filters)
	}
}

func TestColumnsViewShowsHiddenAndAlias(t *testing.T) {
	m, p := columnsModel(t)
	p.spec.Cols[colIdx(p, "status")].Alias = "st"
	p.spec.Cols[colIdx(p, "amount")].Hidden = true
	m.panel = p
	v := m.View().Content
	if !strings.Contains(v, "st (status)") {
		t.Errorf("view must show alias (name): %q", v)
	}
}
