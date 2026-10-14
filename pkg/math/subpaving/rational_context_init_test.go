package subpaving

import (
	"errors"
	"math"
	"testing"
)

func TestRationalContextInitializeRootResumesUnitsAndWatchedQueue(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	if err := m.SetInt32(value, 5); err != nil {
		t.Fatal(err)
	}
	xLower, err := c.NewInequality(x, value, true, false)
	if err != nil {
		t.Fatal(err)
	}
	yLower, err := c.NewInequality(y, value, true, false)
	if err != nil {
		t.Fatal(err)
	}
	clause, err := c.AddClause([]*RationalInequality{yLower, xLower})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddInequality(x, value, false, false, true); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddInequality(y, value, false, false, true); err != nil {
		t.Fatal(err)
	}
	c.limit.Limit = 1
	if err := c.InitializeRoot(); !errors.Is(err, ErrRationalContextCanceled) || c.unitHead != 1 || c.initStage != 0 || len(c.queue) != 1 || c.numBounds != 1 {
		t.Fatal("root did not retain first unit on cancellation", err, c.unitHead, c.initStage)
	}
	c.limit.Limit = 3
	if err := c.InitializeRoot(); !errors.Is(err, ErrRationalContextCanceled) || c.unitHead != 2 || c.initStage != 1 || len(c.queue) != 2 || c.queueHead != 0 || c.numBounds != 2 {
		t.Fatal("root did not retain second unit before queued cancellation", err, c.unitHead, c.initStage)
	}
	c.limit.Limit = 0
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	n := c.root
	if n == nil || n.conflictVar != math.MaxUint32 || n.lower[y] == nil || n.lower[y].reasonClause != clause || c.initStage != 2 || c.numBounds != 3 || c.numVisited != 1 || len(c.queue) != 0 || c.queueHead != 0 {
		t.Fatal("root did not finish watched propagation once", n, c.initStage, c.numBounds, c.numVisited)
	}
	steps := c.limit.Steps
	if err := c.InitializeRoot(); err != nil || c.limit.Steps != steps || c.numBounds != 3 || c.numVisited != 1 {
		t.Fatal("completed root initialization replayed", err, c.limit.Steps, steps)
	}
}
