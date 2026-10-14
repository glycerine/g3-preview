package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalContextRootUnitCheckpointConflictAndOwnership(t *testing.T) {
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
	limit := &control.Reslimit{Limit: 1}
	c, err := NewRationalContext(m, limit)
	must(err)
	x, err := c.NewVariable(false)
	must(err)
	value := numeral.NewRational()
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Del(&value); err != nil {
			t.Error(err)
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("retained rational unit budget", budget.Used())
		}
	})
	must(m.SetInt32(&value, 2))
	first, err := c.AddInequality(x, &value, true, false, true)
	must(err)
	must(m.SetInt32(&value, 1))
	second, err := c.AddInequality(x, &value, false, false, false)
	must(err)
	must(m.SetInt32(&value, 3))
	third, err := c.AddInequality(x, &value, true, false, true)
	must(err)
	if first.refs != 1 || second.refs != 1 || third.refs != 1 {
		t.Fatal("unit references were not retained")
	}
	root, err := c.AssertRootUnits()
	if !errors.Is(err, ErrRationalContextCanceled) || root == nil || c.unitHead != 1 || c.numBounds != 1 || root.trail == nil || !root.trail.axiom || root.trail.timestamp != 0 {
		t.Fatalf("first unit checkpoint: root=%p head=%d bounds=%d err=%v", root, c.unitHead, c.numBounds, err)
	}
	if root.lower[x] == nil || root.upper[x] != nil || root.conflictVar == x {
		t.Fatal("first unit prefix did not persist")
	}
	limit.Limit = 3
	again, err := c.AssertRootUnits()
	must(err)
	if again != root || c.unitHead != 2 || c.numBounds != 2 || c.numConflicts != 1 || root.conflictVar != x || root.trail == nil || root.trail.axiom || root.trail.previous == nil || !root.trail.previous.axiom {
		t.Fatal("resumed conflicting unit prefix")
	}
	_, err = c.AssertRootUnits()
	must(err)
	if c.unitHead != 2 || c.numBounds != 2 || root.queuedLeaf || limit.Steps != 3 {
		t.Fatal("conflict replayed or checked later units", c.unitHead, c.numBounds, limit.Steps)
	}
}
