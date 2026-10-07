// Source: src/sat/sat_clause.h/.cpp; src/util/id_gen.h; src/util/vector.h.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATClauseRuntimeIDFreeVectorIsActualSourceStorage(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, IDRuntime: r})
	satOK(t, e)
	first, e := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, e)
	id, e := a.ID(first)
	satOK(t, e)
	before := r.Stats().LiveBlocks
	satOK(t, a.Delete(first))
	roots := a.runtimeIDs.RawRoots()
	if roots[0] == (util.Block{}) || roots[1] != (util.Block{}) || r.Stats().LiveBlocks != before+1 {
		t.Fatal("source unsigned free-ID vector lacks real allocation")
	}
	cap, e := a.runtimeIDs.Capacity()
	satOK(t, e)
	extent, e := r.Size(roots[0])
	satOK(t, e)
	if extent != 8+4*uint64(cap) {
		t.Fatal("source vector actual byte extent", extent, cap)
	}
	second, e := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, e)
	got, e := a.ID(second)
	satOK(t, e)
	if got != id {
		t.Fatal("actual LIFO ID", got, id)
	}
	satOK(t, a.Delete(second))
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("complete actual ID/allocator graph", b.Used())
	}
}
func TestSATClauseRuntimeIDRecycleRefusalKeepsClauseThenOOMSkips(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, IDRuntime: r})
	satOK(t, e)
	c, e := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, e)
	b.FailAfter(0)
	e = a.Delete(c)
	var f util.Failure
	if !errors.As(e, &f) || f.Kind != util.FailureMemout || !r.State().OutOfMemory || !a.IsLive(c) {
		t.Fatal("source recycle-before-physical-free prefix", e, r.State())
	}
	roots := a.runtimeIDs.RawRoots()
	if roots != ([2]util.Block{}) {
		t.Fatal("unallocated failed prefix", roots)
	}
	// Source sticky OOM skips recycle, before accessing the free vector. No
	// fault reset or synthetic OOM mutation is required for this caller retry.
	satOK(t, a.Delete(c))
	if a.IsLive(c) || a.runtimeIDs.RawRoots() != roots {
		t.Fatal("OOM skip changed source free-list")
	}
	b.FailAfter(-1)
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATClauseRuntimeIDReadRefusalRetainsDeclaredCandidate(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.Variables.mkVar(false, true)
		satOK(t, e)
	}
	a := s.Clauses
	c, e := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, e)
	satOK(t, a.Delete(c))
	root := a.runtimeIDs.RawRoots()[0]
	cap, e := a.runtimeIDs.Capacity()
	satOK(t, e)
	// Actual malformed raw-pointer domain: source back() reads size-1 beyond
	// the true physical free-ID vector. No source field/counter is restored.
	satOK(t, s.runtime.memory.WriteUint32(root, 4, cap+2))
	candidate, e := s.Lifecycle.mkNaryClauseSpan(4, satSolverLiteralSpan{literals: []Literal{0, 2, 4, 6}}, true)
	if !errors.Is(e, arena.ErrBounds) || candidate == (StoredClauseRef{}) || !a.IsLive(candidate) {
		t.Fatal(candidate, e)
	}
	if s.Lifecycle.pendingCandidate != candidate || !s.Lifecycle.pendingUnconstructed || s.Lifecycle.pendingDeclaredSize != 4 {
		t.Fatal("lost actual pre-placement declaration", s.Lifecycle.pendingCandidate)
	}
	if len(s.Lifecycle.learned) != 0 || len(s.Lifecycle.Original) != 0 {
		t.Fatal("source vector publication preceded failed ID read")
	}
	if e = a.Delete(candidate); e == nil || !a.IsLive(candidate) {
		t.Fatal("source Delete suppressed reached unconstructed/header/ID guard", e)
	}
	satOK(t, s.Lifecycle.cleanupCandidate())
	if a.IsLive(candidate) || s.Lifecycle.pendingCandidate != (StoredClauseRef{}) {
		t.Fatal("owning abort did not consume real declared candidate")
	}
	// The malformed source ID header is still present; owning allocator Close
	// releases that actual raw root, with no fabricated size/header repair.
	if size, e := a.runtimeIDs.Size(); e != nil || size != cap+2 {
		t.Fatal("ID state repaired", size, e)
	}
	satOK(t, s.Close())
}

func TestSATClauseRuntimeIDBorrowLifetimeCheckedAtRecycleRead(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	satOK(t, e)
	a, e := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, IDRuntime: r})
	satOK(t, e)
	c, e := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, e)
	// The explicit standalone caller violates the borrowed Runtime lifetime;
	// the genuine memory OOM read must guard before ID access/physical release.
	satOK(t, r.Close())
	e = a.Delete(c)
	if !errors.Is(e, arena.ErrClosed) || !a.IsLive(c) {
		t.Fatal("closed Runtime read released source clause", e)
	}
	satOK(t, a.cleanupAllocated(c, 3))
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
