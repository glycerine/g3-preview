// Source: src/util/small_object_allocator.cpp:72-145; src/util/debug.h:63.
// Licensed under the MIT license.
// Source authors: Nikolaj bjorner (nbjorner), 2007-08-06; Leonardo de Moura (leonardo), 2006/2011.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
)

func TestSmallObjectAllocatorActualAssertionSwitchForeignDirect(t *testing.T) {
	rt, b, region, out := smallObjectDiagnosticFixture(t)
	debug, e := NewDebugContext(region, b)
	smallObjectMust(t, e)
	defer func() { smallObjectMust(t, debug.Close()) }()
	cfg := utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, Debug: debug, Diagnostics: out}
	a, e := newUtilSmallObjectAllocator(cfg)
	smallObjectMust(t, e)
	d, e := newUtilSmallObjectAllocator(cfg)
	smallObjectMust(t, e)
	v := smallObjectAlloc(t, d, 300)
	h, o, e := d.core.RawLocation(v)
	smallObjectMust(t, e)
	p := memory.RawObjectPointer{Runtime: rt, Block: h, Offset: o}
	smallObjectAssertionPhysical(t, rt, b, region, out, debug, a, d)
	before := rt.State()
	used := b.Used()
	_, e = a.core.DeallocateRawTracked(300, p)
	if !errors.Is(e, ErrSmallObjectAssertion) {
		t.Fatal("enabled source assertion", e)
	}
	if n, e := a.AllocationSize(); e != nil || n != 0 || rt.State() != before || b.Used() != used {
		t.Fatal("enabled stopped before subtraction", n, e)
	}
	smallObjectMust(t, debug.EnableAssertions(false))
	q, e := a.core.DeallocateRawTracked(300, p)
	smallObjectMust(t, e)
	if !q.PhysicalReleased || !q.RequestedAdjusted {
		t.Fatal("disabled assertions must reach actual raw source free", q)
	}
	n, e := a.AllocationSize()
	smallObjectMust(t, e)
	if n != math.MaxUint64-299 {
		t.Fatal("source unsigned receiver count", n)
	}
	smallObjectAssertionPhysical(t, rt, b, region, out, debug, a, d)
	if rt.Stats().LiveBlocks != 0 {
		t.Fatal("donor block not released")
	}
	if n, e = d.AllocationSize(); e != nil || n != 300 {
		t.Fatal("donor source counter changed", n, e)
	}
	smallObjectMust(t, a.Close())
	smallObjectMust(t, d.Close())
	text, e := out.String()
	smallObjectMust(t, e)
	if text != "Memory leak detected for small object allocator 'unknown'. 18446744073709551316 bytes leaked\nMemory leak detected for small object allocator 'unknown'. 300 bytes leaked\n" {
		t.Fatal("DEBUG leak body must remain enabled independently", text)
	}
}
func TestSmallObjectAllocatorAssertionBindingReachedAndErased(t *testing.T) {
	for _, profile := range []utilSmallObjectProfile{{PointerBits: 64}, {PointerBits: 64, Debug: true}, {PointerBits: 64, Debug: true, Windows: true}} {
		t.Run(func() string {
			if !profile.Debug {
				return "release"
			}
			if !profile.Windows {
				return "debug-direct"
			}
			return "debug-windows"
		}(), func(t *testing.T) {
			rt, b, region, out := smallObjectDiagnosticFixture(t)
			debug, e := NewDebugContext(region, b)
			smallObjectMust(t, e)
			smallObjectMust(t, debug.Close())
			a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: profile, Debug: debug, Diagnostics: out})
			smallObjectMust(t, e)
			before := rt.State()
			zero, e := a.Allocate(0)
			if e != nil || zero != (utilSmallObjectRef{}) || rt.State() != before {
				t.Fatal("zero must erase unused assertion binding", zero, e)
			}
			e = a.Deallocate(0, utilSmallObjectRef{})
			smallObjectMust(t, e)
			v, e := a.Allocate(1)
			if profile.Debug && profile.Windows {
				if !errors.Is(e, arena.ErrClosed) || v != (utilSmallObjectRef{}) {
					t.Fatal("actual reached assertions-enabled read", v, e)
				}
				if n, e := a.AllocationSize(); e != nil || n != 1 || rt.State() != before {
					t.Fatal("count before source slot assertion", n, e)
				}
			} else {
				smallObjectMust(t, e)
				smallObjectMust(t, a.Deallocate(1, v))
			}
			smallObjectMust(t, a.Close())
		})
	}
}

func TestSmallObjectAllocatorClosedAssertionOwnerDirectFreePrefix(t *testing.T) {
	rt, b, region, out := smallObjectDiagnosticFixture(t)
	debug, e := NewDebugContext(region, b)
	smallObjectMust(t, e)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, Debug: debug, Diagnostics: out})
	smallObjectMust(t, e)
	smallObjectMust(t, debug.Close())
	// The source direct Allocate branch precedes every SASSERT dispatch.
	v := smallObjectAlloc(t, a, 300)
	h, o, e := a.core.RawLocation(v)
	smallObjectMust(t, e)
	before := rt.State()
	stats := rt.Stats()
	used := b.Used()
	q, e := a.core.DeallocateTracked(300, v)
	if !errors.Is(e, arena.ErrClosed) || q.RequestedAdjusted || q.PhysicalReleased {
		t.Fatal("checked free reached closed assertion owner before subtraction", q, e)
	}
	q, e = a.core.DeallocateRawTracked(300, memory.RawObjectPointer{Runtime: rt, Block: h, Offset: o})
	if !errors.Is(e, arena.ErrClosed) || q.RequestedAdjusted || q.PhysicalReleased {
		t.Fatal("raw free reached closed assertion owner before subtraction", q, e)
	}
	if n, e := a.AllocationSize(); e != nil || n != 300 || rt.State() != before || rt.Stats() != stats || b.Used() != used {
		t.Fatal("closed assertion source-prefix physical graph", n, e)
	}
	// Separate caller raw-pointer release: no source allocator replay or repair.
	smallObjectMust(t, rt.Deallocate(&h))
	smallObjectMust(t, a.Close())
}

func smallObjectAssertionPhysical(t *testing.T, rt *utilMemoryRuntime, b *arena.Budget, region *arena.BytesRegion, out *NumericTextStream, debug *DebugContext, owners ...*utilSmallObjectAllocator) {
	t.Helper()
	rs := region.Stats()
	ds := rs.Descriptors
	stream := out.Stats()
	want := rt.Stats().TotalBytes + rs.OwnerBytes + rs.PayloadBytes + rs.PageHeaderBytes + rs.DirectoryBytes + rs.LogBytes + rs.ScopeBytes + ds.OwnerBytes + ds.RecordBytes + ds.MetadataBytes + ds.DirectoryBytes + stream.OwnerBytes + stream.ScratchBytes + stream.OutputOwnerBytes + stream.OutputOverflowBytes
	if debug != nil {
		want += debug.ownerBytes
	}
	for _, a := range owners {
		if a != nil {
			want += a.Stats().TotalBytes
		}
	}
	if b.Used() != want {
		t.Fatal("once-only actual cold/control/provider graph", b.Used(), want)
	}
}
func TestSmallObjectAllocatorAssertionControlColdEveryRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		rt, b, region, out := smallObjectDiagnosticFixture(t)
		debug, e := NewDebugContext(region, b)
		smallObjectMust(t, e)
		before := rt.State()
		used := b.Used()
		b.FailAfter(ordinal)
		a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: rt, Profile: utilSmallObjectProfile{PointerBits: 64, Debug: true, Windows: true}, Debug: debug, Diagnostics: out})
		smallObjectAssertionPhysical(t, rt, b, region, out, debug, a)
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) || a != nil || b.Used() != used {
				t.Fatal("actual cold refusal prefix", ordinal, a, e)
			}
		} else {
			success++
			if a.core.Stats().OwnerBytes != uint64(reflect.TypeOf(memory.SmallAllocator{}).Size()) || a.ioBytes != uint64(reflect.TypeOf(smallObjectIOBridge{}).Size()) {
				t.Fatal("actual reflected cold interface/header charge")
			}
		}
		if rt.State() != before || rt.Stats().LiveBlocks != 0 {
			t.Fatal("cold binding fabricated source Runtime allocation")
		}
		b.FailAfter(-1)
		if a != nil {
			smallObjectMust(t, a.Close())
		}
		smallObjectMust(t, debug.Close())
		if success != 0 {
			if ordinal != 3 {
				t.Fatal("actual first cold success horizon", ordinal)
			}
			break
		}
	}
	if failures != 3 || success != 1 {
		t.Fatal("complete actual cold frontier", failures, success)
	}
}
