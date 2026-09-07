package ui

import (
	"reflect"
	"testing"
)

func TestFitClampsToBounds(t *testing.T) {
	headers := []string{"id", "description", "x"}
	maxLens := []int{2, 120, 7}
	got := Fit(maxLens, headers)
	want := []int{4, 40, 7}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFitNeverNarrowerThanHeader(t *testing.T) {
	headers := []string{"a_very_long_header_name"}
	got := Fit([]int{3}, headers)
	if got[0] != len("a_very_long_header_name") {
		t.Errorf("got %d, want %d", got[0], len("a_very_long_header_name"))
	}
}

func TestFitHandlesShortMaxLens(t *testing.T) {
	got := Fit([]int{5}, []string{"a", "b"})
	if len(got) != 2 {
		t.Fatalf("got %d widths, want 2", len(got))
	}
	if got[1] != MinColWidth {
		t.Errorf("missing maxLen should fall back to MinColWidth, got %d", got[1])
	}
}

func TestWindowCountsFittingColumns(t *testing.T) {
	widths := []int{10, 10, 10}
	if got := Window(widths, 0, 10); got != 1 {
		t.Errorf("term 10 -> %d, want 1", got)
	}
	if got := Window(widths, 0, 23); got != 2 {
		t.Errorf("term 23 -> %d, want 2", got)
	}
	if got := Window(widths, 0, 100); got != 3 {
		t.Errorf("term 100 -> %d, want 3", got)
	}
}

func TestWindowAlwaysShowsOne(t *testing.T) {
	if got := Window([]int{80}, 0, 10); got != 1 {
		t.Errorf("over-wide column -> %d, want 1", got)
	}
}

func TestWindowRespectsStart(t *testing.T) {
	if got := Window([]int{10, 10, 10}, 2, 100); got != 1 {
		t.Errorf("start 2 -> %d, want 1", got)
	}
	if got := Window([]int{10, 10, 10}, 3, 100); got != 0 {
		t.Errorf("start past end -> %d, want 0", got)
	}
}
