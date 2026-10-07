package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

func filtersFixture() engine.QuerySpec {
	q := engine.NewSpec([]engine.ColumnMeta{{Name: "status", Type: "VARCHAR"}, {Name: "amount", Type: "DOUBLE"}})
	q.Filter = "EAST"
	q.Filters = []engine.FilterSpec{
		{Col: "status", Op: engine.OpEq, Value: "FAILED", ColType: "VARCHAR"},
		{Col: "amount", Op: engine.OpGt, Value: "100", ColType: "DOUBLE"},
	}
	return q
}

func updateFilters(p Panel, msg tea.Msg) (*filtersPanel, PanelAction) {
	next, a := p.Update(msg)
	return next.(*filtersPanel), a
}

func TestFiltersPanelListsSearchFirst(t *testing.T) {
	p := newFiltersPanel(filtersFixture())
	if p.rows() != 3 {
		t.Fatalf("rows = %d, want 3", p.rows())
	}
	if got := p.View(DefaultStyles(), 80); !strings.Contains(got, "search: EAST") {
		t.Fatalf("view missing search entry: %q", got)
	}
}

func TestFiltersPanelDeleteCommits(t *testing.T) {
	p := newFiltersPanel(filtersFixture())
	p, _ = updateFilters(p, press("j"))
	p, _ = updateFilters(p, press("d"))
	_, a := updateFilters(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !a.Close || !a.Push || a.Spec == nil {
		t.Fatalf("action = %+v", a)
	}
	if len(a.Spec.Filters) != 1 || a.Spec.Filters[0].Col != "amount" {
		t.Fatalf("filters = %v", a.Spec.Filters)
	}
	if a.Spec.Filter != "EAST" {
		t.Fatalf("search lost: %q", a.Spec.Filter)
	}
}

func TestFiltersPanelDeleteSearch(t *testing.T) {
	p := newFiltersPanel(filtersFixture())
	p, _ = updateFilters(p, press("d"))
	_, a := updateFilters(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if a.Spec.Filter != "" || len(a.Spec.Filters) != 2 {
		t.Fatalf("filter %q, filters %v", a.Spec.Filter, a.Spec.Filters)
	}
}

func TestFiltersPanelEscDiscards(t *testing.T) {
	p := newFiltersPanel(filtersFixture())
	p, _ = updateFilters(p, press("d"))
	_, a := updateFilters(p, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !a.Close || a.Spec != nil || a.Push {
		t.Fatalf("action = %+v", a)
	}
}

func TestFiltersPanelEnterWithoutDeletesCloses(t *testing.T) {
	p := newFiltersPanel(filtersFixture())
	_, a := updateFilters(p, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !a.Close || a.Spec != nil || a.Push {
		t.Fatalf("action = %+v", a)
	}
}
