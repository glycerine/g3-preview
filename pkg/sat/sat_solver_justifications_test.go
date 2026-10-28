package sat

// Source: src/sat/sat_solver.cpp and sat_justification.h; native value storage.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func satJustificationTestValue(t *testing.T, c *satSolverJustifications, i uint32) satJustification {
	t.Helper()
	j, e := c.read(i)
	satOK(t, e)
	return j
}
func satJustificationTestSize(t *testing.T, c *satSolverJustifications) uint32 {
	t.Helper()
	n, e := c.size()
	satOK(t, e)
	return n
}

func TestSATSolverNativeJustificationAssignmentWithoutMetadataReservation(t *testing.T) {
	p, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = 1
	c, e := p.clauses.MkClause([]Literal{0, 2, 4, 6}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(c)
	satOK(t, e)
	j := newSATClauseJustification(o)
	before, raw := b.Used(), v.memory.State()
	b.FailAfter(0)
	e = p.assignCore(0, j)
	b.FailAfter(-1)
	satOK(t, e)
	if got := satJustificationTestValue(t, &v.justification, 0); got != j {
		t.Fatal("whole native clause locator", got, j)
	}
	if v.assignment[0] != lbool.True || v.level[0] != 1 || v.phase[0] != 0 || !v.assignedSinceGC[0] {
		t.Fatal("complete assignment statement sequence")
	}
	l, e := v.trailRead(0)
	satOK(t, e)
	if l != 0 || b.Used() != before || v.memory.State() != raw {
		t.Fatal("native justification/trail changed allocator", l)
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	satOK(t, p.clauses.Delete(c))
}

func TestSATSolverNativeJustificationFiveKindsGrowthBoundsAndWarmWrites(t *testing.T) {
	p, v, b := satPropagationFixture(t, 5)
	clause, e := p.clauses.MkClause([]Literal{0, 2, 4, 6}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(clause)
	satOK(t, e)
	kinds := []satJustification{{}, newSATBinaryJustification(0xffffffff), newSATTernaryJustification(8, 0xffffffff), newSATClauseJustification(o), newSATExternalJustification(^uint32(0))}
	column := &v.justification
	before, raw := b.Used(), v.memory.State()
	b.FailAfter(0)
	want := make([]satJustification, 4097)
	for i := range want {
		want[i] = kinds[i%len(kinds)]
		if i < 5 {
			satOK(t, column.write(uint32(i), want[i]))
		} else {
			satOK(t, column.push(want[i]))
		}
	}
	b.FailAfter(-1)
	if satJustificationTestSize(t, column) != uint32(len(want)) || b.Used() != before || v.memory.State() != raw {
		t.Fatal("native growth changed allocator")
	}
	for i, j := range want {
		if got := satJustificationTestValue(t, column, uint32(i)); got != j {
			t.Fatal("kind/literal/locator lost during native growth", i, got, j)
		}
	}
	b.FailAfter(0)
	allocations := testing.AllocsPerRun(20, func() {
		satOK(t, column.write(3, kinds[3]))
		if satJustificationTestValue(t, column, 3) != kinds[3] {
			t.Fatal("warm exact locator")
		}
	})
	b.FailAfter(-1)
	if allocations != 0 || b.Used() != before || v.memory.State() != raw {
		t.Fatal("warm native writes", allocations)
	}
	copy := *column
	if _, e = copy.read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal("copied embedded owner", e)
	}
	satOK(t, column.shrink(1))
	bounds := error(arena.ErrBounds)
	if satStorageSourceDebug {
		bounds = ErrSATAssertion
	}
	if _, e = column.read(4096); !errors.Is(e, bounds) {
		t.Fatal("removed logical tail", e)
	}
	if e = column.shrink(2); !errors.Is(e, bounds) {
		t.Fatal("oversized shrink", e)
	}
	satOK(t, column.push(newSATExternalJustification(9)))
	if got := satJustificationTestValue(t, column, 1); got != newSATExternalJustification(9) {
		t.Fatal("current member after shrink and push", got)
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	satOK(t, p.clauses.Delete(clause))
	satOK(t, v.Close())
	if column.values != nil {
		t.Fatal("closed member retained native values")
	}
	if _, e = column.read(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed actual member guard", e)
	}
}

func TestSATSolverNativeJustificationStaleGenerationAndForeignAllocator(t *testing.T) {
	p, v, _ := satPropagationFixture(t, 1)
	old, e := p.clauses.MkClause([]Literal{0, 2}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(old)
	satOK(t, e)
	j := newSATClauseJustification(o)
	satOK(t, v.justification.write(0, j))
	satOK(t, p.clauses.Delete(old))
	next, e := p.clauses.MkClause([]Literal{0, 2}, false)
	satOK(t, e)
	if next.slot != old.slot || next.generation == old.generation {
		t.Fatal("real recycled generation fixture", next, old)
	}
	got := satJustificationTestValue(t, &v.justification, 0)
	if got != j {
		t.Fatal("must retain original generation", got, j)
	}
	if _, e = p.clauses.GetClause(got.ClauseOffset()); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale guard at clause dereference", e)
	}
	a := satStore(t, nil)
	foreign := satMk(t, a, []Literal{0, 2}, false)
	fo, e := a.Offset(foreign)
	satOK(t, e)
	if e = v.justification.bindClauses(a); !errors.Is(e, ErrSATOwner) {
		t.Fatal("fixed allocator binding", e)
	}
	if e = v.justification.write(0, newSATClauseJustification(fo)); !errors.Is(e, ErrSATOwner) {
		t.Fatal("foreign value guard", e)
	}
	if got := satJustificationTestValue(t, &v.justification, 0); got != j {
		t.Fatal("rejected foreign write changed native value", got, j)
	}
	no, e := p.clauses.Offset(next)
	satOK(t, e)
	satOK(t, v.justification.write(0, newSATClauseJustification(no)))
	if got := satJustificationTestValue(t, &v.justification, 0); got.ClauseOffset() != no {
		t.Fatal("current replacement locator", got, no)
	}
	satOK(t, p.clauses.Delete(next))
}
