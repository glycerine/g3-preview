package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func rationalClauseFixture(t *testing.T) (*RationalContext, *numeral.RationalManager, *numeral.Rational, uint32, uint32) {
	t.Helper()
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewRationalContext(m, &control.Reslimit{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	y, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	v := numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{c.Close(), m.Del(&v), m.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("rational clause retained numeral budget", budget.Used())
		}
	})
	return c, m, &v, x, y
}

func TestRationalClauseExactTruthAndRegistration(t *testing.T) {
	c, m, v, x, y := rationalClauseFixture(t)
	if err := m.SetInt32(v, 5); err != nil {
		t.Fatal(err)
	}
	loClosed, _ := c.NewInequality(x, v, true, false)
	loOpen, _ := c.NewInequality(x, v, true, true)
	hiClosed, _ := c.NewInequality(x, v, false, false)
	hiOpen, _ := c.NewInequality(x, v, false, true)
	yAtom, _ := c.NewInequality(y, v, true, false)
	clause, err := c.AddClause([]*RationalInequality{yAtom, loOpen, hiClosed, loClosed})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.clauseWatchers[x]) != 1 || len(c.clauseWatchers[y]) != 1 || clause.atoms[0] != loOpen || clause.atoms[1] != hiClosed || clause.atoms[2] != loClosed || clause.atoms[3] != yAtom {
		t.Fatal("stable atom order or distinct-variable watches")
	}
	if _, err := c.AddClause([]*RationalInequality{loClosed, nil}); !errors.Is(err, arena.ErrInvalidSlot) || loClosed.refs != 1 {
		t.Fatal("invalid clause changed references", err, loClosed.refs)
	}
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, x, v, false, false); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		a    *RationalInequality
		want rationalTruth
	}{{loClosed, rationalUnknown}, {loOpen, rationalFalse}, {hiClosed, rationalTrue}, {hiOpen, rationalUnknown}} {
		got, err := c.inequalityValue(tc.a, root)
		if err != nil || got != tc.want {
			t.Fatalf("closed upper tie: got %v, want %v, err %v", got, tc.want, err)
		}
	}
	if err := c.PublishBound(root, x, v, false, true); err != nil {
		t.Fatal(err)
	}
	if got, err := c.inequalityValue(hiOpen, root); err != nil || got != rationalTrue {
		t.Fatal("open upper tie", got, err)
	}
	if got, err := c.inequalityValue(loClosed, root); err != nil || got != rationalFalse {
		t.Fatal("open upper falsifies closed lower", got, err)
	}
}

func TestRationalClauseQueuedWatchCancellationAndConflict(t *testing.T) {
	c, m, v, x, y := rationalClauseFixture(t)
	if err := m.SetInt32(v, 5); err != nil {
		t.Fatal(err)
	}
	xLower, _ := c.NewInequality(x, v, true, false)
	yLower, _ := c.NewInequality(y, v, true, false)
	clause, err := c.AddClause([]*RationalInequality{yLower, xLower})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(v, 4); err != nil {
		t.Fatal(err)
	}
	_, err = c.AddInequality(x, v, false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	root, err := c.AssertRootUnits()
	if err != nil || len(c.queue) != 1 {
		t.Fatal("root upper unit", err)
	}
	c.limit.Limit = c.limit.Steps
	if err := c.PropagateQueued(root); !errors.Is(err, ErrRationalContextCanceled) || c.queueHead != 0 {
		t.Fatal("checkpoint before watched bound", err, c.queueHead)
	}
	c.limit.Limit = 0
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	if root.lower[y] == nil || root.lower[y].reasonClause != clause || root.lower[y].axiom || root.conflictVar != math.MaxUint32 || c.numVisited != 1 || clause.visited != c.timestamp || len(c.queue) != 0 {
		t.Fatal("queued watch failed to publish and drain justified bound")
	}
	if err := c.PropagateQueued(root); err != nil || c.numVisited != 1 {
		t.Fatal("watched bound replayed", err, c.numVisited)
	}
}

func TestRationalClauseAllFalseFirstAtomConflict(t *testing.T) {
	c, m, v, x, y := rationalClauseFixture(t)
	if err := m.SetInt32(v, 5); err != nil {
		t.Fatal(err)
	}
	xLower, _ := c.NewInequality(x, v, true, false)
	yLower, _ := c.NewInequality(y, v, true, false)
	clause, err := c.AddClause([]*RationalInequality{yLower, xLower})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(v, 4); err != nil {
		t.Fatal(err)
	}
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	for _, variable := range []uint32{x, y} {
		if err := c.PublishBound(root, variable, v, false, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.PropagateClause(clause, root); err != nil || root.conflictVar != x || root.trail.reasonClause != clause || c.numConflicts != 1 {
		t.Fatal("all-false clause did not record sorted first-atom conflict", err, root.conflictVar)
	}
}
