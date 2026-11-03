package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func exportModel(t *testing.T) (Model, *exportPanel) {
	t.Helper()
	m := gridModel(t)
	p := newExportPanel(m.spec.Clone(), func(q engine.QuerySpec, path string) tea.Cmd {
		return nil
	})
	m.panel = p
	return m, p
}

func exportPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "out.csv")
}

func TestExportExtensionRefused(t *testing.T) {
	m, p := exportModel(t)
	p.input.SetValue(filepath.Join(t.TempDir(), "out.xlsx"))
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.panel == nil {
		t.Fatal("unsupported extension must keep the panel open")
	}
	if got.panel.(*exportPanel).errText == "" {
		t.Error("refusal must set an error message")
	}
}

func TestExportNewPathRuns(t *testing.T) {
	m, p := exportModel(t)
	p.input.SetValue(exportPath(t))
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := next.(Model); got.panel == nil {
		t.Error("fresh-path export must keep the panel open while the write is in flight")
	}
}

func TestExportExistingPathAsks(t *testing.T) {
	path := exportPath(t)
	if err := writeFile(path, "x"); err != nil {
		t.Fatal(err)
	}
	m, p := exportModel(t)
	p.input.SetValue(path)
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	cp := next.(Model).panel
	if cp == nil {
		t.Fatal("existing path must ask for confirmation")
	}
	if !cp.(*exportPanel).confirm {
		t.Error("panel must enter confirm state")
	}
	m.panel = cp
	next, _ = m.Update(press("n"))
	cp = next.(Model).panel
	if cp.(*exportPanel).confirm {
		t.Error("n must return to the input")
	}
}

func TestExportConfirmNoKeepsPanel(t *testing.T) {
	path := exportPath(t)
	if err := writeFile(path, "x"); err != nil {
		t.Fatal(err)
	}
	m, p := exportModel(t)
	p.input.SetValue(path)
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.panel = next.(Model).panel
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := next.(Model); got.panel == nil {
		t.Error("esc in confirm state must return to the input, not close")
	}
}

func TestExportConfirmYesRuns(t *testing.T) {
	path := exportPath(t)
	if err := writeFile(path, "x"); err != nil {
		t.Fatal(err)
	}
	var ranTo string
	ran := false
	m := gridModel(t)
	p := newExportPanel(m.spec.Clone(), func(q engine.QuerySpec, path string) tea.Cmd {
		ran = true
		ranTo = path
		return func() tea.Msg { return exportMsg{id: 1, path: path} }
	})
	p.input.SetValue(path)
	m.panel = p
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.panel = next.(Model).panel
	_, cmd := m.Update(press("y"))
	if !ran {
		t.Error("y must invoke the export run func")
	}
	if ranTo != path {
		t.Errorf("run path = %q, want %q", ranTo, path)
	}
	if cmd == nil {
		t.Error("y must return the run cmd")
	}
}

func TestExportMsgSuccessSetsNotice(t *testing.T) {
	m := gridModel(t)
	m.exportReq = 3
	next, _ := m.Update(exportMsg{id: 3, path: "out.csv"})
	got := next.(Model)
	if got.notice != "wrote out.csv" {
		t.Errorf("notice = %q", got.notice)
	}
	if got.panel != nil {
		t.Error("success must close any open export panel")
	}
}

func TestExportMsgFailureKeepsPanel(t *testing.T) {
	m := gridModel(t)
	m.exportReq = 3
	p := newExportPanel(m.spec.Clone(), func(engine.QuerySpec, string) tea.Cmd { return nil })
	p.input.SetValue("out.csv")
	m.panel = p
	next, _ := m.Update(exportMsg{id: 3, err: errFake})
	got := next.(Model)
	if got.panel == nil {
		t.Error("failure must keep the panel open")
	}
	if got.errText == "" {
		t.Error("failure must show in the error bar")
	}
}

var errFake = fakeErr{}

type fakeErr struct{}

func (fakeErr) Error() string { return "fake export failure" }

func writeFile(path, content string) error {
	return osWriteFile(path, []byte(content))
}

func TestExportFormat(t *testing.T) {
	for path, want := range map[string]string{"a.csv": "csv", "a.PARQUET": "parquet", "a.ndjson": "ndjson", "a.json": ""} {
		if got, ok := exportFormat(path); got != want || ok != (want != "") {
			t.Errorf("exportFormat(%q) = %q %v, want %q", path, got, ok, want)
		}
	}
}
