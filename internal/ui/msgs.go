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
