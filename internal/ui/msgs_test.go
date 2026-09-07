package ui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sv222/pivotlens/internal/engine"
)

func testSession(t *testing.T) *engine.Session {
	t.Helper()
	s, err := engine.Open(context.Background(), filepath.Join("..", "..", "testdata", "tiny.csv"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestFetchPageCmdCarriesID(t *testing.T) {
	s := testSession(t)
	q := engine.NewSpec(s.Columns())
	msg := fetchPageCmd(context.Background(), s, q, 7, 2, 0)()
	pm, ok := msg.(pageMsg)
	if !ok {
		t.Fatalf("got %T, want pageMsg", msg)
	}
	if pm.id != 7 {
		t.Errorf("id = %d, want 7", pm.id)
	}
	if pm.err != nil {
		t.Fatal(pm.err)
	}
	if len(pm.rows) != 2 || len(pm.cols) != 4 {
		t.Errorf("got %d rows, %d cols", len(pm.rows), len(pm.cols))
	}
}

func TestCountCmdCarriesID(t *testing.T) {
	s := testSession(t)
	q := engine.NewSpec(s.Columns())
	msg := countCmd(context.Background(), s, q, 3)()
	cm, ok := msg.(countMsg)
	if !ok {
		t.Fatalf("got %T, want countMsg", msg)
	}
	if cm.id != 3 || cm.n != 5 || cm.err != nil {
		t.Errorf("got %+v", cm)
	}
}

func TestWidthsCmdCarriesID(t *testing.T) {
	s := testSession(t)
	q := engine.NewSpec(s.Columns())
	msg := widthsCmd(context.Background(), s, q, 9)()
	wm, ok := msg.(widthsMsg)
	if !ok {
		t.Fatalf("got %T, want widthsMsg", msg)
	}
	if wm.id != 9 || wm.err != nil || len(wm.w) != 4 {
		t.Errorf("got %+v", wm)
	}
}

func TestCmdCapturesError(t *testing.T) {
	s := testSession(t)
	q := engine.NewSpec(s.Columns())
	q.Cols[0].Cast = "NOT_A_TYPE"
	msg := fetchPageCmd(context.Background(), s, q, 1, 1, 0)()
	pm := msg.(pageMsg)
	if pm.err == nil {
		t.Error("bad cast must surface as pageMsg.err, not a panic")
	}
}
