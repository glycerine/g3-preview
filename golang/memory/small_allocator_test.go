// Copyright (c) 2006, 2007, 2015 Microsoft Corporation.
// Original source allocator tests; source actions and names preserved.
package memory

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func smallObjectFixture(t *testing.T, p SmallAllocatorProfile) (*SmallAllocator, *Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: p.PointerBits})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: p})
	if e != nil {
		rt.Close()
		t.Fatal(e)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		if e := a.CleanupPending(); e != nil {
			t.Error(e)
		}
		if e := a.Close(); e != nil {
			t.Error(e)
		}
		if e := rt.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("retained physical charge", b.Used())
		}
	})
	return a, rt, b
}
func smallObjectMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func smallObjectAlloc(t *testing.T, a *SmallAllocator, n uint64) ObjectRef {
	t.Helper()
	r, e := a.Allocate(n)
	smallObjectMust(t, e)
	return r
}
func smallObjectPointerWord(t *testing.T, rt *Runtime, h Block, offset uint64, bits uint8) uint64 {
	t.Helper()
	var word uint64
	for i := uint64(0); i < uint64(bits/8); i++ {
		v, e := rt.Read(h, offset+i)
		smallObjectMust(t, e)
		word |= uint64(v) << (i * 8)
	}
	return word
}

func TestSmallObjectAllocatorRegisteredPrintOnlyReplay(t *testing.T) {
	profiles := []SmallAllocatorProfile{{PointerBits: 64}, {PointerBits: 32}, {PointerBits: 64, Debug: true}, {PointerBits: 32, Debug: true}, {PointerBits: 64, Debug: true, Windows: true}, {PointerBits: 32, Debug: true, Windows: true}}
	for _, profile := range profiles {
		a, _, _ := smallObjectFixture(t, profile)
		// Exact active tst_small_object_allocator actions; source has zero ENSURE.
		p1 := smallObjectAlloc(t, a, 13)
		q1 := smallObjectAlloc(t, a, 14)
		p2 := smallObjectAlloc(t, a, 13)
		smallObjectMust(t, a.Deallocate(13, p1))
		smallObjectMust(t, a.Deallocate(14, q1))
		smallObjectMust(t, a.Deallocate(13, p2))
		p3 := smallObjectAlloc(t, a, 13)
		if !(profile.Debug && !profile.Windows) && p3.LogicalAddress() != p2.LogicalAddress() {
			t.Fatal("independent same-class LIFO")
		}
		smallObjectMust(t, a.Deallocate(13, p3))
		r1 := smallObjectAlloc(t, a, 1)
		r2 := smallObjectAlloc(t, a, 1)
		r3 := smallObjectAlloc(t, a, 1)
		r4 := smallObjectAlloc(t, a, 1)
		originalR1 := r1
		smallObjectMust(t, a.Deallocate(1, r1))
		smallObjectMust(t, a.Deallocate(1, r3))
		oldR3 := r3
		r1 = smallObjectAlloc(t, a, 1)
		smallObjectMust(t, a.Deallocate(1, r4))
		oldR4 := r4
		r4 = smallObjectAlloc(t, a, 1)
		oldR1 := originalR1
		r3 = smallObjectAlloc(t, a, 1)
		if !(profile.Debug && !profile.Windows) && (r1.LogicalAddress() != oldR3.LogicalAddress() || r4.LogicalAddress() != oldR4.LogicalAddress() || r3.LogicalAddress() != oldR1.LogicalAddress()) {
			t.Fatal("independent interleaved LIFO")
		}
		smallObjectMust(t, a.Deallocate(1, r1))
		smallObjectMust(t, a.Deallocate(1, r2))
		smallObjectMust(t, a.Deallocate(1, r3))
		smallObjectMust(t, a.Deallocate(1, r4))
		n, e := a.AllocationSize()
		smallObjectMust(t, e)
		if n != 0 {
			t.Fatal("all source requested bytes freed", n)
		}
	}
}

func TestSmallObjectAllocatorProfilesThresholdsAndHeaders(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, mode := range []uint8{0, 1, 2} {
			p := SmallAllocatorProfile{PointerBits: bits, Debug: mode != 0, Windows: mode == 2}
			a, rt, _ := smallObjectFixture(t, p)
			threshold := 256 - uint64(bits/8)
			for _, n := range []uint64{1, 7, 8, 9, 13, 14, 247, 248, 251, 252} {
				r := smallObjectAlloc(t, a, n)
				h, offset, e := a.RawLocation(r)
				smallObjectMust(t, e)
				extent, e := rt.Size(h)
				smallObjectMust(t, e)
				direct := (p.Debug && !p.Windows) || n >= threshold
				if direct {
					if offset != 0 || extent != n {
						t.Fatal("direct extent", p, n, offset, extent)
					}
				} else {
					if extent != 8192 || offset < 2*uint64(bits/8) || offset%uint64(bits/8) != 0 {
						t.Fatal("real source chunk layout", p, n, offset, extent)
					}
				}
				current, e := a.AllocationSize()
				smallObjectMust(t, e)
				want := n
				if p.Debug && !p.Windows {
					want = 0
				}
				if current != want {
					t.Fatal("requested counter", p, n, current, want)
				}
				smallObjectMust(t, a.Write(r, n-1, 0x7b))
				v, e := a.Read(r, n-1)
				smallObjectMust(t, e)
				if v != 0x7b {
					t.Fatal(v)
				}
				smallObjectMust(t, a.Deallocate(n, r))
			}
			if p.Debug && !p.Windows {
				if a.Stats().Chunks != 0 {
					t.Fatal("debug non-Windows bypass built pooled chunks")
				}
				w, e := a.WastedSize()
				smallObjectMust(t, e)
				if w != 0 {
					t.Fatal(w)
				}
			}
		}
	}
}

func TestSmallObjectAllocatorIntrusiveLinksAndPreservedBytes(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: bits})
		x := smallObjectAlloc(t, a, 13)
		y := smallObjectAlloc(t, a, 14)
		z := smallObjectAlloc(t, a, 13)
		smallObjectMust(t, a.Write(z, 12, 0xac))
		h, off, e := a.RawLocation(z)
		smallObjectMust(t, e)
		smallObjectMust(t, a.Deallocate(13, x))
		smallObjectMust(t, a.Deallocate(14, y))
		smallObjectMust(t, a.Deallocate(13, z))
		if link := smallObjectPointerWord(t, rt, h, off, bits); link != uint64(y.slot) {
			t.Fatal("free link not in actual object bytes", link, y.slot)
		}
		z2 := smallObjectAlloc(t, a, 13)
		if z2.LogicalAddress() != z.LogicalAddress() || z2.generation == z.generation {
			t.Fatal("physical reuse/generation")
		}
		v, e := a.Read(z2, 12)
		smallObjectMust(t, e)
		if v != 0xac {
			t.Fatal("recycle overwrote non-link bytes", v)
		}
		if _, e = a.Read(z, 0); e == nil {
			t.Fatal("stale checked token accepted")
		}
		smallObjectMust(t, a.Deallocate(13, z2))
		free, e := a.NumFreeObjects()
		smallObjectMust(t, e)
		waste, e := a.WastedSize()
		smallObjectMust(t, e)
		if free != 3 || waste != 48 {
			t.Fatal("intrusive getter traversal", free, waste)
		}
	}
}

func TestSmallObjectAllocatorStrictChunkEndpointConsolidateQuirk(t *testing.T) {
	a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	refs := make([]ObjectRef, 1022)
	for i := range refs {
		refs[i] = smallObjectAlloc(t, a, 1)
	}
	if a.Stats().Chunks != 2 || rt.Stats().PhysicalBytes != 2*(8192+8) {
		t.Fatal("strict endpoint", a.Stats(), rt.Stats())
	}
	first, _, e := a.RawLocation(refs[0])
	smallObjectMust(t, e)
	lastFirst, _, e := a.RawLocation(refs[1020])
	smallObjectMust(t, e)
	second, _, e := a.RawLocation(refs[1021])
	smallObjectMust(t, e)
	if first != lastFirst || first == second {
		t.Fatal("wrong strict boundary")
	}
	for _, r := range refs {
		smallObjectMust(t, a.Deallocate(1, r))
	}
	smallObjectMust(t, a.Consolidate())
	if a.Stats().Chunks != 2 {
		t.Fatal("repaired source floor-count quirk", a.Stats())
	}
	free, e := a.NumFreeObjects()
	smallObjectMust(t, e)
	if free != 1022 {
		t.Fatal(free)
	}
}

func TestSmallObjectAllocatorConsolidateReclaimsRemainderClass(t *testing.T) {
	a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	refs := make([]ObjectRef, 340)
	for i := range refs {
		refs[i] = smallObjectAlloc(t, a, 17)
	}
	if a.Stats().Chunks != 1 {
		t.Fatal(a.Stats())
	}
	for _, r := range refs {
		smallObjectMust(t, a.Deallocate(17, r))
	}
	smallObjectMust(t, a.Consolidate())
	if a.Stats().Chunks != 0 || rt.Stats().PhysicalBytes != 0 {
		t.Fatal("source exact floor equality did not reclaim", a.Stats(), rt.Stats())
	}
	free, e := a.NumFreeObjects()
	smallObjectMust(t, e)
	if free != 0 {
		t.Fatal(free)
	}
}

func TestSmallObjectAllocatorRequestedPrefixBeforeRealRawRefusal(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: bits})
		smallObjectMust(t, rt.SetMaxSize(1))
		r, e := a.Allocate(7)
		var z Failure
		if !errors.As(e, &z) || z.ErrorCode() != uint32(101) || r != (ObjectRef{}) {
			t.Fatal("source raw refusal", r, e)
		}
		count, e := a.AllocationSize()
		smallObjectMust(t, e)
		s := rt.State()
		if count != 7 || s.GlobalSize != 8192+int64(bits/8) || s.GlobalCount != 1 || rt.Stats().LiveBlocks != 0 || a.Stats().Chunks != 0 {
			t.Fatal("counter/header action prefix", count, s, rt.Stats(), a.Stats())
		}
	}
}

func TestSmallObjectAllocatorConsolidateLexicalScratchAndFailure(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: bits})
		x := smallObjectAlloc(t, a, 1)
		y := smallObjectAlloc(t, a, 13)
		smallObjectMust(t, a.Deallocate(1, x))
		smallObjectMust(t, a.Deallocate(13, y))
		before := rt.State()
		physical := rt.Stats().PhysicalBytes
		smallObjectMust(t, a.Consolidate())
		after := rt.State()
		if after.GlobalCount != before.GlobalCount+2 || rt.Stats().PhysicalBytes != physical || rt.Stats().LiveBlocks != 2 {
			t.Fatal("two cap2 scratch vectors retained/reused then destroyed", before, after, rt.Stats())
		}
	}
	a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	x := smallObjectAlloc(t, a, 1)
	smallObjectMust(t, a.Deallocate(1, x))
	before := rt.State()
	physical := rt.Stats().PhysicalBytes
	smallObjectMust(t, rt.SetMaxCount(uint64(before.GlobalCount+1)))
	e := a.Consolidate()
	var z Failure
	if !errors.As(e, &z) || z.ErrorCode() != uint32(113) {
		t.Fatal("second scratch refusal", e)
	}
	if rt.State().GlobalCount != before.GlobalCount+2 || rt.Stats().PhysicalBytes != physical || rt.Stats().LiveBlocks != 1 {
		t.Fatal("lexical first scratch not cleaned", rt.State(), rt.Stats())
	}
	free, e := a.NumFreeObjects()
	smallObjectMust(t, e)
	if free != 1 {
		t.Fatal("class heads changed before gather completed", free)
	}
}

func TestSmallObjectAllocatorDirectCallerLifetimeClose(t *testing.T) {
	a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	ref := smallObjectAlloc(t, a, 500)
	raw, off, e := a.RawLocation(ref)
	smallObjectMust(t, e)
	if off != 0 {
		t.Fatal(off)
	}
	smallObjectMust(t, a.Close())
	if rt.Stats().LiveBlocks != 1 || rt.Stats().PhysicalBytes != 508 {
		t.Fatal("invented source direct free event", rt.Stats())
	}
	if _, e = a.Read(ref, 0); e == nil {
		t.Fatal("allocator context survived destruction")
	}
	smallObjectMust(t, rt.Deallocate(&raw)) // explicit caller-owned root cleanup.
}

func TestSmallObjectAllocatorResetRetainsDirectThreeSourceProfiles(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, mode := range []uint8{0, 1, 2} {
			p := SmallAllocatorProfile{PointerBits: bits, Debug: mode != 0, Windows: mode == 2}
			a, rt, _ := smallObjectFixture(t, p)
			ref := smallObjectAlloc(t, a, 500)
			raw, _, e := a.RawLocation(ref)
			smallObjectMust(t, e)
			smallObjectMust(t, a.Write(ref, 0, 0x7e))
			smallObjectMust(t, a.Reset())
			value, e := a.Read(ref, 0)
			smallObjectMust(t, e)
			if value != 0x7e {
				t.Fatal("source Reset invalidated direct storage", p)
			}
			before := rt.Stats()
			e = a.Deallocate(500, ref)
			count, ce := a.AllocationSize()
			smallObjectMust(t, ce)
			if p.Debug && p.Windows {
				if e != ErrAllocatorAssertion || count != 0 || rt.Stats() != before {
					t.Fatal("counter assertion did not precede raw free", p, e, count, rt.Stats(), before)
				}
				smallObjectMust(t, rt.Deallocate(&raw)) // explicit cleanup after source assertion.
			} else {
				smallObjectMust(t, e)
				want := uint64(0)
				if !p.Debug {
					want = ^uint64(0) - 499
					if bits == 32 {
						want = uint64(^uint32(0)) - 499
					}
				}
				if count != want || rt.Stats().LiveBlocks != 0 {
					t.Fatal("source wrapped counter/direct free", p, count, want, rt.Stats())
				}
			}
		}
	}
}

func TestSmallObjectAllocatorIndependentModelWarmPlateau(t *testing.T) {
	a, _, b := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	var refs [37]ObjectRef
	var sizes [37]uint64
	state := uint64(0xd17a9)
	var requested uint64
	for step := 0; step < 4000; step++ {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		i := int(state % 37)
		if sizes[i] != 0 {
			smallObjectMust(t, a.Deallocate(sizes[i], refs[i]))
			requested -= sizes[i]
			sizes[i] = 0
		} else {
			n := uint64(1 + ((state >> 9) % 500))
			refs[i] = smallObjectAlloc(t, a, n)
			sizes[i] = n
			requested += n
		}
		got, e := a.AllocationSize()
		smallObjectMust(t, e)
		if got != requested {
			t.Fatal("independent requested model", step, got, requested)
		}
	}
	for i, n := range sizes {
		if n != 0 {
			smallObjectMust(t, a.Deallocate(n, refs[i]))
		}
	}
	r := smallObjectAlloc(t, a, 13)
	smallObjectMust(t, a.Deallocate(13, r))
	before := b.Used()
	stats := a.Stats()
	b.FailAfter(0)
	n := testing.AllocsPerRun(1000, func() { r, e := a.Allocate(13); smallObjectMust(t, e); smallObjectMust(t, a.Deallocate(13, r)) })
	if n != 0 || b.Used() != before || a.Stats() != stats {
		t.Fatal("warmed actual storage plateau", n, b.Used(), before, a.Stats(), stats)
	}
	b.FailAfter(-1)
}

func TestSmallObjectAllocatorScalarRefAuditAndPlacementDeletes(t *testing.T) {
	smallObjectMust(t, arena.ValidateRecordType(reflect.TypeOf(ObjectRef{})))
	a, _, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	r, e := a.Allocate(0)
	smallObjectMust(t, e)
	if r != (ObjectRef{}) {
		t.Fatal("source zero allocation")
	}
	smallObjectMust(t, a.Deallocate(0, ObjectRef{}))
	if a.PlacementDelete(r) != ErrAllocatorUnreachable || a.PlacementDeleteArray(r) != ErrAllocatorUnreachable {
		t.Fatal("placement delete fabricated a free")
	}
	copy := *a
	if _, e := copy.Allocate(1); e != ErrAllocatorOwner {
		t.Fatal("cold owner shallow copy accepted", e)
	}
}

func TestSmallObjectAllocatorConstructorAndMetadataFailurePrefixes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	smallObjectMust(t, e)
	before := b.Used()
	raw := rt.State()
	if _, e = NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 32}}); e != arena.ErrConfig || b.Used() != before {
		t.Fatal("mismatched source profile charged", e, b.Used(), before)
	}
	b.FailAfter(0)
	if _, e = NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 64}}); e != arena.ErrInjectedFailure || b.Used() != before || rt.State() != raw {
		t.Fatal("cold header refusal fabricated raw source event", e, b.Used(), rt.State())
	}
	b.FailAfter(-1)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	before = b.Used()
	raw = rt.State()
	b.FailAfter(0)
	ref, e := a.Allocate(7)
	if e != arena.ErrInjectedFailure || ref != (ObjectRef{}) {
		t.Fatal("expected cold locator refusal", ref, e)
	}
	n, e := a.AllocationSize()
	smallObjectMust(t, e)
	if n != 7 || b.Used() != before || rt.State() != raw {
		t.Fatal("source requested increment preceded cold preparation", n, b.Used(), before, rt.State(), raw)
	}
	b.FailAfter(-1)
	smallObjectMust(t, a.Close())
	smallObjectMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("constructor/metadata cleanup", b.Used())
	}
}

func TestSmallObjectAllocatorPendingCompletedRawRoots(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: bits})
		requested := ^uint64(0)
		physical := uint64(7)
		if bits == 32 {
			requested = uint64(^uint32(0))
			physical = 3
		}
		ref, e := a.Allocate(requested)
		if e != arena.ErrBounds || ref != (ObjectRef{}) || a.Stats().PendingBlocks != 1 || rt.Stats().PhysicalBytes != physical || rt.Stats().LiveBlocks != 1 {
			t.Fatal("completed wrapped raw physical root lost at header guard", bits, ref, e, a.Stats(), rt.Stats())
		}
		n, e := a.AllocationSize()
		smallObjectMust(t, e)
		if n != requested || rt.State().GlobalSize != int64(physical) {
			t.Fatal("wrapped requested/source accounting prefix", bits, n, rt.State())
		}
		header := a.Stats().OwnerBytes
		if e := a.Close(); e != ErrAllocatorPending || a.Stats().OwnerBytes != header {
			t.Fatal("pending partial Close lost charged owner", e, a.Stats(), header)
		}
		smallObjectMust(t, a.CleanupPending())
		smallObjectMust(t, a.Close())
		if a.Stats().PendingBlocks != 0 || rt.Stats().LiveBlocks != 0 || rt.State().GlobalSize != int64(physical) {
			t.Fatal("partial cleanup must retain headerless source-counter prefix", a.Stats(), rt.Stats(), rt.State())
		}
	}
	b := arena.NewBudget(1 << 24)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	smallObjectMust(t, rt.SetMaxSize(1))
	ref, e := a.Allocate(100000)
	var z Failure
	if !errors.As(e, &z) || z.ErrorCode() != uint32(101) || ref != (ObjectRef{}) || a.Stats().PendingBlocks != 1 || rt.Stats().PhysicalBytes != 100008 {
		t.Fatal("TLS postpublication refusal lost root", ref, e, a.Stats(), rt.Stats())
	}
	n, e := a.AllocationSize()
	smallObjectMust(t, e)
	if n != 100000 || rt.State().GlobalCount != 1 || rt.State().GlobalSize != 100008 {
		t.Fatal("TLS actual source counter prefix", n, rt.State())
	}
	smallObjectMust(t, a.CleanupPending())
	if rt.State().GlobalSize != 0 || rt.State().GlobalCount != 2 || rt.Stats().LiveBlocks != 0 {
		t.Fatal("source cumulative TLS free synchronization", rt.State(), rt.Stats())
	}
	smallObjectMust(t, a.Close())
	smallObjectMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("TLS pending cleanup", b.Used())
	}
}

func TestSmallObjectAllocatorConsolidateLateFreeRetainedRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		rt.accounting.count = 0
		rt.lane.count = 0
		_ = a.CleanupChunks()
		_ = a.Close()
		_ = rt.Close()
	})
	var refs [681]ObjectRef
	for i := range refs {
		refs[i] = smallObjectAlloc(t, a, 17)
	}
	for i := 0; i < 680; i++ {
		smallObjectMust(t, a.Deallocate(17, refs[i]))
	}
	head := a.chunks[3]
	// Inject the signed counter transport boundary, while all physical roots and
	// free operations remain the genuine runtime path. Scratch allocations move
	// the lane toward zero; the first actual chunk free then synchronizes.
	rt.accounting.count = 1<<63 - 1
	rt.lane.count = 1
	rt.lane.size = -100000
	e = a.Consolidate()
	if e != arena.ErrBounds || a.chunks[3] != head || a.Stats().Chunks != 2 || rt.Stats().LiveBlocks != 2 {
		t.Fatal("actual free prefix/root retention", e, a.Stats(), rt.Stats(), a.chunks[3], head)
	}
	if v, e := a.Read(refs[680], 0); e != nil && e != ErrRuntimeUninitialized {
		t.Fatal("surviving object lost", v, e)
	}
	rt.accounting.count = 0
	rt.lane.count = 0
	if e := a.Close(); e != ErrAllocatorPending {
		t.Fatal("interrupted source heads must not be traversed", e)
	}
	smallObjectMust(t, a.CleanupChunks())
	smallObjectMust(t, a.CleanupChunks())
	smallObjectMust(t, a.Close())
	smallObjectMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("remaining chunk cleanup", b.Used())
	}
}

func TestSmallObjectAllocatorPooledDuplicateFreePrefix(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: bits})
		r := smallObjectAlloc(t, a, 13)
		raw, offset, e := a.RawLocation(r)
		smallObjectMust(t, e)
		smallObjectMust(t, a.Deallocate(13, r))
		smallObjectMust(t, a.Deallocate(13, r))
		n, e := a.AllocationSize()
		smallObjectMust(t, e)
		want := ^uint64(0) - 12
		if bits == 32 {
			want = uint64(^uint32(0)) - 12
		}
		if n != want || smallObjectPointerWord(t, rt, raw, offset, bits) != uint64(r.slot) {
			t.Fatal("finite counter/relink prefix", bits, n, want)
		}
		if _, e := a.NumFreeObjects(); e != ErrAllocatorNonterminating {
			t.Fatal("source cycle continuation", e)
		}
	}
	a, rt, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true})
	r := smallObjectAlloc(t, a, 13)
	other := smallObjectAlloc(t, a, 13)
	raw, offset, e := a.RawLocation(r)
	smallObjectMust(t, e)
	smallObjectMust(t, a.Deallocate(13, r))
	smallObjectMust(t, a.Deallocate(13, r))
	n, e := a.AllocationSize()
	smallObjectMust(t, e)
	if n != 0 || smallObjectPointerWord(t, rt, raw, offset, 64) != uint64(r.slot) {
		t.Fatal("debug passed source counter assertion/relink", n)
	}
	if e := a.Write(other, 12, 7); e != nil {
		t.Fatal("other live object invalidated", e)
	}

}

func TestSmallObjectAllocatorColdFailureSweepAndResetLedger(t *testing.T) {
	failed, success := 0, 0
	for stop := int64(0); stop < 12; stop++ {
		b := arena.NewBudget(1 << 24)
		rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		smallObjectMust(t, e)
		baseline := b.Used()
		b.FailAfter(stop)
		a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 64}})
		if e == nil {
			_, e = a.Allocate(13)
		}
		if e != nil {
			var z Failure
			if e != arena.ErrInjectedFailure && (!errors.As(e, &z) || z.ErrorCode() != uint32(101)) {
				t.Fatal("unexpected bounded cold failure", stop, e)
			}
			failed++
		} else {
			success++
		}
		b.FailAfter(-1)
		if a != nil {
			smallObjectMust(t, a.CleanupPending())
			smallObjectMust(t, a.Close())
		}
		if rt.Stats().LiveBlocks != 0 || b.Used() != rt.Stats().TotalBytes || b.Used() < baseline {
			t.Fatal("unowned cold failure root", stop, rt.Stats(), b.Used(), baseline)
		}
		smallObjectMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("failure sweep teardown", stop, b.Used())
		}
	}
	if failed == 0 || success == 0 {
		t.Fatal("sweep did not cross allocation boundaries", failed, success)
	}
	a, _, b := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	first := smallObjectAlloc(t, a, 13)
	smallObjectMust(t, a.Reset())
	next := smallObjectAlloc(t, a, 13)
	if first == next || (first.slot == next.slot && first.generation >= next.generation) {
		t.Fatal("reset reused stale checked identity", first, next)
	}
	if _, e := a.Read(first, 0); e != ErrAllocatorOwner {
		t.Fatal("stale reset token", e)
	}
	b.FailAfter(0)
	smallObjectMust(t, a.Reset())
	b.FailAfter(-1)
}

func TestSmallObjectAllocatorLexicalFailedFreeRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32, ThreadLocal: true})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: rt, Profile: SmallAllocatorProfile{PointerBits: 32}})
	smallObjectMust(t, e)
	chunks := smallObjectPointerVector{owner: a}
	freeObjs := smallObjectPointerVector{owner: a}
	// Actual vector storage exists; comparator/address lookups are unused here.
	smallObjectMust(t, chunks.push(1))
	smallObjectMust(t, freeObjs.push(1))
	t.Cleanup(func() {
		rt.lane.size = 0
		_ = rt.Deallocate(&freeObjs.raw)
		_ = rt.Deallocate(&chunks.raw)
		_ = a.CleanupPending()
		_ = a.Close()
		_ = rt.Close()
	})
	rt.lane.size = -1 << 63
	if e := freeObjs.close(); e != arena.ErrBounds {
		t.Fatal("before-free signed domain", e)
	}
	if e := chunks.close(); e != arena.ErrBounds {
		t.Fatal("second before-free signed domain", e)
	}
	if a.Stats().PendingBlocks != 2 || rt.Stats().LiveBlocks != 2 || rt.Stats().PhysicalBytes != 40 {
		t.Fatal("lexical roots escaped owning graph", a.Stats(), rt.Stats())
	}
	header := a.Stats().OwnerBytes
	if e := a.Close(); e != ErrAllocatorPending || a.Stats().OwnerBytes != header {
		t.Fatal("pending lexical Close lost charged owner", e, a.Stats(), header)
	}
	before := rt.Stats()
	charge := b.Used()
	if r, e := a.Allocate(13); e != arena.ErrClosed || r != (ObjectRef{}) || rt.Stats() != before || b.Used() != charge {
		t.Fatal("destructor-complete owner accepted ordinary use", r, e, rt.Stats(), before, b.Used(), charge)
	}
	if _, e := a.WastedSize(); e != arena.ErrClosed {
		t.Fatal("destructor-complete traversal", e)
	}
	rt.lane.size = 40
	smallObjectMust(t, a.CleanupPending())
	if r, e := a.Allocate(13); e != arena.ErrClosed || r != (ObjectRef{}) {
		t.Fatal("cleanup reopened completed destructor", r, e)
	}
	smallObjectMust(t, a.CleanupPending())
	if rt.Stats().LiveBlocks != 0 || rt.lane.size != 0 {
		t.Fatal("actual cleanup events", rt.Stats(), rt.State())
	}
	smallObjectMust(t, a.Close())
	smallObjectMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("lexical cleanup charge", b.Used())
	}
}

func TestSmallObjectAllocatorGenerationExhaustionLedger(t *testing.T) {
	a, _, _ := smallObjectFixture(t, SmallAllocatorProfile{PointerBits: 64})
	r := smallObjectAlloc(t, a, 13)
	a.locations[r.slot-1].generation = ^uint32(0)
	r.generation = ^uint32(0)
	smallObjectMust(t, a.Deallocate(13, r))
	if got, e := a.Allocate(13); e != arena.ErrSlotLimit || got != (ObjectRef{}) {
		t.Fatal("generation exhausted after source free-head pop", got, e)
	}
	if !a.locations[r.slot-1].retired || a.free[2] != 0 {
		t.Fatal("retirement/free-prefix", a.locations[r.slot-1], a.free[2])
	}
	next := smallObjectAlloc(t, a, 13)
	if next.slot == r.slot {
		t.Fatal("retired identity reused", next, r)
	}
	smallObjectMust(t, a.Reset())
	after := smallObjectAlloc(t, a, 13)
	if after.slot == r.slot || !a.locations[r.slot-1].retired {
		t.Fatal("reset lost retired ledger", after, r)
	}
}
