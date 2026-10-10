// Source: src/util/small_object_allocator.cpp:allocate/deallocate.
package memory

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSmallAllocatorPendingTransferKeepsExactOlderRoot(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	smallObjectMust(t, e)
	padding, e := r.Allocate(99980)
	smallObjectMust(t, e)
	smallObjectMust(t, r.SetMaxSize(1))
	var out SmallAllocatorAllocationOutcome
	_, e = a.AllocateWithOutcome(32, &out)
	if e == nil || out.Pending == (Block{}) {
		t.Fatal(e, out)
	}
	old := out.Pending
	used := b.Used()
	stats := r.Stats()
	var owned SmallAllocatorPendingAllocation
	smallObjectMust(t, a.TransferAllocationPending(&out, &owned))
	if a.pending != (Block{}) || owned.Block() != old || r.Stats() != stats || b.Used() != used {
		t.Fatal("transfer changed physical ownership")
	}
	copied := owned
	if e = a.CleanupTransferredAllocation(&copied); e != ErrAllocatorOwner {
		t.Fatal("copied owner authorized", e)
	}
	if e = a.TransferAllocationPending(&out, &owned); e != ErrAllocatorOwner {
		t.Fatal("repeated transfer authorized", e)
	}
	smallObjectMust(t, r.SetMaxSize(0))
	var fresh SmallAllocatorAllocationOutcome
	object, e := a.AllocateWithOutcome(32, &fresh)
	smallObjectMust(t, e)
	if a.MatchesAllocationOutcome(&out) || !a.MatchesAllocationOutcome(&fresh) || owned.Block() != old {
		t.Fatal("later allocation invalidated older owner")
	}
	smallObjectMust(t, a.CleanupTransferredAllocation(&owned))
	if owned.Block() != (Block{}) {
		t.Fatal("older root retained")
	}
	smallObjectMust(t, a.CleanupTransferredAllocation(&owned))
	smallObjectMust(t, a.Deallocate(32, object))
	smallObjectMust(t, r.Deallocate(&padding))
	smallObjectMust(t, a.Close())
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSmallAllocatorPendingTransferEveryReservation(t *testing.T) {
	for _, direct := range []bool{false, true} {
		completed := false
		for limit := int64(0); limit < 32; limit++ {
			b := arena.NewBudget(1 << 24)
			r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
			smallObjectMust(t, e)
			a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: direct}})
			smallObjectMust(t, e)
			var out SmallAllocatorAllocationOutcome
			b.FailAfter(limit)
			object, body := a.AllocateWithOutcome(32, &out)
			var owned SmallAllocatorPendingAllocation
			if out.Pending != (Block{}) {
				used, stats := b.Used(), r.Stats()
				original := out.Pending
				smallObjectMust(t, a.TransferAllocationPending(&out, &owned))
				if owned.Block() != original || b.Used() != used || r.Stats() != stats {
					t.Fatal("transfer changed failed physical prefix", limit, direct)
				}
				copiedAllocator := *a
				if e = copiedAllocator.CleanupTransferredAllocation(&owned); e != ErrAllocatorOwner {
					t.Fatal("copied allocator authorized", e)
				}
			}
			b.FailAfter(-1)
			if owned.Block() != (Block{}) {
				fresh, e := a.Allocate(32)
				smallObjectMust(t, e)
				smallObjectMust(t, a.CleanupTransferredAllocation(&owned))
				smallObjectMust(t, a.Deallocate(32, fresh))
			}
			if object != (ObjectRef{}) {
				smallObjectMust(t, a.Deallocate(32, object))
			}
			smallObjectMust(t, a.CleanupPending())
			smallObjectMust(t, a.Close())
			smallObjectMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("prefix retained unnamed ownership", limit, direct, b.Used())
			}
			if body == nil {
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("allocation horizon not reached", direct)
		}
	}
}
