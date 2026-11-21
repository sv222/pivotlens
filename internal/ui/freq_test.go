package ui

import (
	"errors"
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
