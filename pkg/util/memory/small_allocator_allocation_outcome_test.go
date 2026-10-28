package memory

// Source: src/util/small_object_allocator.cpp:allocate/deallocate.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSmallAllocatorOutcomeActualAcquisitionAndUnchangedQuickGuard(t *testing.T) {
	for _, direct := range []bool{false, true} {
		a, _, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64, Debug: direct})
		var first SmallAllocatorAllocationOutcome
		one, e := a.AllocateWithOutcome(32, &first)
		smallObjectMust(t, e)
		if !a.MatchesAllocationOutcome(&first) || first.Object != one || !first.FreshRaw || first.Pending != (Block{}) {
			t.Fatal("actual cold object acquisition", first)
		}
		smallObjectMust(t, a.Deallocate(32, one))
		var second SmallAllocatorAllocationOutcome
		two, e := a.AllocateWithOutcome(32, &second)
		smallObjectMust(t, e)
		if !a.MatchesAllocationOutcome(&second) || a.MatchesAllocationOutcome(&first) || first.invocation == second.invocation || second.Object != two || second.FreshRaw != direct {
			t.Fatal("actual guarded replacement/free-list object", first, second)
		}
		before, stats := second, a.Stats()
		zero, e := a.AllocateWithOutcome(0, &second)
		smallObjectMust(t, e)
		if zero != (ObjectRef{}) || second != before || a.Stats() != stats {
			t.Fatal("source zero acquired an outcome")
		}
		smallObjectMust(t, a.Deallocate(32, two))
	}
}

func TestSmallAllocatorOutcomeActualPendingLeafAndRejectedEntry(t *testing.T) {
	for _, direct := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
		smallObjectMust(t, e)
		a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
		smallObjectMust(t, e)
		external, e := r.Allocate(99980)
		smallObjectMust(t, e)
		smallObjectMust(t, r.SetMaxSize(1))
		var out SmallAllocatorAllocationOutcome
		object, e := a.AllocateWithOutcome(32, &out)
		if e == nil || object != (ObjectRef{}) || a.pending == (Block{}) || !a.MatchesAllocationOutcome(&out) || out.Pending != a.pending || !out.FreshRaw || out.Object != (ObjectRef{}) {
			t.Fatal("actual late TLS pending allocation leaf", e, out)
		}
		before, state, stats, used := out, r.State(), r.Stats(), b.Used()
		if stats.LiveBlocks != 2 {
			t.Fatal("named external and acquired pending block census", stats)
		}
		_, e = a.AllocateWithOutcome(32, &out)
		if e != ErrAllocatorPending || out != before || r.State() != state || r.Stats() != stats || b.Used() != used {
			t.Fatal("rejected entry changed exact acquired outcome", e)
		}
		smallObjectMust(t, r.SetMaxSize(0))
		smallObjectMust(t, a.CleanupPending())
		smallObjectMust(t, r.Deallocate(&external))
		smallObjectMust(t, a.Close())
		smallObjectMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("named allocation owner retained charge", b.Used())
		}
	}
}
