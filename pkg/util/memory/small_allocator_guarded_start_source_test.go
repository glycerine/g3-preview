// Source: src/util/small_object_allocator.cpp:97-144.
package memory

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSmallAllocatorGuardedStartPreservesRejectedHistory(t *testing.T) {
	a, _, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64, Debug: true})
	var o SmallAllocatorAllocationOutcome
	object, started, e := a.AllocateWithOutcomeStarted(32, &o)
	smallObjectMust(t, e)
	if !started || !a.MatchesAllocationOutcome(&o) {
		t.Fatal("actual guarded start", started, o)
	}
	before := o
	_, started, e = a.AllocateWithOutcomeStarted(0, &o)
	smallObjectMust(t, e)
	if started || o != before {
		t.Fatal("zero changed old outcome", started, o)
	}
	smallObjectMust(t, a.Deallocate(32, object))
}

func TestSmallAllocatorGuardedStartActualPendingAndValidRetry(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	smallObjectMust(t, e)
	held, e := r.Allocate(99980)
	smallObjectMust(t, e)
	smallObjectMust(t, r.SetMaxSize(1))
	var o SmallAllocatorAllocationOutcome
	object, started, cause := a.AllocateWithOutcomeStarted(32, &o)
	if cause == nil || !started || object != (ObjectRef{}) || o.Pending != (a.pending) || o.Pending == (Block{}) {
		t.Fatal("actual pending start", cause, started, o)
	}
	before, stats, state, used := o, r.Stats(), r.State(), b.Used()
	if stats.LiveBlocks != 2 {
		t.Fatal("held + actual pending census", stats)
	}
	_, started, e = a.AllocateWithOutcomeStarted(32, &o)
	if e != ErrAllocatorPending || started || o != before || r.Stats() != stats || r.State() != state || b.Used() != used {
		t.Fatal("rejected prior pending stamped/changed history", e, started, o)
	}
	// Defined source zero precedes pending guard; no invocation is acquired.
	_, started, e = a.AllocateWithOutcomeStarted(0, &o)
	smallObjectMust(t, e)
	if started || o != before {
		t.Fatal("zero with older pending changed history")
	}
	smallObjectMust(t, a.CleanupPending())
	if r.Stats().LiveBlocks != 1 {
		t.Fatal("exact pending before-limit-lift owning cleanup", r.Stats())
	}
	smallObjectMust(t, r.SetMaxSize(0))
	object, started, e = a.AllocateWithOutcomeStarted(32, &o)
	smallObjectMust(t, e)
	if !started || o.invocation != before.invocation+1 || o.Object != object || o.Pending != (Block{}) {
		t.Fatal("actual guarded new start", o, before)
	}
	smallObjectMust(t, a.Deallocate(32, object))
	smallObjectMust(t, r.Deallocate(&held))
	smallObjectMust(t, a.Close())
	before = o
	_, started, e = a.AllocateWithOutcomeStarted(32, &o)
	if e == nil || started || o != before {
		t.Fatal("closed entry stamped history", e, started)
	}
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("owned graph", b.Used())
	}
}
