package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalContextSourceBoundConflicts(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	c, err := NewRationalContext(m, &control.Reslimit{Limit: 100})
	must(err)
	x, err := c.NewVariable(false)
	must(err)
	root, err := c.Root()
	must(err)
	lo, hi := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Del(&lo); err != nil {
			t.Error(err)
		}
		if err := m.Del(&hi); err != nil {
			t.Error(err)
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("retained conflict budget", budget.Used())
		}
	})
	must(m.SetInt32(&lo, 1))
	must(m.SetInt32(&hi, 2))
	closed, err := c.NewChild(root)
	must(err)
	must(c.PublishBound(closed, x, &lo, true, false))
	conflict, err := closed.ConflictingBounds(x)
	must(err)
	if conflict || closed.conflictVar != math.MaxUint32 {
		t.Fatal("one-sided bound conflicts")
	}
	must(c.PublishBound(closed, x, &lo, false, false))
	conflict, err = closed.ConflictingBounds(x)
	must(err)
	if conflict || closed.conflictVar != math.MaxUint32 {
		t.Fatal("closed singleton conflicts")
	}
	open, err := c.NewChild(root)
	must(err)
	must(c.PublishBound(open, x, &lo, true, true))
	must(c.PublishBound(open, x, &lo, false, false))
	if open.conflictVar != x || c.numConflicts != 1 {
		t.Fatal("open singleton conflict", open.conflictVar, c.numConflicts)
	}
	if open.queuedLeaf || c.leafHead != closed || c.leafTail != root {
		t.Fatal("open conflict remained queued")
	}
	if err := c.PublishBound(open, x, &hi, false, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("published after conflict: %v", err)
	}
	reversed, err := c.NewChild(root)
	must(err)
	must(c.PublishBound(reversed, x, &hi, true, false))
	must(c.PublishBound(reversed, x, &lo, false, false))
	if reversed.conflictVar != x || c.numConflicts != 2 || c.numBounds != 6 {
		t.Fatal("reversed conflict counters", reversed.conflictVar, c.numConflicts, c.numBounds)
	}
	if reversed.queuedLeaf || c.leafHead != closed || c.leafTail != root {
		t.Fatal("reversed conflict remained queued")
	}
	selected, err := c.PopLeafBFS()
	must(err)
	if selected != root {
		t.Fatal("BFS did not pop oldest root")
	}
	selected, err = c.PopLeafBFS()
	must(err)
	if selected != closed {
		t.Fatal("BFS did not pop remaining child")
	}
	selected, err = c.PopLeafBFS()
	must(err)
	if selected != nil || c.leafHead != nil || c.leafTail != nil {
		t.Fatal("empty rational leaf list")
	}
	inherited, err := c.NewChild(reversed)
	must(err)
	if inherited.conflictVar != x {
		t.Fatal("child lost conflict status")
	}
	if err := c.PublishBound(inherited, x, &lo, false, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("inherited conflict allowed publication: %v", err)
	}
}
