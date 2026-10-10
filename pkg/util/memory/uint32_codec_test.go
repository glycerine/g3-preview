package memory

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMemoryUint32CodecActualBackingAndInitializedBytes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	x, e := r.Allocate(8)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.ReadUint32(x, 0); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatalf("uninitialized word: %v", e)
	}
	if e = r.Write(x, 0, 0x12); e != nil {
		t.Fatal(e)
	}
	if _, e = r.ReadUint32(x, 0); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatalf("partially initialized word: %v", e)
	}
	before := r.State()
	if e = r.WriteUint32(x, 0, 0x78563412); e != nil {
		t.Fatal(e)
	}
	for i, w := range []byte{0x12, 0x34, 0x56, 0x78} {
		v, e := r.Read(x, uint64(i))
		if e != nil || v != w {
			t.Fatalf("physical byte %d: %x %v", i, v, e)
		}
	}
	v, e := r.ReadUint32(x, 0)
	if e != nil || v != 0x78563412 {
		t.Fatalf("word: %x %v", v, e)
	}
	if r.State() != before {
		t.Fatal("codec invented allocation event")
	}
	if e = r.WriteUint32(x, 5, 1); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("endangered write: %v", e)
	}
	for _, offset := range []uint64{5, math.MaxUint64} {
		if _, e = r.ReadUint32(x, offset); !errors.Is(e, arena.ErrBounds) {
			t.Fatalf("offset %d: %v", offset, e)
		}
	}
	if e = r.Deallocate(&x); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatalf("leaked charge %d", b.Used())
	}
}

func TestMemoryAllocatorUint32CodecUsesIntrusivePayload(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(32)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.WriteUint32(x, 4, 6); e != nil {
		t.Fatal(e)
	}
	raw, off, e := a.RawLocation(x)
	if e != nil {
		t.Fatal(e)
	}
	v, e := r.ReadUint32(raw, off+4)
	if e != nil || v != 6 {
		t.Fatalf("not actual shared capacity bytes %d %v", v, e)
	}
	state := r.State()
	if e = a.Deallocate(32, x); e != nil {
		t.Fatal(e)
	}
	y, e := a.Allocate(32)
	if e != nil {
		t.Fatal(e)
	}
	if y.LogicalAddress() != x.LogicalAddress() || y.Generation() == x.Generation() {
		t.Fatal("source reuse/checked lifetime mismatch")
	}
	if _, e = a.ReadUint32(x, 4); e == nil {
		t.Fatal("stale generation accepted")
	}
	if e = a.WriteUint32(y, 4, 6); e != nil {
		t.Fatal(e)
	}
	if r.State() != state {
		t.Fatal("recycle/codec invented raw event")
	}
	if e = a.Deallocate(32, y); e != nil {
		t.Fatal(e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("leaked metadata")
	}
}

func TestMemoryMPZRoundedPhysicalObjectExtent(t *testing.T) {
	for _, tc := range []struct {
		bits   uint8
		debug  bool
		extent uint64
	}{{64, false, 8}, {32, false, 4}, {64, true, 4}} {
		b := arena.NewBudget(1 << 20)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: tc.bits})
		if e != nil {
			t.Fatal(e)
		}
		a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: tc.bits, Debug: tc.debug}})
		if e != nil {
			t.Fatal(e)
		}
		x, e := a.Allocate(4)
		if e != nil {
			t.Fatal(e)
		}
		extent, e := a.ObjectExtent(x)
		if e != nil || extent != tc.extent {
			t.Fatalf("actual rounded extent%d want%d %v", extent, tc.extent, e)
		}
		if e = a.Write(x, 4, 1); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("ordinary requested-byte API widened")
		}
		raw, off, e := a.RawLocation(x)
		if e != nil {
			t.Fatal(e)
		}
		if extent >= 8 { // Real LP64 pooled cell_size wrap still fits both header words.
			if e = r.WriteUint32(raw, off+4, 0xffffffff); e != nil {
				t.Fatal(e)
			}
			v, e := r.ReadUint32(raw, off+4)
			if e != nil || v != 0xffffffff {
				t.Fatal("actual physical header publication lost")
			}
		}
		if e = a.Deallocate(4, x); e != nil {
			t.Fatal(e)
		}
		if e = a.Close(); e != nil {
			t.Fatal(e)
		}
		if e = r.Close(); e != nil {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal("root leak")
		}
	}
}

func TestMemoryUint32CodecGuardsAndZeroAlloc(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	other, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r})
	if e != nil {
		t.Fatal(e)
	}
	foreign, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: other})
	if e != nil {
		t.Fatal(e)
	}
	x, e := a.Allocate(32)
	if e != nil {
		t.Fatal(e)
	}
	raw, off, e := a.RawLocation(x)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.WriteUint32(x, 1, 0x11223344); e != nil {
		t.Fatal(e)
	}
	if v, e := r.ReadUint32(raw, off+1); e != nil || v != 0x11223344 {
		t.Fatal("unaligned concrete byte codec mismatch")
	}
	if _, e = other.ReadUint32(raw, off); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatalf("foreign runtime %v", e)
	}
	if _, e = foreign.ObjectExtent(x); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatalf("foreign allocator %v", e)
	}
	copiedRuntime := *r
	if e = copiedRuntime.WriteUint32(raw, off, 1); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatalf("copied runtime %v", e)
	}
	copiedAllocator := *a
	if _, e = copiedAllocator.ObjectExtent(x); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatalf("copied allocator %v", e)
	}
	var nilRuntime *Runtime
	var nilAllocator *SmallAllocator
	if _, e = nilRuntime.ReadUint32(raw, 0); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatalf("nil runtime %v", e)
	}
	if e = nilAllocator.WriteUint32(x, 0, 1); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatalf("nil allocator %v", e)
	}
	if e = a.WriteUint32(x, 28, 0xaabbccdd); e != nil {
		t.Fatal(e)
	}
	if e = a.WriteUint32(x, 29, 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("endangered word published partial bytes")
	}
	if v, e := a.ReadUint32(x, 28); e != nil || v != 0xaabbccdd {
		t.Fatal("failed word changed previous value")
	}
	state, stats, used := r.State(), r.Stats(), b.Used()
	b.FailAfter(0)
	var problem error
	var wrongValue bool
	allocs := testing.AllocsPerRun(1000, func() {
		if problem != nil || wrongValue {
			return
		}
		if e := a.WriteUint32(x, 0, 7); e != nil {
			problem = e
			return
		}
		if v, e := a.ReadUint32(x, 0); e != nil || v != 7 {
			problem = e
			wrongValue = v != 7
			return
		}
		if _, e := a.ObjectExtent(x); e != nil {
			problem = e
			return
		}
		if e := r.WriteUint32(raw, off+4, 8); e != nil {
			problem = e
			return
		}
		if v, e := r.ReadUint32(raw, off+4); e != nil || v != 8 {
			problem = e
			wrongValue = v != 8
		}
	})
	if problem != nil || wrongValue || allocs != 0 || r.State() != state || r.Stats() != stats || b.Used() != used {
		t.Fatalf("warm codec hostalloc=%g err%v", allocs, problem)
	}
	b.FailAfter(-1)
	if e = a.Deallocate(32, x); e != nil {
		t.Fatal(e)
	}
	if _, e = a.ObjectExtent(x); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatalf("freed lifetime extent %v", e)
	}
	if e = a.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = a.ReadUint32(x, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("closed allocator %v", e)
	}
	if e = foreign.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if e = other.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = r.ReadUint32(raw, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("closed runtime %v", e)
	}
	if b.Used() != 0 {
		t.Fatal("owners leaked")
	}
}
