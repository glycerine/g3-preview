package subpaving

import (
	"errors"
	"testing"
)

func TestSubpavingFastContextRootInitializationOrderAndResume(t *testing.T) {
	m, c, v, root := fastMonomialFixture(t, []bool{false, false})
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	x, err := c.DefineMonomial([]FastPower{{Variable: 0, Degree: 2}})
	must(err)
	makeAtom := func(variable uint32, value int32, lower bool) *FastInequality {
		t.Helper()
		must(m.SetInt32(v, value))
		a, e := c.NewInequality(variable, v, lower, false)
		must(e)
		return a
	}
	must(c.AddUnitClause(makeAtom(0, 2, true), true))
	must(c.AddUnitClause(makeAtom(0, 3, false), true))
	clause, err := c.AddClause([]*FastInequality{makeAtom(1, 5, true), makeAtom(0, 0, false)}, false, true)
	must(err)
	if clause.atoms[0].variable != 0 {
		t.Fatal("source stable clause order")
	}
	c.limit.Limit, c.limit.Steps = 1, 0
	if err := c.InitializeRoot(); !errors.Is(err, ErrFastContextCanceled) || c.unitHead != 1 || c.initStage != 0 {
		t.Fatal("first root checkpoint", c.unitHead, c.initStage, err)
	}
	fastMonomialCheckBound(t, m, root, v, 0, 2, true)
	if root.upper[0] != nil || root.lower[1] != nil {
		t.Fatal("unasserted suffix already published")
	}
	c.limit.Limit, c.limit.Steps = 1000, 0
	must(c.InitializeRoot())
	if c.unitHead != 2 || c.initStage != 3 || c.initDefHead != uint32(len(c.isInt)) || len(c.queue) != 0 || c.queueHead != 0 {
		t.Fatal("root initialization did not finish", c.unitHead, c.initStage, c.initDefHead, len(c.queue))
	}
	fastMonomialCheckBound(t, m, root, v, 0, 2, true)
	fastMonomialCheckBound(t, m, root, v, 0, 3, false)
	fastMonomialCheckBound(t, m, root, v, x, 4, true)
	fastMonomialCheckBound(t, m, root, v, x, 9, false)
	fastMonomialCheckBound(t, m, root, v, 1, 5, true)
	if root.inconsistent || clause.visited == 0 {
		t.Fatal("root queue clause propagation", root.inconsistent, clause.visited)
	}
	trail, timestamp, visits := root.trail, c.timestamp, c.numVisited
	must(c.InitializeRoot())
	if root.trail != trail || c.timestamp != timestamp || c.numVisited != visits {
		t.Fatal("root initialization repeated work")
	}
}

func TestSubpavingFastContextRootInitializationUnitConflict(t *testing.T) {
	m, c, v, root := fastMonomialFixture(t, []bool{false})
	for _, tc := range []struct {
		value int32
		lower bool
	}{{5, true}, {4, false}} {
		if err := m.SetInt32(v, tc.value); err != nil {
			t.Fatal(err)
		}
		a, err := c.NewInequality(0, v, tc.lower, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddUnitClause(a, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	if !root.inconsistent || root.conflictVar != 0 || c.initStage != 3 || len(c.queue) != 0 {
		t.Fatal("unit conflict did not end initialization", root.inconsistent, root.conflictVar, c.initStage)
	}
}
