// Source: src/sat/sat_simplifier.cpp register_clauses/scoped_finalize; Go owning abort.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// This is an interrupted merge ownership-protocol fixture, not an injected
// comparator failure: valid source size comparisons have no allocation call.
func TestSATSimplifierStoppedMergeExactOwnersAndLifecycleClose(t *testing.T) {
	s, l := satSimplifierFixture(t, 4)
	refs := make([]StoredClauseRef, 4)
	for i := range refs {
		c, e := l.MkClauseCore(3, []Literal{0, 2, Literal(4 + i%2)}, false)
		satOK(t, e)
		refs[i] = c
	}
	bytes := 4 * uint64(reflect.TypeOf(StoredClauseRef{}).Size())
	satOK(t, l.budget.Reserve(bytes))
	// A completed first pass lives in scratch; one second-pass result has moved
	// to the vector. The vector's other scalar copies are nonowning borrows.
	scratch := make([]StoredClauseRef, 4)
	copy(scratch, refs)
	l.Original[0] = refs[2]
	s.clauseSort = satSimplifierClauseSort{scratch: scratch, bytes: bytes, n: 4, left: 0, mid: 2, right: 3, end: 4, out: 1, sourceScratch: true, active: true}
	l.simplifierContinuation = s
	satSimplifierAudit(t, s)
	if e := l.pushClause(false, refs[0]); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("stopped exact owner rejects unrelated append", e)
	}
	satOK(t, l.Close())
	if s.clauseSort.active || l.propagation.clauses.storage.Stats().Live != 0 {
		t.Fatal("actual moved owner disappeared at lifecycle teardown")
	}
	satOK(t, s.Close())
}

func TestSATSimplifierDestructorActualExplicitFinalizeBeforeMembers(t *testing.T) {
	s, l := satSimplifierFixture(t, 4)
	todo := l.propagation.variables.elimTodo
	if todo.Empty() || todo.ValuesCapacity() == 0 {
		t.Fatal("real MkVar todo setup")
	}
	satOK(t, s.Close())
	if !todo.Empty() || todo.ValuesCapacity() != 0 || todo.MembershipCapacity() != 0 {
		t.Fatal("source destructor finalize must reach actual borrowed elim_todo before member teardown")
	}
}

func TestSATSimplifierColdCopyClosedAndWarmEmptyPhysicalPlateau(t *testing.T) {
	s, l := satSimplifierFixture(t, 0)
	copyOwner := *s
	if e := copyOwner.Run(false); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copyOwner.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copyOwner.Finalize(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("copied finalize must not release actual member owners", e)
	}
	satOK(t, s.Run(false))
	satOK(t, s.Run(false))
	satOK(t, s.Run(false))
	before := l.budget.Used()
	rawBefore := l.propagation.memory.Stats()
	var runErr error
	allocs := testing.AllocsPerRun(20, func() { runErr = s.Run(false) })
	satOK(t, runErr)
	if allocs != 0 || l.budget.Used() != before || l.propagation.memory.Stats().TotalBytes != rawBefore.TotalBytes {
		t.Fatal("warmed empty actual source path", allocs, before, l.budget.Used())
	}
	satSimplifierAudit(t, s)
	satOK(t, s.Close())
	if e := s.Run(false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

// Explicit checked dangling-member domain: the caller releases the actual
// descriptor behind the borrowed source field. This is not a fake free refusal.
func TestSATSimplifierLateFinalizeGuardDoesNotReplayOuterPhases(t *testing.T) {
	s, l := satSimplifierFixture(t, 0)
	satOK(t, s.initialize())
	s.scopeActive = true
	s.running = false
	s.debugInvariant = true
	l.simplifierContinuation = s
	satOK(t, s.literalStore.Free(s.bsLS))
	before := l.propagation.limit.Count()
	e := s.scopedFinalize()
	if !errors.Is(e, arena.ErrInvalidSlot) || s.scopedStage != 4 || s.finalizeStage != 6 || !s.scopeActive {
		t.Fatal("real late guard and completed cursor", e, s.scopedStage, s.finalizeStage)
	}
	satSimplifierAudit(t, s)
	// Caller acknowledgment of its own external destruction, not reconstruction
	// or a repair of live source values. Owning Close now resumes only cleanup.
	s.bsLS = satSolverLiteralListRef{}
	satOK(t, s.Close())
	want := uint64(0)
	if satStorageSourceDebug {
		want = 1
	}
	if l.propagation.limit.Count()-before != want {
		t.Fatal("completed invariant was replayed after later Finalize guard")
	}
}
