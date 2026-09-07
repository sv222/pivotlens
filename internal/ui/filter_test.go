package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFilterTypingSchedulesDebounce(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeFilter
	m.filter.Focus()
	before := m.keySeq
	next, cmd := m.updateFilter(tea.KeyPressMsg{Code: 'E', Text: "E"})
	got := next.(Model)
	if got.keySeq == before {
		t.Error("typing must bump keySeq")
	}
	if cmd == nil {
		t.Error("typing must schedule a debounce")
	}
	if got.filter.Value() != "E" {
		t.Errorf("input value = %q, want E", got.filter.Value())
	}
	if got.spec.Filter != "" {
		t.Error("spec must not change before the debounce fires")
	}
}

func TestStaleDebounceIsIgnored(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeFilter
	m.keySeq = 9
	m.filter.SetValue("EU")
	next, cmd := m.Update(debounceMsg{id: 4})
	if got := next.(Model); got.spec.Filter != "" {
		t.Error("stale debounce applied the filter")
	}
	if cmd != nil {
		t.Error("stale debounce issued a query")
	}
}

func TestFreshDebounceAppliesFilter(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeFilter
	m.keySeq = 9
	m.filter.SetValue("EU")
	next, cmd := m.Update(debounceMsg{id: 9})
	got := next.(Model)
	if got.spec.Filter != "EU" {
		t.Errorf("spec.Filter = %q, want EU", got.spec.Filter)
	}
	if cmd == nil {
		t.Error("applying a filter must issue a reload")
	}
}

func TestApplyFilterIsIdempotent(t *testing.T) {
	m := gridModel(t)
	m.spec.Filter = "EU"
	m.filter.SetValue("EU")
	m.keySeq = 2
	next, cmd := m.Update(debounceMsg{id: 2})
	if cmd != nil {
		t.Error("unchanged filter must not re-query")
	}
	if got := next.(Model); len(got.hist) != 0 {
		t.Error("unchanged filter must not push undo history")
	}
}

func TestFilterEnterClosesMode(t *testing.T) {
	m := gridModel(t)
	m.mode = ModeFilter
	m.filter.SetValue("EU")
	next, _ := m.updateFilter(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.mode != ModeGrid {
		t.Error("enter should return to grid")
	}
	if got.spec.Filter != "EU" {
		t.Errorf("enter should apply the filter, got %q", got.spec.Filter)
	}
}

func TestFilterEscapeReverts(t *testing.T) {
	m := gridModel(t)
	m.spec.Filter = "EU"
	m.mode = ModeFilter
	m.filter.SetValue("typing-in-progress")
	next, cmd := m.updateFilter(tea.KeyPressMsg{Code: tea.KeyEscape})
	got := next.(Model)
	if got.mode != ModeGrid {
		t.Error("esc should return to grid")
	}
	if got.spec.Filter != "EU" {
		t.Errorf("esc must not change the applied filter, got %q", got.spec.Filter)
	}
	if cmd != nil {
		t.Error("esc must not issue a query")
	}
}
