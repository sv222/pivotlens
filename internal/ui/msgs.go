package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

const DebounceDelay = 150 * time.Millisecond

type reqID uint64

type pageMsg struct {
	id   reqID
	rows [][]string
	cols []string
	err  error
}

type countMsg struct {
	id  reqID
	n   int64
	err error
}

type widthsMsg struct {
	id  reqID
	w   []int
	err error
}

type debounceMsg struct{ id reqID }

type exportMsg struct {
	id   reqID
	path string
	err  error
}

type viewSavedMsg struct {
	id   reqID
	path string
	err  error
}

type viewLoadedMsg struct {
	id   reqID
	spec engine.QuerySpec
	err  error
}

func saveViewCmd(q engine.QuerySpec, path string, id reqID) tea.Cmd {
	return func() tea.Msg {
		return viewSavedMsg{id: id, path: path, err: engine.SaveView(path, q)}
	}
}

func loadViewCmd(ctx context.Context, s *engine.Session, path string, id reqID) tea.Cmd {
	return func() tea.Msg {
		q, err := s.LoadView(ctx, path)
		return viewLoadedMsg{id: id, spec: q, err: err}
	}
}

type pivotMsg struct {
	id   reqID
	spec engine.QuerySpec
	err  error
}

func pivotCmd(ctx context.Context, s *engine.Session, q engine.QuerySpec, id reqID) tea.Cmd {
	return func() tea.Msg {
		r, err := s.ResolvePivot(ctx, q)
		return pivotMsg{id: id, spec: r, err: err}
	}
}

type statsMsg struct {
	id    reqID
	col   string
	stats engine.ColumnStats
	err   error
}

func statsCmd(ctx context.Context, s *engine.Session, q engine.QuerySpec, col string, id reqID) tea.Cmd {
	return func() tea.Msg {
		st, err := s.Stats(ctx, q, col)
		return statsMsg{id: id, col: col, stats: st, err: err}
	}
}

func exportCmd(ctx context.Context, s *engine.Session, q engine.QuerySpec, path, format string, id reqID) tea.Cmd {
	return func() tea.Msg {
		return exportMsg{id: id, path: path, err: s.Export(ctx, q, path, format)}
	}
}

func fetchPageCmd(ctx context.Context, s *engine.Session, q engine.QuerySpec, id reqID, limit, offset int) tea.Cmd {
	return func() tea.Msg {
		rows, cols, err := s.FetchPage(ctx, q, limit, offset)
		return pageMsg{id: id, rows: rows, cols: cols, err: err}
	}
}

func countCmd(ctx context.Context, s *engine.Session, q engine.QuerySpec, id reqID) tea.Cmd {
	return func() tea.Msg {
		n, err := s.Count(ctx, q)
		return countMsg{id: id, n: n, err: err}
	}
}

func widthsCmd(ctx context.Context, s *engine.Session, q engine.QuerySpec, id reqID) tea.Cmd {
	return func() tea.Msg {
		w, err := s.Widths(ctx, q, engine.WidthSample)
		return widthsMsg{id: id, w: w, err: err}
	}
}

func debounceCmd(id reqID) tea.Cmd {
	return tea.Tick(DebounceDelay, func(time.Time) tea.Msg { return debounceMsg{id: id} })
}
