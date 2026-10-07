// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/sat/sat_justification.h; src/util/vector.h; src/util/memory_manager.cpp.
// Licensed under the MIT license.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverJustificationActualEightByteSourceAllocation(t *testing.T) {
	v, _, _ := satVariableStateFixture(t)
	_, e := v.mkVar(false, true)
	satOK(t, e)
	n, e := v.memory.AllocationSize()
	satOK(t, e)
	if n != 242+600 || v.memory.Stats().LiveBlocks != 14 {
		t.Fatal("actual source justification allocation absent", n, v.memory.Stats())
	}
}

func TestSATSolverJustificationProvenanceRefusalFollowsRawSourceCopy(t *testing.T) {
	p, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = 1
	c, e := p.clauses.MkClause([]Literal{0, 2, 4, 6}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(c)
	satOK(t, e)
	j := newSATClauseJustification(o)
	b.FailAfter(0)
	e = p.assignCore(0, j)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("real metadata reservation", e)
	}
	first, e := ReadRuntimeWord(v.memory, v.justification.raw.Block, 8)
	satOK(t, e)
	second, e := ReadRuntimeWord(v.memory, v.justification.raw.Block, 12)
	satOK(t, e)
	if first != j.val1 || second != j.val2 {
		t.Fatal("completed source copy before metadata refusal", first, second, j)
	}
	if _, e = v.justification.read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal("missing provenance may not infer live registry token", e)
	}
	if satAssignmentTestValue(t, &v.assignment, 0) != int8(util.LTrue) || satScalarTestUint(t, &v.level, 0) != 1 || satAssignmentTestValue(t, &v.phase, 0) != 2 || satAssignmentTestValue(t, &v.assignedSinceGC, 0) != 0 {
		t.Fatal("actual assignment statement prefix")
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	b.FailAfter(-1)
	satOK(t, v.justification.write(0, j))
	got, e := v.justification.read(0)
	satOK(t, e)
	if got != j {
		t.Fatal(got, j)
	}
	satOK(t, p.clauses.Delete(c))
}

func satJustificationTestValue(t *testing.T, c *satSolverRuntimeJustificationColumn, i uint32) satJustification {
	t.Helper()
	j, e := c.read(i)
	satOK(t, e)
	return j
}
func satJustificationTestSize(t *testing.T, c *satSolverRuntimeJustificationColumn) uint32 {
	t.Helper()
	n, e := c.size()
	satOK(t, e)
	return n
}

func TestSATSolverJustificationPushSizePublishesBeforeProvenanceRefusal(t *testing.T) {
	p, v, b := satPropagationFixture(t, 1)
	c, e := p.clauses.MkClause([]Literal{0, 2}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(c)
	satOK(t, e)
	b.FailAfter(0)
	e = v.justification.push(newSATClauseJustification(o))
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	n, e := v.justification.size()
	satOK(t, e)
	if n != 2 {
		t.Fatal("source constructor and size++ completed before metadata reserve", n)
	}
	b.FailAfter(-1)
	satOK(t, p.clauses.Delete(c))
}

func TestSATSolverJustificationFiveKindsActualWordsAndWarmMetadata(t *testing.T) {
	p, v, b := satPropagationFixture(t, 5)
	c, e := p.clauses.MkClause([]Literal{0, 2, 4, 6}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(c)
	satOK(t, e)
	values := []satJustification{{}, newSATBinaryJustification(0xffffffff), newSATTernaryJustification(8, 0xffffffff), newSATClauseJustification(o), newSATExternalJustification(^uint32(0))}
	for i, j := range values {
		satOK(t, v.justification.write(uint32(i), j))
		got, e := v.justification.read(uint32(i))
		satOK(t, e)
		if got != j {
			t.Fatal(i, got, j)
		}
		for k := uint64(0); k < 8; k++ {
			want := byte(j.val1 >> (8 * k))
			if k >= 4 {
				want = byte(j.val2 >> (8 * (k - 4)))
			}
			x, e := v.memory.Read(v.justification.raw.Block, 8+8*uint64(i)+k)
			satOK(t, e)
			if x != want {
				t.Fatal("actual source two-word layout", i, k, x, want)
			}
		}
	}
	raw, e := v.memory.Size(v.justification.raw.Block)
	satOK(t, e)
	if raw != 48 || v.justification.provenanceBytes != 60 {
		t.Fatal("source stride8 versus actual cold provenance stride12", raw, v.justification.provenanceBytes)
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	before := b.Used()
	b.FailAfter(0)
	allocations := testing.AllocsPerRun(20, func() {
		satOK(t, v.justification.write(3, values[3]))
		if satJustificationTestValue(t, &v.justification, 3) != values[3] {
			t.Fatal("warm exact locator")
		}
	})
	if allocations != 0 || b.Used() != before {
		t.Fatal("warm source copy/provenance plateau", allocations, b.Used(), before)
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	b.FailAfter(-1)
	satOK(t, p.clauses.Delete(c))
}
func TestSATSolverJustificationStaleGenerationAndForeignAllocatorTiming(t *testing.T) {
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
	got, e := v.justification.read(0)
	satOK(t, e)
	if got != j {
		t.Fatal("may not infer replacement generation", got, j)
	}
	if _, e = p.clauses.GetClause(got.ClauseOffset()); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("guard at actual clause dereference", e)
	}
	a := satStore(t, nil)
	foreign := satMk(t, a, []Literal{0, 2}, false)
	fo, e := a.Offset(foreign)
	satOK(t, e)
	e = v.justification.write(0, newSATClauseJustification(fo))
	if !errors.Is(e, ErrSATOwner) {
		t.Fatal("fixed real allocator guard", e)
	}
	tag, e := ReadRuntimeWord(v.memory, v.justification.raw.Block, 12)
	satOK(t, e)
	if tag != 3 {
		t.Fatal("source words precede namespace guard")
	}
	if _, e = v.justification.read(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal("old counterpart must remain invalid", e)
	}
	no, e := p.clauses.Offset(next)
	satOK(t, e)
	satOK(t, v.justification.write(0, newSATClauseJustification(no)))
	satOK(t, p.clauses.Delete(next))
}
func TestSATSolverJustificationGrowthCopiesSourceAndMetadataHasRealPeak(t *testing.T) {
	prepare := func(t *testing.T, b *arena.Budget) (*satSolverPropagationState, *satSolverVariableState, StoredClauseRef, satJustification) {
		p, v, _ := satPropagationFixtureWithBudget(t, 2, b)
		c, e := p.clauses.MkClause([]Literal{0, 2}, false)
		satOK(t, e)
		o, e := p.clauses.Offset(c)
		satOK(t, e)
		j := newSATClauseJustification(o)
		satOK(t, v.justification.write(0, j))
		old := v.justification.raw.Block
		for satJustificationTestSize(t, &v.justification) < 9 {
			satOK(t, v.justification.push(satJustification{}))
		}
		if _, e = v.memory.Read(old, 8); e == nil {
			t.Fatal("raw realloc lifetime")
		}
		if satJustificationTestValue(t, &v.justification, 0) != j {
			t.Fatal("old provenance and source words moved")
		}
		if v.justification.provenanceBytes != 24 {
			t.Fatal("actual old metadata", v.justification.provenanceBytes)
		}
		return p, v, c, j
	}
	calibration := arena.NewBudget(^uint64(0))
	p, v, c, j := prepare(t, calibration)
	before := calibration.Used()
	limited := arena.NewBudget(before + 143)
	qp, qv, qc, qj := prepare(t, limited)
	if limited.Used() != before {
		t.Fatal("same real fixture calibrated physical graph", limited.Used(), before)
	}
	e := qv.justification.write(8, qj)
	if !errors.Is(e, arena.ErrBudget) || qv.justification.provenanceBytes != 24 || limited.Used() != before {
		t.Fatal("reserve real whole new144 while old24 charged", e, qv.justification.provenanceBytes, limited.Used(), before)
	}
	tag, e := ReadRuntimeWord(qv.memory, qv.justification.raw.Block, 76)
	satOK(t, e)
	if tag != 3 {
		t.Fatal("source raw publication retained")
	}
	if _, e = qv.justification.read(8); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satScalarAuditRoots(t, qv)
	satPropagationAssertGraph(t, qp)
	satOK(t, qp.clauses.Delete(qc))
	satOK(t, v.justification.write(8, j))
	if v.justification.provenanceBytes != 144 || calibration.Used() != before+120 {
		t.Fatal("new metadata replaces old after success", v.justification.provenanceBytes, calibration.Used(), before)
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	satOK(t, p.clauses.Delete(c))
}

func TestSATSolverJustificationTLSPendingAtActualSourceAllocation(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	satOK(t, e)
	v, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, r)
	satOK(t, e)
	// Source malloc accounting: heap600 + external request+8 =99980, assignment18
	// completes below100000, then actual justification32 crosses the TLS check.
	prefix, e := r.Allocate(99372)
	satOK(t, e)
	satOK(t, r.SetMaxSize(1))
	_, e = v.mkVar(true, true)
	if !satAssignmentActualRefusal(r, e) || v.justification.raw.Pending == (util.Block{}) || v.justification.raw.Block != (util.Block{}) {
		t.Fatal("actual justification owned physical refusal", e, r.State())
	}
	if satAssignmentTestSize(t, &v.assignment) != 2 || satJustificationTestSize(t, &v.justification) != 0 || satAssignmentTestSize(t, &v.decision) != 0 || v.justification.provenanceBytes != 0 {
		t.Fatal("source publication before actual justification malloc", v.Stats())
	}
	satScalarAuditOwner(t, v, prefix)
	if _, e = r.Read(v.justification.raw.Pending, 0); !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal("no vector capacity/size publication", e)
	}
	satOK(t, v.Close())
	satOK(t, r.Deallocate(&prefix))
	satOK(t, todo.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverJustificationRetainedInitializedSlotAndCurrentMember(t *testing.T) {
	p, v, _ := satPropagationFixture(t, 2)
	j := newSATBinaryJustification(0xffffffff)
	satOK(t, v.justification.write(1, j))
	satOK(t, v.justification.raw.Shrink(1))
	got, e := v.justification.read(1)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
		if got != j {
			t.Fatal(got, j)
		}
	}
	// The column wrapper captures the actual member, not a stale []justification
	// shadow. Source push at retained index overwrites, then growth relocates.
	column := &v.justification
	satOK(t, column.push(newSATExternalJustification(9)))
	old := column.raw.Block
	satOK(t, column.push(newSATBinaryJustification(17)))
	if _, e = v.memory.Read(old, 8); e == nil {
		t.Fatal("captured physical payload invalidated")
	}
	if satJustificationTestValue(t, column, 2) != newSATBinaryJustification(17) {
		t.Fatal("current member after source realloc")
	}
	satScalarAuditRoots(t, v)
	satPropagationAssertGraph(t, p)
	satOK(t, v.Close())
	if _, e = column.read(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed actual member guard", e)
	}
}
