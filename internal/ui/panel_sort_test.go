package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func sortModel(t *testing.T) (Model, *sortPanel) {
	t.Helper()
	m := gridModel(t)
	p := newSortPanel(m.headers, 0, m.spec.Clone())
	m.panel = p
	return m, p
}

func TestSortPanelAscending(t *testing.T) {
	m, p := sortModel(t)
	p.idx = 2
	m.panel = p
	next, cmd := m.Update(press("a"))
	got := next.(Model)
	if got.panel != nil {
		t.Error("applying a sort must close the panel")
	}
	if len(got.spec.Sort) != 1 || got.spec.Sort[0].Col != "amount" || got.spec.Sort[0].Desc {
		t.Fatalf("sort = %+v, want amount ASC", got.spec.Sort)
	}
	if cmd == nil {
		t.Error("applying a sort must issue a reload")
	}
	if len(got.hist) != 1 {
		t.Error("applying a sort must push undo history")
	}
}

func TestSortPanelDescending(t *testing.T) {
	m, _ := sortModel(t)
	next, _ := m.Update(press("d"))
	if got := next.(Model); !got.spec.Sort[0].Desc || got.spec.Sort[0].Col != "region" {
		t.Errorf("sort = %+v, want region DESC", got.spec.Sort)
	}
}

func TestSortPanelClear(t *testing.T) {
	m, _ := sortModel(t)
	m.spec.Sort = []engine.SortKey{{Col: "region"}}
	m.panel = newSortPanel(m.headers, 0, m.spec.Clone())
	next, _ := m.Update(press("x"))
	if got := next.(Model); len(got.spec.Sort) != 0 {
		t.Errorf("sort not cleared: %+v", got.spec.Sort)
	}
}

func TestSortPanelCancel(t *testing.T) {
	m, _ := sortModel(t)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got := next.(Model)
	if got.panel != nil {
		t.Error("esc must close the panel")
	}
	if len(got.spec.Sort) != 0 {
		t.Error("esc must not change the spec")
	}
	if cmd != nil {
		t.Error("esc must not issue a query")
	}
}

func TestSortPanelSelectionMoves(t *testing.T) {
	m, _ := sortModel(t)
	next, _ := m.Update(press("j"))
	if got := next.(Model); got.panel.(*sortPanel).idx != 1 {
		t.Errorf("idx = %d, want 1", got.panel.(*sortPanel).idx)
	}
	m.panel = &sortPanel{headers: m.headers, spec: m.spec.Clone(), idx: 1}
	next, _ = m.Update(press("k"))
	if got := next.(Model); got.panel.(*sortPanel).idx != 0 {
		t.Errorf("idx = %d, want 0", got.panel.(*sortPanel).idx)
	}
}

func TestSortPanelEnterIsAscending(t *testing.T) {
	m, _ := sortModel(t)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := next.(Model); got.spec.Sort[0].Desc {
		t.Error("enter sorts ascending")
	}
}
