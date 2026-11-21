package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

// drainBatch runs a command and everything it produces until the model has
// seen every message.
func drainBatch(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msgs := []tea.Msg{cmd()}
	for len(msgs) > 0 {
		msg := msgs[0]
		msgs = msgs[1:]
		switch msg := msg.(type) {
		case tea.BatchMsg:
			for _, c := range msg {
				msgs = append(msgs, c())
			}
		default:
			next, _ := m.Update(msg)
			m = next.(Model)
		}
	}
	return m
}

func freqModel(t *testing.T) Model {
	t.Helper()
	next, cmd := gridModel(t).Update(press("v"))
	m := next.(Model)
	if cmd == nil {
		t.Fatal("v must reload the grid")
	}
	return drainBatch(t, m, cmd)
}

func TestVKeyOpensFreqView(t *testing.T) {
	m := freqModel(t)
	if m.spec.Freq == nil || m.spec.Freq.Col != "region" || m.spec.Freq.Top != engine.DefaultTop {
		t.Fatalf("freq = %+v, want region with the default top", m.spec.Freq)
	}
	want := []string{"value", "count", "percent", "cum_percent", barCol}
	if !reflect.DeepEqual(m.headers, want) {
		t.Fatalf("headers = %v, want %v", m.headers, want)
	}
	if len(m.rows) == 0 || m.rows[0][0] != "EU-CENT" || m.rows[0][1] != "2" || m.rows[0][2] != "40" {
		t.Fatalf("rows = %v", m.rows)
	}
	if m.rows[0][4] != strings.Repeat("\u2588", 8) {
		t.Errorf("bar = %d blocks, want 8", len(m.rows[0][4]))
	}
	if got := m.widths[len(m.widths)-1]; got != FreqBarWidth {
		t.Errorf("bar width = %d, want %d", got, FreqBarWidth)
	}
	if len(m.hist) != 1 {
		t.Errorf("history has %d entries, want 1", len(m.hist))
	}
	if m.spec.Sort != nil {
		t.Error("source sort must not carry into the value counts view")
	}
	back, _ := m.Update(press("u"))
	if back.(Model).spec.Freq != nil {
		t.Error("u must leave the value counts view")
	}
}

func TestVKeyRefused(t *testing.T) {
	m := gridModel(t)
	m.spec.Pivot = &engine.PivotSpec{Rows: []string{"region"}, Measures: []engine.Measure{{Agg: "count"}}}
	next, _ := m.Update(press("v"))
	if got := next.(Model); got.spec.Freq != nil || got.errText == "" {
		t.Error("v on a pivot view must be refused")
	}
	m = freqModel(t)
	next, _ = m.Update(press("v"))
	if got := next.(Model); got.spec.Freq.Col != "region" || got.errText == "" {
		t.Errorf("second v must be refused, err %q", got.errText)
	}
	m = gridModel(t)
	m.colCursor = len(m.headers) + 1
	next, _ = m.Update(press("v"))
	if got := next.(Model); got.spec.Freq != nil || got.errText == "" {
		t.Error("v with an out-of-range column must be refused")
	}
}

func TestFreqKeysRefusedOnFreqView(t *testing.T) {
	m := freqModel(t)
	for _, k := range []string{"/", "f", "F", "c", "i", "p"} {
		next, _ := m.Update(press(k))
		if got := next.(Model); got.panel != nil || got.errText == "" {
			t.Errorf("key %q must be refused on a value counts view", k)
		}
	}
}

func TestFreqStatusBar(t *testing.T) {
	m := freqModel(t)
	m.w = 200
	s := m.renderStatus()
	for _, want := range []string{"value counts: region", "[enter] filter", "[space] mark", "[e] export", "[u] back"} {
		if !strings.Contains(s, want) {
			t.Errorf("status %q missing %q", s, want)
		}
	}
	m.freqReq = 1
	next, _ := m.Update(freqStatsMsg{id: 1, total: 5000, distinct: 1500})
	m = next.(Model)
	if s := m.renderStatus(); !strings.Contains(s, "top 1000 of 1500 distinct") {
		t.Errorf("status = %q, want the truncation notice", s)
	}
	m.freqReq = 2
	next, _ = m.Update(freqStatsMsg{id: 2, total: 5000, distinct: 800})
	m = next.(Model)
	if s := m.renderStatus(); strings.Contains(s, "distinct") {
		t.Errorf("status = %q, must not report truncation", s)
	}
	m.freqReq = 3
	next, _ = m.Update(freqStatsMsg{id: 3, err: errors.New("gone")})
	m = next.(Model)
	if m.freqTotal != 5000 || m.freqDistinct != 800 {
		t.Errorf("a failed freqStatsMsg must keep the stats, got %d/%d", m.freqTotal, m.freqDistinct)
	}
}

func TestFreqBar(t *testing.T) {
	full := strings.Repeat("\u2588", FreqBarWidth)
	for _, c := range []struct {
		pct  string
		want string
	}{
		{"0", ""},
		{"NULL", ""},
		{"x", ""},
		{"100", full},
		{"40", strings.Repeat("\u2588", 8)},
		{"12.5", strings.Repeat("\u2588", 3)},
	} {
		if got := freqBar(c.pct); got != c.want {
			t.Errorf("freqBar(%q) = %d blocks, want %d", c.pct, len(got), len(c.want))
		}
	}
}

func TestFreqPageMsgKeepsBarAcrossPages(t *testing.T) {
	m := freqModel(t)
	rows := [][]string{{"a", "2", "40", "40"}, {"b", "3", "60", "100"}}
	next, _ := m.Update(pageMsg{id: m.pageReq, rows: rows, cols: []string{"value", "count", "percent", "cum_percent"}})
	got := next.(Model)
	if !reflect.DeepEqual(got.headers, []string{"value", "count", "percent", "cum_percent", barCol}) {
		t.Errorf("headers = %v", got.headers)
	}
	if got.rows[0][4] != strings.Repeat("\u2588", 8) || got.rows[1][4] != strings.Repeat("\u2588", 12) {
		t.Errorf("bars = %q, %q", got.rows[0][4], got.rows[1][4])
	}
}

func modelOn(t *testing.T, name string) Model {
	t.Helper()
	s, err := engine.Open(context.Background(), filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	m := New(s)
	m.headers = displayNames(m.spec)
	return m
}

func TestFreqEnterAddsEqFilter(t *testing.T) {
	m := freqModel(t)
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("enter must reload the grid")
	}
	if m.spec.Freq != nil {
		t.Error("enter must close the value counts view")
	}
	if len(m.spec.Filters) != 1 {
		t.Fatalf("filters = %+v", m.spec.Filters)
	}
	if f := m.spec.Filters[0]; f.Col != "region" || f.Op != engine.OpEq || f.Value != "EU-CENT" || f.ColType != "VARCHAR" {
		t.Errorf("filter = %+v", f)
	}
	if len(m.hist) != 1 {
		t.Errorf("history has %d entries, want 1", len(m.hist))
	}
	m = drainBatch(t, m, cmd)
	if len(m.rows) != 2 || m.rows[0][0] != "EU-CENT" || m.rows[1][0] != "EU-CENT" {
		t.Errorf("rows after drill-down = %v, want the 2 EU-CENT rows", m.rows)
	}
	back, _ := m.Update(press("u"))
	if b := back.(Model); b.spec.Freq != nil || len(b.spec.Filters) != 0 {
		t.Errorf("u after drill-down leaves %+v, want the unfiltered parent", b.spec)
	}
}

func TestFreqEnterNullRow(t *testing.T) {
	m := modelOn(t, "nulls.csv")
	m.colCursor = 1
	next, cmd := m.Update(press("v"))
	m = drainBatch(t, next.(Model), cmd)
	if m.spec.Freq.Col != "status" {
		t.Fatalf("freq = %+v", m.spec.Freq)
	}
	if m.rows[1][0] != engine.FreqNullText {
		t.Fatalf("rows = %v, want the null row at 1", m.rows)
	}
	next, _ = m.Update(press("j"))
	m = next.(Model)
	next, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if len(m.spec.Filters) != 1 || m.spec.Filters[0].Op != engine.OpNull || m.spec.Filters[0].Col != "status" {
		t.Fatalf("filters = %+v, want status is null", m.spec.Filters)
	}
	m = drainBatch(t, m, cmd)
	if len(m.rows) != 1 || m.rows[0][0] != "US" {
		t.Errorf("rows = %v, want the null-status row", m.rows)
	}
}

func TestFreqEnterEmptyRow(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "e.ndjson")
	if err := os.WriteFile(p, []byte("{\"v\":\"x\"}\n{\"v\":\"\"}\n{\"v\":null}\n{\"v\":\"x\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := engine.Open(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	m := New(s)
	m.headers = displayNames(m.spec)
	next, cmd := m.Update(press("v"))
	m = drainBatch(t, next.(Model), cmd)
	if m.rows[1][0] != engine.FreqEmptyText {
		t.Fatalf("rows = %v, want the empty row at 1", m.rows)
	}
	next, _ = m.Update(press("j"))
	m = next.(Model)
	next, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if len(m.spec.Filters) != 1 {
		t.Fatalf("filters = %+v", m.spec.Filters)
	}
	if f := m.spec.Filters[0]; f.Op != engine.OpEq || f.Value != "" || f.ColType != "VARCHAR" {
		t.Errorf("filter = %+v, want v = empty string", f)
	}
	m = drainBatch(t, m, cmd)
	if len(m.rows) != 1 || m.rows[0][0] != "" {
		t.Errorf("rows = %v, want the empty-string row", m.rows)
	}
}

func TestFreqMarkedRowsAddInFilter(t *testing.T) {
	m := freqModel(t)
	space := tea.KeyPressMsg{Code: ' '}
	next, _ := m.Update(space)
	m = next.(Model)
	next, _ = m.Update(press("j"))
	m = next.(Model)
	next, _ = m.Update(space)
	m = next.(Model)
	if len(m.marked) != 2 {
		t.Fatalf("marked = %v", m.marked)
	}
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if len(m.spec.Filters) != 1 || m.spec.Filters[0].Op != engine.OpIn {
		t.Fatalf("filters = %+v, want one in filter", m.spec.Filters)
	}
	if !reflect.DeepEqual(m.spec.Filters[0].Values, []string{"EU-CENT", "AP-SOUTH"}) {
		t.Errorf("values = %v", m.spec.Filters[0].Values)
	}
	if !strings.Contains(m.renderStatus(), "region in (EU-CENT, AP-SOUTH)") {
		t.Errorf("status = %q", m.renderStatus())
	}
	m = drainBatch(t, m, cmd)
	if len(m.rows) != 3 {
		t.Errorf("rows = %d, want 3", len(m.rows))
	}
	back, _ := m.Update(press("u"))
	if b := back.(Model); b.spec.Freq != nil || len(b.spec.Filters) != 0 {
		t.Errorf("u after drill-down leaves %+v, want the unfiltered parent", b.spec)
	}
}

func TestFreqMarkedNullDroppedFromIn(t *testing.T) {
	m := modelOn(t, "nulls.csv")
	m.colCursor = 1
	next, cmd := m.Update(press("v"))
	m = drainBatch(t, next.(Model), cmd)
	space := tea.KeyPressMsg{Code: ' '}
	next, _ = m.Update(space)
	m = next.(Model)
	next, _ = m.Update(press("j"))
	m = next.(Model)
	next, _ = m.Update(space)
	m = next.(Model)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if len(m.spec.Filters) != 1 {
		t.Fatalf("filters = %+v", m.spec.Filters)
	}
	if f := m.spec.Filters[0]; f.Op != engine.OpIn || !reflect.DeepEqual(f.Values, []string{"OK"}) {
		t.Errorf("filter = %+v, want in (OK)", f)
	}
	m = drainBatch(t, m, cmd)
	if len(m.rows) != 3 {
		t.Errorf("rows = %d, want the 3 OK rows", len(m.rows))
	}
}

func TestFreqSpaceToggles(t *testing.T) {
	m := freqModel(t)
	space := tea.KeyPressMsg{Code: ' '}
	next, _ := m.Update(space)
	m = next.(Model)
	if len(m.marked) != 1 {
		t.Fatalf("marked = %v", m.marked)
	}
	next, _ = m.Update(space)
	m = next.(Model)
	if len(m.marked) != 0 {
		t.Errorf("marked = %v, want empty after the second space", m.marked)
	}
}

func TestFreqMarkedRowRendersStyled(t *testing.T) {
	m := freqModel(t)
	plain := m.renderBody()
	next, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	m = next.(Model)
	if m.renderBody() == plain {
		t.Error("a marked row must render differently")
	}
}

func TestFreqOtherRowCannotBeSelected(t *testing.T) {
	m := freqModel(t)
	m.rows = [][]string{{engine.FreqOtherText, "7", "58.33", "100", strings.Repeat("\u2588", 12)}}
	space := tea.KeyPressMsg{Code: ' '}
	next, _ := m.Update(space)
	m = next.(Model)
	if len(m.marked) != 0 || m.errText == "" {
		t.Error("space on (other) must be refused")
	}
	m.errText = ""
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if m.spec.Freq == nil || m.errText == "" {
		t.Error("enter on (other) must be refused")
	}
}
