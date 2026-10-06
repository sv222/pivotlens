package ui

import (
	"context"
	"reflect"

	tea "charm.land/bubbletea/v2"

	"github.com/sv222/pivotlens/internal/engine"
)

const PageSize = 200

type Model struct {
	sess *engine.Session
	spec engine.QuerySpec
	hist []engine.QuerySpec

	panel   Panel
	pending *engine.QuerySpec

	w, h   int
	styles Styles

	rows     [][]string
	headers  []string
	widths   []int
	blockOff int

	cursor    int
	colCursor int
	colStart  int

	total   int64
	counted bool

	specReq   reqID
	pageReq   reqID
	keySeq    reqID
	exportReq reqID

	ctx     context.Context
	cancel  context.CancelFunc
	errText string
	notice  string
}

func New(s *engine.Session) Model {
	q := engine.NewSpec(s.Columns())
	ctx, cancel := context.WithCancel(context.Background())
	m := Model{
		sess:    s,
		spec:    q,
		w:       80,
		h:       24,
		styles:  DefaultStyles(),
		specReq: 1,
		pageReq: 1,
		ctx:     ctx,
		cancel:  cancel,
	}
	m.headers = displayNames(q)
	m.widths = Fit(nil, m.headers)
	return m
}

func displayNames(q engine.QuerySpec) []string {
	vis := q.Visible()
	out := make([]string, len(vis))
	for i, c := range vis {
		out[i] = c.Display()
	}
	return out
}

func blockFor(row int) int { return (row / PageSize) * PageSize }

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchPageCmd(m.ctx, m.sess, m.spec, m.pageReq, PageSize, 0),
		countCmd(m.ctx, m.sess, m.spec, m.specReq),
		widthsCmd(m.ctx, m.sess, m.spec, m.specReq),
	)
}

func (m Model) reload() (tea.Model, tea.Cmd) {
	m.cancel()
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx, m.cancel = ctx, cancel
	m.specReq++
	m.pageReq++
	m.counted = false
	m.notice = ""
	m.cursor, m.colStart, m.blockOff = 0, 0, 0
	m.headers = displayNames(m.spec)
	return m, tea.Batch(
		fetchPageCmd(ctx, m.sess, m.spec, m.pageReq, PageSize, 0),
		countCmd(ctx, m.sess, m.spec, m.specReq),
		widthsCmd(ctx, m.sess, m.spec, m.specReq),
	)
}

func (m Model) push() Model {
	m.hist = append(m.hist, m.spec.Clone())
	return m
}

func (m Model) undo() (tea.Model, tea.Cmd) {
	if len(m.hist) == 0 {
		return m, nil
	}
	m.spec = m.hist[len(m.hist)-1]
	m.hist = m.hist[:len(m.hist)-1]
	return m.reload()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil

	case pageMsg:
		if msg.id != m.pageReq {
			return m, nil
		}
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.errText = ""
		m.rows = msg.rows
		if len(msg.cols) > 0 {
			m.headers = msg.cols
			if len(m.widths) != len(msg.cols) {
				m.widths = Fit(nil, m.headers)
			}
		}
		return m, nil

	case countMsg:
		if msg.id != m.specReq {
			return m, nil
		}
		if msg.err == nil {
			m.total, m.counted = msg.n, true
		}
		return m, nil

	case widthsMsg:
		if msg.id != m.specReq {
			return m, nil
		}
		if msg.err == nil {
			m.widths = Fit(msg.w, m.headers)
		}
		return m, nil

	case debounceMsg:
		if msg.id != m.keySeq || m.pending == nil {
			return m, nil
		}
		pending := *m.pending
		m.pending = nil
		if reflect.DeepEqual(m.spec, pending) {
			return m, nil
		}
		m.spec = pending
		return m.reload()

	case exportMsg:
		if msg.id != m.exportReq {
			return m, nil
		}
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.panel = nil
		m.errText = ""
		m.notice = "wrote " + msg.path
		return m, nil

	case tea.KeyPressMsg:
		if m.panel != nil {
			np, a := m.panel.Update(msg)
			m.panel = np
			return m.applyPanelAction(a)
		}
		return m.updateGrid(msg)
	}

	if m.panel != nil {
		np, a := m.panel.Update(msg)
		m.panel = np
		return m.applyPanelAction(a)
	}
	return m, nil
}
