package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func typedFixture(t *testing.T) *filterTypedPanel {
	t.Helper()
	q := engine.NewSpec([]engine.ColumnMeta{
		{Name: "status", Type: "VARCHAR"},
		{Name: "amount", Type: "DOUBLE"},
	})
	return newFilterTypedPanel(q, "status", "VARCHAR")
}

func updateTyped(p Panel, msg tea.Msg) (*filterTypedPanel, PanelAction) {
	next, a := p.Update(msg)
	return next.(*filterTypedPanel), a
}

func TestTypedPanelShowsOpsByType(t *testing.T) {
	p := typedFixture(t)
	if len(p.existing) != 0 || len(p.ops) != 7 {
		t.Fatalf("existing %v, ops %d", p.existing, len(p.ops))
	}
	num := newFilterTypedPanel(typedFixture(t).spec, "amount", "DOUBLE")
	if len(num.ops) != 8 || num.ops[2] != engine.OpGt {
		t.Fatalf("numeric ops = %v", num.ops)
	}
}

func TestTypedPanelNewFilterFlow(t *testing.T) {
	p, a := updateTyped(typedFixture(t), tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.Spec != nil || p.stage != 1 {
		t.Fatalf("after enter on first op: stage %d, spec %v", p.stage, a.Spec)
	}
	p.value.SetValue("FAILED")
	p, a = updateTyped(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !a.Close || !a.Push || a.Spec == nil {
		t.Fatalf("action = %+v", a)
	}
	if len(a.Spec.Filters) != 1 || a.Spec.Filters[0].String() != "status = FAILED" || a.Spec.Filters[0].ColType != "VARCHAR" {
		t.Fatalf("filters = %v", a.Spec.Filters)
	}
}

func TestTypedPanelNullOpAppliesImmediately(t *testing.T) {
	p := typedFixture(t)
	for i := 0; i < 5; i++ {
		p, _ = updateTyped(p, press("j"))
	}
	_, a := updateTyped(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !a.Close || a.Spec == nil {
		t.Fatalf("action = %+v", a)
	}
	if got := a.Spec.Filters[0]; got.Op != engine.OpNull || got.Col != "status" || got.Value != "" {
		t.Fatalf("filter = %+v", got)
	}
}

func TestTypedPanelEmptyValueRefused(t *testing.T) {
	p, _ := updateTyped(typedFixture(t), tea.KeyPressMsg{Code: tea.KeyEnter})
	p, a := updateTyped(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.Spec != nil || a.Close || p.stage != 1 || p.errText != "value required" {
		t.Fatalf("action = %+v, stage %d, err %q", a, p.stage, p.errText)
	}
}

func TestTypedPanelEditReplaces(t *testing.T) {
	q := typedFixture(t).spec
	q.Filters = []engine.FilterSpec{{Col: "status", Op: engine.OpEq, Value: "PENDING", ColType: "VARCHAR"}}
	p, _ := updateTyped(newFilterTypedPanel(q, "status", "VARCHAR"), tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.stage != 1 || p.value.Value() != "PENDING" {
		t.Fatalf("stage %d, value %q", p.stage, p.value.Value())
	}
	p.value.SetValue("FAILED")
	_, a := updateTyped(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(a.Spec.Filters) != 1 || a.Spec.Filters[0].Value != "FAILED" {
		t.Fatalf("filters = %v", a.Spec.Filters)
	}
}

func TestTypedPanelEscFlow(t *testing.T) {
	p, _ := updateTyped(typedFixture(t), tea.KeyPressMsg{Code: tea.KeyEnter})
	p, a := updateTyped(p, tea.KeyPressMsg{Code: tea.KeyEscape})
	if p.stage != 0 || a.Close {
		t.Fatalf("esc from value: stage %d, action %+v", p.stage, a)
	}
	_, a = updateTyped(p, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !a.Close || a.Spec != nil {
		t.Fatalf("esc from list: %+v", a)
	}
}
