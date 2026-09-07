package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func gridModel(t *testing.T) Model {
	t.Helper()
	m := New(testSession(t))
	m.headers = displayNames(m.spec)
	return m
}

func TestSortAscending(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeSort
	m.sortIdx = 2
	next, cmd := m.updateSort(tea.KeyPressMsg{Code: 'a', Text: "a"})
	got := next.(Model)
	if got.mode != ModeGrid {
		t.Error("sort mode should close after applying")
	}
	if len(got.spec.Sort) != 1 {
		t.Fatalf("got %d sort keys, want 1", len(got.spec.Sort))
	}
	if got.spec.Sort[0].Col != "amount" || got.spec.Sort[0].Desc {
		t.Errorf("got %+v, want amount ASC", got.spec.Sort[0])
	}
	if cmd == nil {
		t.Error("applying a sort must issue a reload")
	}
}

func TestSortDescending(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeSort
	m.sortIdx = 0
	next, _ := m.updateSort(tea.KeyPressMsg{Code: 'd', Text: "d"})
	got := next.(Model)
	if !got.spec.Sort[0].Desc || got.spec.Sort[0].Col != "region" {
		t.Errorf("got %+v, want region DESC", got.spec.Sort[0])
	}
}

func TestSortClear(t *testing.T) {
	m := gridModel(t)
	m.spec.Sort = []engine.SortKey{{Col: "region"}}
	m.mode = ModeSort
	next, _ := m.updateSort(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if got := next.(Model); len(got.spec.Sort) != 0 {
		t.Errorf("sort not cleared: %+v", got.spec.Sort)
	}
}

func TestSortCancel(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeSort
	next, cmd := m.updateSort(tea.KeyPressMsg{Code: tea.KeyEscape})
	got := next.(Model)
	if got.mode != ModeGrid {
		t.Error("esc should return to grid")
	}
	if len(got.spec.Sort) != 0 {
		t.Error("esc must not change the spec")
	}
	if cmd != nil {
		t.Error("esc must not issue a query")
	}
}

func TestSortSelectionMoves(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeSort
	m.sortIdx = 0
	next, _ := m.updateSort(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if got := next.(Model); got.sortIdx != 1 {
		t.Errorf("sortIdx = %d, want 1", got.sortIdx)
	}
	m.sortIdx = 0
	next, _ = m.updateSort(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if got := next.(Model); got.sortIdx != 0 {
		t.Errorf("sortIdx = %d, want 0 (clamped)", got.sortIdx)
	}
}

func TestSortPushesUndo(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeSort
	m.sortIdx = 0
	next, _ := m.updateSort(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if got := next.(Model); len(got.hist) != 1 {
		t.Errorf("undo stack has %d entries, want 1", len(got.hist))
	}
}
