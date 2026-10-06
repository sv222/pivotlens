package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func gridModel(t *testing.T) Model {
	t.Helper()
	m := New(testSession(t))
	m.headers = displayNames(m.spec)
	return m
}

func press(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

func TestCloseClearsPanel(t *testing.T) {
	m := gridModel(t)
	m.panel = newSortPanel(m.headers, 0, m.spec.Clone())
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := next.(Model); got.panel != nil {
		t.Error("esc must clear the panel")
	}
}

func TestPushReachesHistoryOnce(t *testing.T) {
	m := gridModel(t)
	m.panel = newSortPanel(m.headers, 0, m.spec.Clone())
	next, _ := m.Update(press("a"))
	got := next.(Model)
	if len(got.hist) != 1 {
		t.Fatalf("history has %d entries, want 1", len(got.hist))
	}
	if len(got.spec.Sort) != 1 || got.spec.Sort[0].Col != "region" {
		t.Fatalf("sort = %+v", got.spec.Sort)
	}
}

func TestDebounceDoesNotReloadImmediately(t *testing.T) {
	m := gridModel(t)
	m.panel = newFilterPanel(m.spec.Clone())
	next, cmd := m.Update(press("E"))
	got := next.(Model)
	if got.pending == nil {
		t.Fatal("keystroke must stage a pending spec")
	}
	if got.spec.Filter != "" {
		t.Error("spec must not change before the debounce fires")
	}
	if cmd == nil {
		t.Error("keystroke must schedule the debounce")
	}
}

func TestFreshDebounceAdoptsPending(t *testing.T) {
	m := gridModel(t)
	m.panel = newFilterPanel(m.spec.Clone())
	next, _ := m.Update(press("E"))
	got := next.(Model)
	next, cmd := got.Update(debounceMsg{id: got.keySeq})
	got = next.(Model)
	if got.spec.Filter != "E" {
		t.Errorf("Filter = %q, want E", got.spec.Filter)
	}
	if got.pending != nil {
		t.Error("pending must clear after adoption")
	}
	if cmd == nil {
		t.Error("adoption must reload")
	}
}

func TestUnchangedDebounceIsNoop(t *testing.T) {
	m := gridModel(t)
	m.spec.Filter = "EU"
	m.panel = newFilterPanel(m.spec.Clone())
	p := m.panel.(*filterPanel)
	p.input.SetValue("EU")
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	got := next.(Model)
	if got.pending != nil {
		t.Fatal("identical filter must not stage a pending spec")
	}
	if len(got.hist) != 0 {
		t.Error("identical filter must not push history")
	}
}
