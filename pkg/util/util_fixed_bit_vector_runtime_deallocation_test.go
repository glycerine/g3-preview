// Source: src/util/fixed_bit_vector.cpp:78-80; src/util/small_object_allocator.cpp:72-96.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFixedBitVectorRuntimeForeignDirectReceiverDeallocation(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	cfg := SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	a, e := NewRuntimeFixedBitVectorManager(65, cfg)
	heapMust(t, e)
	d, e := NewRuntimeFixedBitVectorManager(65, cfg)
	heapMust(t, e)
	v, e := d.Allocate0()
	heapMust(t, e)
	view, e := d.Borrow(v)
	heapMust(t, e)
	span, e := d.Capture(v)
	heapMust(t, e)
	p, e := a.DeallocateView(view)
	heapMust(t, e)
	if !p.PhysicalReleased {
		t.Fatal("actual receiver raw free", p)
	}
	if _, e = span.Word(0); e == nil {
		t.Fatal("donor actual raw block still alive")
	}
	if aSize, e := a.allocator.AllocationSize(); e != nil || aSize != 0 {
		t.Fatal(aSize, e)
	}
	if dSize, e := d.allocator.AllocationSize(); e != nil || dSize != 0 {
		t.Fatal(dSize, e)
	}
	heapMust(t, a.Close())
	heapMust(t, d.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("once-only cohort refund", b.Used())
	}
}

func TestFixedBitVectorRuntimeForeignPostPhysicalReleaseReceipt(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	heapMust(t, e)
	cfg := SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	a, e := NewRuntimeFixedBitVectorManager(1<<20, cfg)
	heapMust(t, e)
	d, e := NewRuntimeFixedBitVectorManager(1<<20, cfg)
	heapMust(t, e)
	v, e := d.Allocate0()
	heapMust(t, e)
	view, e := d.Borrow(v)
	heapMust(t, e)
	span, e := d.Capture(v)
	heapMust(t, e)
	contextTLSNearMinimum(t, r)
	before := v
	progress, e := a.DeallocateView(view)
	if !errors.Is(e, arena.ErrBounds) || !progress.PhysicalReleased || !progress.RawAccounted || progress.RequestedAdjusted {
		t.Fatal("real source late sync receipt", e, progress)
	}
	if v != before || r.Stats().LiveBlocks != 0 {
		t.Fatal("caller pointer or physical frontier", v, r.Stats())
	}
	if _, e = span.Word(0); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("actual dangling pointer guard", e)
	}
	if want := r.Stats().TotalBytes + a.ownerBytes + d.ownerBytes + a.allocator.Stats().TotalBytes + d.allocator.Stats().TotalBytes; b.Used() != want {
		t.Fatal("once-only retained cold graph", b.Used(), want)
	}
	_, _, active := a.allocator.RawReleasePending()
	if active {
		t.Fatal("already consumed raw ownership retained active")
	}
	heapMust(t, a.Close())
	heapMust(t, d.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFixedBitVectorRuntimeForeignDirectDeallocationAfterDonorClose(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	cfg := SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	a, e := NewRuntimeFixedBitVectorManager(65, cfg)
	heapMust(t, e)
	d, e := NewRuntimeFixedBitVectorManager(65, cfg)
	heapMust(t, e)
	v, e := d.Allocate0()
	heapMust(t, e)
	view, e := d.Borrow(v)
	heapMust(t, e)
	heapMust(t, d.Close())
	if e = r.Close(); !errors.Is(e, ErrRuntimeLive) {
		t.Fatal("actual provider must outlive caller raw", e)
	}
	p, e := a.DeallocateView(view)
	heapMust(t, e)
	if !p.PhysicalReleased {
		t.Fatal(p)
	}
	heapMust(t, a.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFixedBitVectorRuntimeStoppedRawInvocationExplicitOwningAbort(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	cfg := SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64}}
	// Actual fixed vectors request whole unsigned words. 129bits gives20bytes,
	// selecting the same pooled class as the underlying17byte engine fixture.
	a, e := NewRuntimeFixedBitVectorManager(129, cfg)
	heapMust(t, e)
	d, e := NewRuntimeFixedBitVectorManager(129, cfg)
	heapMust(t, e)
	av, e := a.Allocate0()
	heapMust(t, e)
	dv, e := d.Allocate1()
	heapMust(t, e)
	view, e := d.Borrow(dv)
	heapMust(t, e)
	as, e := a.Capture(av)
	heapMust(t, e)
	ds, e := d.Capture(dv)
	heapMust(t, e)
	b.FailAfter(0)
	progress, e := a.DeallocateView(view)
	if !errors.Is(e, arena.ErrInjectedFailure) || !progress.RequestedAdjusted || progress.PooledLinked {
		t.Fatal("actual source subtraction before sidecar refusal", e, progress)
	}
	rp, n, active := a.allocator.RawReleasePending()
	if !active || n != 20 {
		t.Fatal("actual pending raw action", rp, n, active)
	}
	for _, span := range []RuntimeFixedBitVectorSpan{as, ds} {
		if _, e = span.Word(0); e != nil {
			t.Fatal("named captured raw root", e)
		}
	}
	if r.Stats().LiveBlocks != 2 || b.Used() != r.Stats().TotalBytes+a.ownerBytes+d.ownerBytes+a.allocator.Stats().TotalBytes+d.allocator.Stats().TotalBytes {
		t.Fatal("immediate once-only cohort graph")
	}
	if e = a.SourceFinalize(); !errors.Is(e, ErrAllocatorPending) {
		t.Fatal("actual source destructor stop", e)
	}
	stop := a.sourceStop
	if _, e = a.DeallocateView(view); !errors.Is(e, arena.ErrClosed) || a.sourceStop != stop {
		t.Fatal("ordinary entry must preserve source stop", e)
	}
	// Owning abort must discard only receiver invocation metadata. No matching
	// source retry, donor deallocation, source count rollback or payload repair.
	heapMust(t, a.CleanupStopped())
	if a.sourceStop != stop || !a.closed || a.ownerBytes != 0 || r.Stats().LiveBlocks != 1 || !a.allocator.RawReleaseAborted() {
		t.Fatal("actual abort frontier", a.sourceStop, stop, r.Stats())
	}
	if e = a.SourceFinalize(); !errors.Is(e, stop) {
		t.Fatal("source history silently repaired", e)
	}
	for i := uint32(0); i < 5; i++ {
		w, e := d.Word(dv, i)
		heapMust(t, e)
		if w != ^uint32(0) {
			t.Fatal("donor payload rewritten", i, w)
		}
	}
	if dsize, e := d.allocator.AllocationSize(); e != nil || dsize != 20 {
		t.Fatal("donor count changed", dsize, e)
	}
	if b.Used() != r.Stats().TotalBytes+d.ownerBytes+d.allocator.Stats().TotalBytes {
		t.Fatal("receiver cold/ownpage not refunded exactly")
	}
	b.FailAfter(-1)
	heapMust(t, d.Deallocate(dv))
	heapMust(t, d.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFixedBitVectorRuntimeHeaderlessRawAbortRetainsCallerAllocation(t *testing.T) {
	// This is the actual wrapped malloc/headerless safety domain, not valid
	// ordinary free coverage. The reached raw header read must remain guarded.
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32})
	heapMust(t, e)
	cfg := SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 32, Debug: true}}
	a, e := NewRuntimeFixedBitVectorManager(129, cfg)
	heapMust(t, e)
	d, e := NewRuntimeFixedBitVectorManager(129, cfg)
	heapMust(t, e)
	v, e := d.Allocate0()
	heapMust(t, e)
	span, e := d.Capture(v)
	heapMust(t, e)
	h := span.Block
	if e = r.Reallocate(&h, uint64(^uint32(0))-3); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual headerless realloc prefix", e)
	}
	raw := RuntimeFixedBitVectorSpan{Runtime: r, Block: h, Direct: true, Profile: cfg.Profile}
	progress, e := a.DeallocateView(raw.View())
	if !errors.Is(e, arena.ErrBounds) || progress.RawAccounted || progress.PhysicalReleased || progress.RequestedAdjusted {
		t.Fatal("guarded source header prefix", e, progress)
	}
	before := r.State()
	physical := r.Stats()
	p, n, active := a.allocator.RawReleasePending()
	if !active || p.Block != h || n != 20 || physical.LiveBlocks != 1 {
		t.Fatal("actual caller/context binding", p, n, active, physical)
	}
	if b.Used() != r.Stats().TotalBytes+a.ownerBytes+d.ownerBytes+a.allocator.Stats().TotalBytes+d.allocator.Stats().TotalBytes {
		t.Fatal("immediate named headerless graph")
	}
	if e = a.SourceFinalize(); !errors.Is(e, ErrAllocatorPending) {
		t.Fatal(e)
	}
	stop := a.sourceStop
	heapMust(t, a.CleanupStopped())
	if r.State() != before || r.Stats() != physical || !a.allocator.RawReleaseAborted() || a.sourceStop != stop {
		t.Fatal("abort mutated provider/caller source history")
	}
	if e = a.SourceFinalize(); !errors.Is(e, stop) {
		t.Fatal("source stop repaired", e)
	}
	// Only the actual caller holding the headerless allocation may perform the
	// separate provider owning abort. No donor ObjectRef registry is consulted.
	heapMust(t, d.Close())
	if e = r.Close(); !errors.Is(e, ErrRuntimeLive) {
		t.Fatal("caller root lost", e)
	}
	heapMust(t, r.CleanupPartial(&h))
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
