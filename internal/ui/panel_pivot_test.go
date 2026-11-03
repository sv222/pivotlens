package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func TestPivotPanelBuildsSpec(t *testing.T) {
	m := gridModel(t)
	m.spec.Sort = []engine.SortKey{{Col: "amount"}}
	next, _ := m.Update(press("p"))
	m = next.(Model)
	if _, ok := m.panel.(*pivotPanel); !ok {
		t.Fatalf("panel = %T, want *pivotPanel", m.panel)
	}
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	for _, k := range []tea.Msg{enter, press("j"), enter, press("j"), enter, press("j"), press("j"), enter, enter} {
		next, _ = m.Update(k)
		m = next.(Model)
	}
	if m.panel != nil {
		t.Fatal("the last pick must close the panel")
	}
	p := newPivotPanel(m.headers, m.sess.NumericFields(m.spec), m.spec.Clone(), func(q engine.QuerySpec) tea.Cmd {
		return func() tea.Msg { return q }
	})
	var a PanelAction
	for _, k := range []tea.Msg{enter, press("j"), enter, press("j"), enter, press("j"), press("j"), enter, enter} {
		_, a = p.Update(k)
	}
	q := a.Cmd().(engine.QuerySpec)
	want := engine.PivotSpec{Rows: []string{"region"}, On: "status", Measures: []engine.Measure{{Agg: "sum", Col: "amount"}}}
	if q.Pivot == nil || !reflect.DeepEqual(*q.Pivot, want) {
		t.Fatalf("pivot = %+v, want %+v", q.Pivot, want)
	}
	if q.Sort != nil {
		t.Error("source sort must not carry into the pivot")
	}
}

func TestPivotPanelSeveralValues(t *testing.T) {
	m := gridModel(t)
	p := newPivotPanel(m.headers, m.sess.NumericFields(m.spec), m.spec.Clone(), func(q engine.QuerySpec) tea.Cmd {
		return func() tea.Msg { return q }
	})
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	esc := tea.KeyPressMsg{Code: tea.KeyEscape}
	var a PanelAction
	for _, k := range []tea.Msg{enter, enter, press("j"), enter, enter, press("j"), enter, enter, enter} {
		_, a = p.Update(k)
	}
	if p.stage != 4 || !strings.Contains(p.View(m.styles, 80), "[sum(amount), count]") {
		t.Fatalf("stage %d view %q", p.stage, p.View(m.styles, 80))
	}
	j := press("j")
	for _, k := range []tea.Msg{esc, esc, esc, j, enter, j, j, j, j, enter, j, j, enter, enter} {
		_, a = p.Update(k)
	}
	q := a.Cmd().(engine.QuerySpec)
	want := []engine.Measure{{Agg: "sum", Col: "amount"}, {Agg: "max", Col: "amount"}}
	if q.Pivot == nil || q.Pivot.On != "" || !reflect.DeepEqual(q.Pivot.Measures, want) {
		t.Fatalf("pivot = %+v, want measures %v", q.Pivot, want)
	}
}

func TestPivotPanelEscGoesBack(t *testing.T) {
	m := gridModel(t)
	p := newPivotPanel(m.headers, m.sess.NumericFields(m.spec), m.spec.Clone(), func(engine.QuerySpec) tea.Cmd { return nil })
	p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.stage != 1 || !strings.Contains(p.View(m.styles, 80), "pivot columns") {
		t.Fatalf("stage = %d", p.stage)
	}
	if _, a := p.Update(tea.KeyPressMsg{Code: tea.KeyEscape}); a.Close || p.stage != 0 {
		t.Fatal("esc must step back a stage")
	}
	if _, a := p.Update(tea.KeyPressMsg{Code: tea.KeyEscape}); !a.Close {
		t.Fatal("esc on the first stage must close")
	}
}

func TestPivotMsgOpensPivotView(t *testing.T) {
	m := gridModel(t)
	m.pivotReq = 2
	q := m.spec.Clone()
	q.Pivot = &engine.PivotSpec{Rows: []string{"region"}, On: "status", Measures: []engine.Measure{{Agg: "count"}}}
	r, err := m.sess.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(pivotMsg{id: 1, spec: r})
	if next.(Model).spec.Pivot != nil {
		t.Fatal("stale pivotMsg must be ignored")
	}
	next, cmd := m.Update(pivotMsg{id: 2, spec: r})
	got := next.(Model)
	if got.spec.Pivot == nil || cmd == nil || len(got.hist) != 1 {
		t.Fatalf("pivot not applied: %+v", got.spec.Pivot)
	}
	if want := "region COMPLETED FAILED PENDING"; strings.Join(got.headers, " ") != want {
		t.Errorf("headers = %v, want %s", got.headers, want)
	}
	if !strings.Contains(got.renderStatus(), "pivot: region × status count") {
		t.Errorf("status = %q", got.renderStatus())
	}
	for _, k := range []string{"p", "/", "c"} {
		n, _ := got.Update(press(k))
		if g := n.(Model); g.panel != nil || g.errText == "" {
			t.Errorf("key %q must be refused on a pivot view", k)
		}
	}
	back, _ := got.Update(press("u"))
	if back.(Model).spec.Pivot != nil {
		t.Error("undo must leave the pivot view")
	}
}

func TestPivotMsgErrorShown(t *testing.T) {
	m := gridModel(t)
	m.pivotReq = 1
	next, _ := m.Update(pivotMsg{id: 1, err: errors.New("too many pivot columns")})
	got := next.(Model)
	if got.spec.Pivot != nil || !strings.Contains(got.renderStatus(), "too many pivot columns") {
		t.Errorf("status = %q", got.renderStatus())
	}
}

func TestPivotPanelSumAvgOfferNumericFieldsOnly(t *testing.T) {
	m := gridModel(t)
	p := newPivotPanel(m.headers, m.sess.NumericFields(m.spec), m.spec.Clone(), func(engine.QuerySpec) tea.Cmd { return nil })
	for _, agg := range []string{"sum", "avg"} {
		p.stage, p.picks[2] = 3, agg
		if got := p.options(); !reflect.DeepEqual(got, []string{"amount"}) {
			t.Errorf("%s options = %v, want [amount]", agg, got)
		}
	}
	for _, agg := range []string{"min", "max"} {
		p.picks[2] = agg
		if got := p.options(); !reflect.DeepEqual(got, m.headers) {
			t.Errorf("%s options = %v, want all headers", agg, got)
		}
	}
	p.picks[2] = "count"
	if got := p.options(); len(got) != len(m.headers)+1 || got[0] != noField {
		t.Errorf("count options = %v", got)
	}
}

func pivotModel(t *testing.T) Model {
	t.Helper()
	m := gridModel(t)
	q := m.spec.Clone()
	q.Pivot = &engine.PivotSpec{Rows: []string{"region"}, On: "status", Measures: []engine.Measure{{Agg: "count"}}}
	r, err := m.sess.ResolvePivot(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	m.pivotReq = 1
	next, _ := m.Update(pivotMsg{id: 1, spec: r})
	return next.(Model)
}

func runPivotKey(t *testing.T, m Model, key string) Model {
	t.Helper()
	next, cmd := m.Update(press(key))
	if cmd == nil {
		t.Fatalf("key %q gave no command: %q", key, next.(Model).errText)
	}
	next, _ = next.Update(cmd())
	return next.(Model)
}

func TestTotalsKeyTogglesTotals(t *testing.T) {
	if next, _ := gridModel(t).Update(press("t")); next.(Model).errText == "" {
		t.Error("t outside a pivot view must be refused")
	}
	m := runPivotKey(t, pivotModel(t), "t")
	if want := "region COMPLETED FAILED PENDING Total"; strings.Join(m.headers, " ") != want {
		t.Errorf("headers = %v, want %s", m.headers, want)
	}
	if !strings.Contains(m.renderStatus(), "with totals") {
		t.Errorf("status = %q", m.renderStatus())
	}
	m.spec.Sort = []engine.SortKey{{Col: "Total", Desc: true}, {Col: "region"}}
	if m = runPivotKey(t, m, "t"); m.spec.Pivot.Totals || !reflect.DeepEqual(m.spec.Sort, []engine.SortKey{{Col: "region"}}) {
		t.Errorf("second t must turn totals off and drop the Total sort, got %+v", m.spec)
	}
}

func TestPercentKeyCyclesModes(t *testing.T) {
	m := pivotModel(t)
	for _, want := range []string{"row", "col", "total", ""} {
		m = runPivotKey(t, m, "%")
		if m.spec.Pivot.Percent != want {
			t.Fatalf("percent = %q, want %q", m.spec.Pivot.Percent, want)
		}
	}
	m = runPivotKey(t, m, "%")
	if !strings.Contains(m.renderStatus(), "as % of row") {
		t.Errorf("status = %q", m.renderStatus())
	}
}
