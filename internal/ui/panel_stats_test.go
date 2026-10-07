package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func TestStatsKeyIssuesQuery(t *testing.T) {
	m := gridModel(t)
	next, cmd := m.Update(press("i"))
	got := next.(Model)
	if got.panel == nil {
		t.Fatal("i must open the stats panel")
	}
	if cmd == nil {
		t.Error("opening stats must issue the query command")
	}
}

func TestStatsKeyGuardsShrunkenHeaders(t *testing.T) {
	m := gridModel(t)
	m.headers = m.headers[:2]
	m.colCursor = 3
	next, cmd := m.Update(press("i"))
	got := next.(Model)
	if got.panel != nil {
		t.Error("out-of-range cursor must not open a panel")
	}
	if cmd != nil {
		t.Error("out-of-range cursor must not issue a stats command")
	}
	if got.errText == "" {
		t.Error("out-of-range cursor must set the error bar")
	}
}

func TestMetaNameResolvesAlias(t *testing.T) {
	m := gridModel(t)
	m.spec.Cols[2].Alias = "amt"
	m.headers = displayNames(m.spec)
	if m.headers[2] != "amt" {
		t.Fatalf("headers = %v, want amt at index 2", m.headers)
	}
	if got := m.metaName("amt"); got != "amount" {
		t.Errorf("metaName(amt) = %q, want amount", got)
	}
	if got := m.metaName("region"); got != "region" {
		t.Errorf("metaName(region) = %q, want region", got)
	}
	if got := m.metaName("missing"); got != "missing" {
		t.Errorf("metaName(missing) = %q, want missing", got)
	}
	m.colCursor = 2
	next, cmd := m.Update(press("i"))
	if cmd == nil {
		t.Fatal("i on an aliased column must issue the stats command")
	}
	if got := next.(Model); got.panel == nil {
		t.Error("i on an aliased column must open the stats panel")
	}
}

func TestStatsPanelLoadingState(t *testing.T) {
	m := gridModel(t)
	m.panel = newStatsPanel("status")
	if !strings.Contains(m.View().Content, "measuring") {
		t.Errorf("loading state must render: %q", m.View().Content)
	}
}

func TestStatsPanelRendersResult(t *testing.T) {
	m := gridModel(t)
	p := newStatsPanel("status")
	p.stats = engine.ColumnStats{Rows: 5, Distinct: 3, Min: "COMPLETED", Max: "PENDING"}
	p.done = true
	m.panel = p
	v := m.View().Content
	for _, want := range []string{"rows 5", "distinct 3", "min COMPLETED", "max PENDING"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q: %q", want, v)
		}
	}
	if strings.Contains(v, "avg") {
		t.Error("non-numeric stats must not show avg")
	}
}

func TestStatsPanelRendersNumeric(t *testing.T) {
	m := gridModel(t)
	p := newStatsPanel("amount")
	p.stats = engine.ColumnStats{Rows: 5, Avg: "2189.85", Sum: "10949.25", Numeric: true}
	p.done = true
	m.panel = p
	v := m.View().Content
	if !strings.Contains(v, "avg 2189.85") || !strings.Contains(v, "sum 10949.25") {
		t.Errorf("numeric fields missing: %q", v)
	}
}

func TestStatsPanelCloseKeys(t *testing.T) {
	m := gridModel(t)
	m.panel = newStatsPanel("status")
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := next.(Model); got.panel != nil {
		t.Error("esc must close the stats panel")
	}
	m.panel = newStatsPanel("status")
	next, _ = m.Update(press("q"))
	if got := next.(Model); got.panel != nil {
		t.Error("q must close the stats panel")
	}
}

func TestStatsPanelNeverChangesSpec(t *testing.T) {
	m := gridModel(t)
	m.panel = newStatsPanel("status")
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := next.(Model); len(got.hist) != 0 {
		t.Error("stats panel must not push history")
	}
}

func TestStatsMsgForwardedAndStaleDiscarded(t *testing.T) {
	m := gridModel(t)
	m.statsReq = 7
	m.panel = newStatsPanel("status")
	next, _ := m.Update(statsMsg{id: 7, stats: engine.ColumnStats{Rows: 5}})
	if got := next.(Model); got.panel.(*statsPanel).stats.Rows != 5 {
		t.Error("fresh statsMsg must reach the open panel")
	}
	m.statsReq = 7
	m.panel = newStatsPanel("status")
	next, _ = m.Update(statsMsg{id: 6, stats: engine.ColumnStats{Rows: 99}})
	if got := next.(Model); got.panel.(*statsPanel).stats.Rows != 0 {
		t.Error("stale statsMsg must be discarded")
	}
}
