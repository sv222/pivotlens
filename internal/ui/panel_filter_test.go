package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func filterModel(t *testing.T) (Model, *filterPanel) {
	t.Helper()
	m := gridModel(t)
	p := newFilterPanel(m.spec.Clone())
	p.input.Focus()
	m.panel = p
	return m, p
}

func TestFilterPanelTypingStagesDebounce(t *testing.T) {
	m, _ := filterModel(t)
	before := m.keySeq
	next, cmd := m.Update(press("E"))
	got := next.(Model)
	if got.keySeq == before {
		t.Error("typing must bump keySeq")
	}
	if cmd == nil {
		t.Error("typing must schedule a debounce")
	}
	if got.pending == nil || got.pending.Filter != "E" {
		t.Fatal("typing must stage the new filter")
	}
	if got.spec.Filter != "" {
		t.Error("spec must not change before the debounce fires")
	}
}

func TestFilterPanelStaleDebounceIgnored(t *testing.T) {
	m, _ := filterModel(t)
	next, _ := m.Update(press("E"))
	got := next.(Model)
	next, cmd := got.Update(debounceMsg{id: got.keySeq - 1})
	if g := next.(Model); g.spec.Filter != "" {
		t.Error("stale debounce applied the filter")
	}
	_ = cmd
}

func TestFilterPanelEnterAppliesAndCloses(t *testing.T) {
	m, p := filterModel(t)
	p.input.SetValue("EU")
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.panel != nil {
		t.Error("enter must close the panel")
	}
	if got.spec.Filter != "EU" {
		t.Errorf("Filter = %q, want EU", got.spec.Filter)
	}
	if len(got.hist) != 1 {
		t.Error("one visit must push exactly one undo entry")
	}
}

func TestFilterPanelEscapeKeepsFilter(t *testing.T) {
	m, _ := filterModel(t)
	m.spec.Filter = "EU"
	m.panel = newFilterPanel(m.spec.Clone())
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got := next.(Model)
	if got.panel != nil {
		t.Error("esc must close the panel")
	}
	if got.spec.Filter != "EU" {
		t.Errorf("esc must not change the applied filter, got %q", got.spec.Filter)
	}
	if cmd != nil {
		t.Error("esc must not issue a query")
	}
}

func TestFilterPanelOneUndoPerVisit(t *testing.T) {
	m, _ := filterModel(t)
	next, _ := m.Update(press("E"))
	got := next.(Model)
	next, _ = got.Update(press("U"))
	got = next.(Model)
	if len(got.hist) != 1 {
		t.Errorf("two keystrokes pushed %d undo entries, want 1", len(got.hist))
	}
	next, cmd := got.Update(debounceMsg{id: got.keySeq})
	if got := next.(Model); len(got.hist) != 1 {
		t.Errorf("adoption pushed again: %d entries", len(got.hist))
	}
	_ = cmd
}

func TestEscapeDiscardsPendingDebounce(t *testing.T) {
	m, _ := filterModel(t)
	next, _ := m.Update(press("E"))
	got := next.(Model)
	next, cmd := got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = next.(Model)
	if cmd != nil {
		t.Error("esc must not issue a query")
	}
	if got.pending != nil {
		t.Error("esc must discard the staged pending spec")
	}
	next, cmd = got.Update(debounceMsg{id: got.keySeq})
	if cmd != nil {
		t.Error("late debounce after esc must not query")
	}
	if got := next.(Model); got.spec.Filter != "" {
		t.Errorf("late debounce after esc applied the filter, got %q", got.spec.Filter)
	}
}
