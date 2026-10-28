package sat

// Source: src/sat/sat_clause.h/.cpp; src/util/id_gen.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestSATClauseNativeIDFreeListAndLIFO(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	satOK(t, err)
	a, err := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, OOMRuntime: r})
	satOK(t, err)
	first, err := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, err)
	id, err := a.ID(first)
	satOK(t, err)
	satOK(t, a.Delete(first))
	if a.ids.FreeLen() != 1 || a.Stats().IDFreeBytes != 0 {
		t.Fatal("native free-ID storage", a.ids.FreeLen(), a.Stats())
	}
	second, err := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, err)
	got, err := a.ID(second)
	satOK(t, err)
	if got != id || a.ids.FreeLen() != 0 {
		t.Fatal("LIFO ID reuse", got, id, a.ids.FreeLen())
	}
	satOK(t, a.Delete(second))
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("complete ID/allocator graph", b.Used())
	}
}

func TestSATClauseNativeIDRecycleUnderRefusedArena(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	satOK(t, err)
	a, err := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, OOMRuntime: r})
	satOK(t, err)
	c, err := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, err)
	b.FailAfter(0)
	satOK(t, a.Delete(c))
	if a.IsLive(c) || a.ids.FreeLen() != 1 || r.State().OutOfMemory {
		t.Fatal("native recycle charged arena", r.State(), a.ids.FreeLen())
	}
	b.FailAfter(-1)
	satOK(t, a.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATClauseIDBorrowedOOMProviderLifetime(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	satOK(t, err)
	a, err := NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}}, OOMRuntime: r})
	satOK(t, err)
	c, err := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, err)
	satOK(t, r.Close())
	err = a.Delete(c)
	if !errors.Is(err, arena.ErrClosed) || !a.IsLive(c) {
		t.Fatal("closed OOM provider released source clause", err)
	}
	satOK(t, a.cleanupAllocated(c, 3))
	satOK(t, a.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
