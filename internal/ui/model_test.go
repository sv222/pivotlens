package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

func TestStalePageMsgIsIgnored(t *testing.T) {
	s := testSession(t)
	m := New(s)
	m.pageReq = 5
	m.rows = [][]string{{"current"}}
	next, _ := m.Update(pageMsg{id: 3, rows: [][]string{{"stale"}}, cols: []string{"c"}})
	got := next.(Model)
	if got.rows[0][0] != "current" {
		t.Errorf("stale pageMsg overwrote rows: %v", got.rows)
	}
}

func TestFreshPageMsgIsApplied(t *testing.T) {
	s := testSession(t)
	m := New(s)
	m.pageReq = 5
	next, _ := m.Update(pageMsg{id: 5, rows: [][]string{{"fresh"}}, cols: []string{"c"}})
	got := next.(Model)
	if got.rows[0][0] != "fresh" {
		t.Errorf("fresh pageMsg not applied: %v", got.rows)
	}
}

func TestStaleCountMsgIsIgnored(t *testing.T) {
	s := testSession(t)
	m := New(s)
	m.specReq = 4
	m.total = 100
	m.counted = true
	next, _ := m.Update(countMsg{id: 2, n: 7})
	if got := next.(Model); got.total != 100 {
		t.Errorf("stale countMsg applied: total = %d", got.total)
	}
}

func TestScrollDoesNotInvalidatePendingCount(t *testing.T) {
	s := testSession(t)
	m := New(s)
	spec := m.specReq
	next, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if got := next.(Model); got.specReq != spec {
		t.Errorf("scrolling bumped specReq from %d to %d", spec, got.specReq)
	}
}

func TestCursorClampedToCountedTotal(t *testing.T) {
	s := testSession(t)
	m := New(s)
	m.total, m.counted = 5, true
	next, _ := m.gotoRow(999)
	if got := next.(Model); got.cursor != 4 {
		t.Errorf("cursor = %d, want 4", got.cursor)
	}
	next, _ = m.gotoRow(-3)
	if got := next.(Model); got.cursor != 0 {
		t.Errorf("cursor = %d, want 0", got.cursor)
	}
}

func TestCrossingBlockBoundaryRefetches(t *testing.T) {
	s := testSession(t)
	m := New(s)
	m.total, m.counted = 100000, true
	before := m.pageReq
	next, cmd := m.gotoRow(PageSize + 1)
	if cmd == nil {
		t.Fatal("crossing a block boundary must issue a fetch")
	}
	got := next.(Model)
	if got.pageReq == before {
		t.Error("pageReq not bumped")
	}
	if got.blockOff != PageSize {
		t.Errorf("blockOff = %d, want %d", got.blockOff, PageSize)
	}
}

func TestPad(t *testing.T) {
	if got := pad("ab", 5); got != "ab   " {
		t.Errorf("pad short = %q", got)
	}
	if got := pad("abcdef", 4); got != "abc…" {
		t.Errorf("pad long = %q", got)
	}
	if got := pad("abcd", 4); got != "abcd" {
		t.Errorf("pad exact = %q", got)
	}
}

func TestGridGolden(t *testing.T) {
	s := testSession(t)
	tm := teatest.NewTestModel(t, New(s), teatest.WithInitialTermSize(100, 14))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 14})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return len(b) > 0
	}, teatest.WithDuration(5*time.Second))
	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	final := tm.FinalModel(t).(Model)
	teatest.RequireEqualOutput(t, []byte(final.View().Content))
}
