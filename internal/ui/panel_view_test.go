package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func typeInto(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = next.(Model)
	}
	return m
}

func runCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command")
	}
	next, _ := m.Update(cmd())
	return next.(Model)
}

func TestSaveViewKeyWritesFile(t *testing.T) {
	m := gridModel(t)
	m.spec.Sort = []engine.SortKey{{Col: "amount", Desc: true}}
	next, _ := m.Update(press("w"))
	m = next.(Model)
	if _, ok := m.panel.(*viewPanel); !ok {
		t.Fatalf("w opened %T, want *viewPanel", m.panel)
	}
	path := filepath.Join(t.TempDir(), "v.json")
	m = typeInto(t, m, path)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = runCmd(t, next.(Model), cmd)
	if m.panel != nil || m.notice == "" {
		t.Errorf("panel = %v, notice = %q", m.panel, m.notice)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("view not written: %v", err)
	}
}

func TestSaveViewAsksBeforeOverwrite(t *testing.T) {
	m := gridModel(t)
	path := filepath.Join(t.TempDir(), "v.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(press("w"))
	m = typeInto(t, next.(Model), path)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil || !next.(Model).panel.(*viewPanel).confirm {
		t.Fatal("existing file must ask for confirmation")
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Error("file overwritten before confirmation")
	}
	next, cmd = next.(Model).Update(press("y"))
	runCmd(t, next.(Model), cmd)
	if b, _ := os.ReadFile(path); string(b) == "old" {
		t.Error("file not overwritten after confirmation")
	}
}

func TestOpenViewKeyAppliesSpecAndUndoes(t *testing.T) {
	m := gridModel(t)
	q := m.spec.Clone()
	q.Sort = []engine.SortKey{{Col: "amount", Desc: true}}
	path := filepath.Join(t.TempDir(), "v.json")
	if err := engine.SaveView(path, q); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(press("o"))
	m = typeInto(t, next.(Model), path)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = runCmd(t, next.(Model), cmd)
	if m.panel != nil || len(m.spec.Sort) != 1 || len(m.hist) != 1 {
		t.Fatalf("panel = %v, sort = %v, hist = %d", m.panel, m.spec.Sort, len(m.hist))
	}
	next, _ = m.Update(press("u"))
	if got := next.(Model); len(got.spec.Sort) != 0 {
		t.Error("undo must restore the previous view")
	}
}

func TestOpenViewErrorKeepsPanel(t *testing.T) {
	m := gridModel(t)
	next, _ := m.Update(press("o"))
	m = typeInto(t, next.(Model), filepath.Join(t.TempDir(), "missing.json"))
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = runCmd(t, next.(Model), cmd)
	if m.errText == "" || m.panel == nil || len(m.hist) != 0 {
		t.Errorf("errText = %q, panel = %v, hist = %d", m.errText, m.panel, len(m.hist))
	}
}

func TestStaleViewMsgIsIgnored(t *testing.T) {
	m := gridModel(t)
	m.viewReq = 3
	q := m.spec.Clone()
	q.Filter = "x"
	next, _ := m.Update(viewLoadedMsg{id: 2, spec: q})
	if got := next.(Model); got.spec.Filter != "" {
		t.Error("stale viewLoadedMsg applied")
	}
}

func TestWithSpec(t *testing.T) {
	m := gridModel(t)
	q := m.spec.Clone()
	q.Cols[0].Hidden = true
	if got := m.WithSpec(q); len(got.headers) != len(m.headers)-1 {
		t.Errorf("headers = %v", got.headers)
	}
}
