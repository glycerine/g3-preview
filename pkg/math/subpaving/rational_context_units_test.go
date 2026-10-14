package subpaving

import (
	"errors"
	"math"
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
	if !errors.Is(err, ErrRationalContextCanceled) || root == nil || c.unitHead != 1 || c.numBounds != 1 || len(c.queue) != 1 || c.queue[0] != root.trail || root.trail == nil || !root.trail.axiom || root.trail.timestamp != 0 {
		t.Fatalf("first unit checkpoint: root=%p head=%d bounds=%d err=%v", root, c.unitHead, c.numBounds, err)
	}
	if root.lower[x] == nil || root.upper[x] != nil || root.conflictVar == x {
		t.Fatal("first unit prefix did not persist")
	}
	limit.Limit = 3
	again, err := c.AssertRootUnits()
	must(err)
	if again != root || c.unitHead != 2 || c.numBounds != 2 || len(c.queue) != 2 || c.queue[1] != root.trail || c.numConflicts != 1 || root.conflictVar != x || root.trail == nil || root.trail.axiom || root.trail.previous == nil || !root.trail.previous.axiom {
		t.Fatal("resumed conflicting unit prefix")
	}
	_, err = c.AssertRootUnits()
	must(err)
	if c.unitHead != 2 || c.numBounds != 2 || root.queuedLeaf || limit.Steps != 3 {
		t.Fatal("conflict replayed or checked later units", c.unitHead, c.numBounds, limit.Steps)
	}
	must(c.PropagateQueued(root))
	if len(c.queue) != 0 || c.queueHead != 0 || limit.Steps != 3 {
		t.Fatal("conflict did not clear queued suffix")
	}
}

func TestRationalContextQueuedBoundCheckpointResume(t *testing.T) {
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
	limit := &control.Reslimit{Limit: 2}
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
			t.Error("retained rational queue budget", budget.Used())
		}
	})
	must(m.SetInt32(&value, 1))
	_, err = c.AddInequality(x, &value, true, false, true)
	must(err)
	must(m.SetInt32(&value, 3))
	_, err = c.AddInequality(x, &value, false, false, true)
	must(err)
	root, err := c.AssertRootUnits()
	must(err)
	if len(c.queue) != 2 || c.queue[0].timestamp != 0 || c.queue[1].timestamp != 1 || root.conflictVar == x {
		t.Fatal("queued unit publication order")
	}
	if err := c.PropagateQueued(root); !errors.Is(err, ErrRationalContextCanceled) || c.queueHead != 0 || len(c.queue) != 2 {
		t.Fatalf("first queue checkpoint: %v, %d", err, c.queueHead)
	}
	limit.Limit = 5
	must(c.PropagateQueued(root))
	if len(c.queue) != 0 || c.queueHead != 0 || limit.Steps != 5 || c.numBounds != 2 {
		t.Fatal("queued suffix did not drain once", len(c.queue), c.queueHead, limit.Steps, c.numBounds)
	}
	must(m.SetInt32(&value, 4))
	_, err = c.AddInequality(x, &value, false, false, true)
	must(err)
	c.timestamp = math.MaxUint64 - 1
	limit.Limit = 0
	_, err = c.AssertRootUnits()
	if !errors.Is(err, ErrRationalContextTimestamp) || c.numBounds != 3 || c.unitHead != 2 || len(c.queue) != 0 || root.trail.timestamp != math.MaxUint64-1 {
		t.Fatal("stopped bound was queued or lost", err, c.numBounds, c.unitHead, len(c.queue))
	}
}
