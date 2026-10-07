// Source: src/util/small_object_allocator.cpp:72-109; src/util/small_object_allocator.h.
// Source revision: Leonardo de Moura (leonardo) 2011-04-27; rewrote/simplified allocator.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSmallAllocatorCheckedActualSizeTEntry(t *testing.T) {
	for _, p := range []SmallAllocatorProfile{{PointerBits: 32}, {PointerBits: 32, Debug: true}, {PointerBits: 32, Debug: true, Windows: true}} {
		t.Run(profileName(p), func(t *testing.T) {
			a, d, r, b := rawAllocatorPair(t, p)
			before := r.State()
			used := b.Used()
			x, e := a.Allocate(uint64(1) << 32)
			if e != nil || x != (ObjectRef{}) {
				t.Fatal("source size_t zero", x, e)
			}
			q, e := a.DeallocateTracked(uint64(1)<<32, ObjectRef{owner: 999, slot: 999, generation: 999})
			if e != nil || q != (ReleaseProgress{}) {
				t.Fatal("unused pointer at converted zero", q, e)
			}
			if r.State() != before || b.Used() != used || a.allocSize != 0 {
				t.Fatal("zero source action")
			}
			x, e = a.Allocate((uint64(1) << 32) + 300)
			smallObjectMust(t, e)
			loc, e := a.object(x)
			smallObjectMust(t, e)
			if loc.requested != 300 {
				t.Fatal("converted allocation", loc.requested)
			}
			ptr := rawAllocatorPointer(t, a, x)
			rawAllocatorGraph(t, a, d, r, b, ptr.Block)
			q, e = a.DeallocateTracked((uint64(1)<<32)+300, x)
			smallObjectMust(t, e)
			if !q.PhysicalReleased || a.allocSize != 0 {
				t.Fatal("converted release", q, a.allocSize)
			}
			rawAllocatorGraph(t, a, d, r, b)
			rawAllocatorFinish(t, a, d, r, b)
		})
	}
}
func TestSmallAllocatorCheckedZeroPreservesStoppedRawInvocation(t *testing.T) {
	for _, p := range []SmallAllocatorProfile{{PointerBits: 64}, {PointerBits: 32}, {PointerBits: 64, Debug: true, Windows: true}} {
		t.Run(profileName(p), func(t *testing.T) {
			a, d, r, b := rawAllocatorPair(t, p)
			av := smallObjectAlloc(t, a, 17)
			dv := smallObjectAlloc(t, d, 17)
			ap := rawAllocatorPointer(t, a, av)
			dp := rawAllocatorPointer(t, d, dv)
			b.FailAfter(0)
			_, e := a.DeallocateRawTracked(17, dp)
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("real sidecar refusal", e)
			}
			frame := a.rawRelease
			state := r.State()
			stats := r.Stats()
			own := a.Stats()
			donor := d.Stats()
			used := b.Used()
			inputs := []uint64{0}
			if p.PointerBits == 32 {
				inputs = append(inputs, uint64(1)<<32)
			}
			for _, n := range inputs {
				x, e := a.Allocate(n)
				if e != nil || x != (ObjectRef{}) {
					t.Fatal("zero allocation during raw stop", x, e)
				}
				q, e := a.DeallocateTracked(n, ObjectRef{owner: 999, slot: 999, generation: 999})
				if e != nil || q != (ReleaseProgress{}) {
					t.Fatal("zero release during raw stop", q, e)
				}
			}
			if a.rawRelease != frame || r.State() != state || r.Stats() != stats || a.Stats() != own || d.Stats() != donor || b.Used() != used {
				t.Fatal("zero changed exact stopped graph")
			}
			rawAllocatorGraph(t, a, d, r, b, ap.Block, dp.Block)
			if _, e = a.Allocate(1); !errors.Is(e, ErrAllocatorPending) {
				t.Fatal("nonzero stop guard", e)
			}
			b.FailAfter(-1)
			_, _, e = a.AbortRawInvocation()
			smallObjectMust(t, e)
			smallObjectMust(t, d.Deallocate(17, dv))
			rawAllocatorFinish(t, a, d, r, b)
		})
	}
}
func TestSmallAllocatorCheckedZeroRetainsIdentityLifetimeGuards(t *testing.T) {
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 32})
	copy := *a
	if _, e := copy.Allocate(uint64(1) << 32); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("copied zero allocation", e)
	}
	if _, e := copy.DeallocateTracked(0, ObjectRef{}); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("copied zero release", e)
	}
	smallObjectMust(t, a.Close())
	if _, e := a.Allocate(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed zero allocation", e)
	}
	if _, e := a.DeallocateTracked(0, ObjectRef{}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed zero release", e)
	}
	rawAllocatorFinish(t, a, d, r, b)
}
func TestSmallAllocatorCheckedDeallocationSizeTEntry(t *testing.T) {
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 32, Debug: true})
	x := smallObjectAlloc(t, a, 300)
	p := rawAllocatorPointer(t, a, x)
	q, e := a.DeallocateTracked((uint64(1)<<32)+300, x)
	if e != nil || !q.PhysicalReleased {
		t.Fatal("source converted bypass release", q, e)
	}
	rawAllocatorGraph(t, a, d, r, b)
	if _, e = r.Read(p.Block, 0); e == nil {
		t.Fatal("actual provider release absent")
	}
	rawAllocatorFinish(t, a, d, r, b)
}

func TestSmallAllocatorCheckedNormalizedOwningReceipt(t *testing.T) {
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 32, Debug: true})
	x := smallObjectAlloc(t, a, 300)
	n := (uint64(1) << 32) + 300
	q, e := a.DeallocateTracked(n, x)
	smallObjectMust(t, e)
	before := r.State()
	used := b.Used()
	if e = a.CompleteRelease(n, x, &q); e != nil {
		t.Fatal("matching converted source invocation", e)
	}
	if !q.PhysicalReleased || r.State() != before || b.Used() != used {
		t.Fatal("receipt replayed actual release")
	}
	if e = a.CompleteRelease(n+1, x, &q); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("mismatched converted size accepted", e)
	}
	foreign := q
	foreign.serial++
	if e = a.CompleteRelease(n, x, &foreign); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("mismatched serial accepted", e)
	}
	rawAllocatorGraph(t, a, d, r, b)
	rawAllocatorFinish(t, a, d, r, b)
}
func TestSmallAllocatorCheckedNormalizedStoppedOwningReceipt(t *testing.T) {
	// Typed dangling-pointer continuation domain: the donor explicitly destroys
	// its page after the receiver has reused its actual foreign free-list pointer.
	a, d, r, b := rawAllocatorPair(t, SmallAllocatorProfile{PointerBits: 32})
	av := smallObjectAlloc(t, a, 17)
	dv := smallObjectAlloc(t, d, 17)
	ap := rawAllocatorPointer(t, a, av)
	dp := rawAllocatorPointer(t, d, dv)
	_, e := a.DeallocateRawTracked(17, dp)
	smallObjectMust(t, e)
	x := smallObjectAlloc(t, a, 17)
	smallObjectMust(t, d.Reset())
	n := (uint64(1) << 32) + 17
	q, cause := a.DeallocateTracked(n, x)
	if cause == nil || !q.RequestedAdjusted || q.PooledLinked {
		t.Fatal("actual stopped pointer-write prefix", q, cause)
	}
	count := a.allocSize
	state := r.State()
	used := b.Used()
	e = a.CompleteRelease(n, x, &q)
	if !errors.Is(e, cause) || errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("matching invocation must reach original pointer guard", e, cause)
	}
	if a.allocSize != count || r.State() != state || b.Used() != used {
		t.Fatal("owning retry repeated requested count")
	}
	rawAllocatorGraph(t, a, d, r, b, ap.Block)
	// Explicit owning chunk cleanup consumes only receiver-owned storage.
	smallObjectMust(t, a.CleanupChunks())
	rawAllocatorFinish(t, a, d, r, b)
}
