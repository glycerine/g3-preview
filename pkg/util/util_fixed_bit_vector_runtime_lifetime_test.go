// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp; src/util/small_object_allocator.cpp.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFixedBitVectorRuntimeRealProfilesResetAndCapturedLifetime(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, p := range []SmallAllocatorProfile{{PointerBits: bits}, {PointerBits: bits, Debug: true}, {PointerBits: bits, Debug: true, Windows: true}} {
			for _, n := range []uint32{65, 2048} {
				b := arena.NewBudget(1 << 24)
				r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: bits})
				heapMust(t, e)
				m, e := NewRuntimeFixedBitVectorManager(n, SmallAllocatorConfig{Runtime: r, Profile: p})
				heapMust(t, e)
				v, e := m.Allocate1()
				heapMust(t, e)
				span, e := m.Capture(v)
				heapMust(t, e)
				if span.Extent != uint64(m.numBytes) || span.Profile != p {
					t.Fatal("per-object raw extent/profile", span)
				}
				before := r.Stats()
				heapMust(t, m.Reset())
				if span.Direct {
					if r.Stats().LiveBlocks != before.LiveBlocks {
						t.Fatal("direct caller survived source reset")
					}
					w, e := span.Word(0)
					heapMust(t, e)
					if w != ^uint32(0) {
						t.Fatal(w)
					}
					e = m.Deallocate(v)
					if p.Debug && p.Windows {
						if !errors.Is(e, ErrAllocatorAssertion) {
							t.Fatal("source allocSize reset before DEBUG decrement guard", e)
						}
						fixedRuntimeAudit(t, m, v)
						heapMust(t, m.Close())
						w, e = span.Word(0)
						heapMust(t, e)
						if w != ^uint32(0) {
							t.Fatal(w)
						}
						heapMust(t, span.ReleaseDirect())
					} else {
						heapMust(t, e)
						heapMust(t, m.Close())
					}
				} else {
					if _, e = span.Word(0); !errors.Is(e, ErrRuntimeOwner) {
						t.Fatal("pooled page freed", e)
					}
					if e = span.ReleaseDirect(); !errors.Is(e, ErrRuntimeVectorOwner) {
						t.Fatal("pooled slice must not release wholepage", e)
					}
					if _, e = m.Get(v, 0); !errors.Is(e, ErrAllocatorOwner) {
						t.Fatal(e)
					}
					heapMust(t, m.Close())
				}
				heapMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal(bits, p, n, b.Used())
				}
			}
		}
	}
}
func TestFixedBitVectorRuntimeDirectPostCloseCallerAndProvider(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	v, e := m.Allocate1()
	heapMust(t, e)
	span, e := m.Capture(v)
	heapMust(t, e)
	heapMust(t, m.Close())
	if _, e = m.Get(v, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	w, e := span.Word(0)
	heapMust(t, e)
	if w != ^uint32(0) {
		t.Fatal(w)
	}
	if e = r.Close(); !errors.Is(e, ErrRuntimeLive) {
		t.Fatal("actual caller root retained", e)
	}
	heapMust(t, span.ReleaseDirect())
	heapMust(t, r.Close())
	if _, e = span.Word(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestFixedBitVectorRuntimeActualDiagnosticStopOwningAcknowledgment(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	m, e := NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true}})
	heapMust(t, e)
	_, e = m.Allocate0()
	heapMust(t, e)
	// Real source leak diagnostic follows pooled frees. Missing transport stops
	// the outer owner; no injected/fake destructor callback is used.
	e = m.Close()
	if !errors.Is(e, ErrAllocatorIO) {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 || m.allocator.Stats().OwnerBytes != 0 || m.closed {
		t.Fatal("completed child, retained outer", r.Stats(), m.allocator.Stats(), m.closed)
	}
	if e = m.SourceFinalize(); !errors.Is(e, ErrAllocatorIO) {
		t.Fatal("source stop sticky", e)
	}
	if _, e = m.Allocate(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	heapMust(t, m.CleanupStopped())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestFixedBitVectorRuntimeColdCopyForeignAndWarmPlateau(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	other, e := NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	heapMust(t, e)
	a, e := m.Allocate0()
	heapMust(t, e)
	src, e := other.Allocate1()
	heapMust(t, e)
	heapMust(t, m.CopyFrom(a, other, src))
	eq, e := m.EqualFrom(a, other, src)
	heapMust(t, e)
	if !eq {
		t.Fatal("explicit foreign physical source")
	}
	if _, e = m.Get(src, 0); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	copyOwner := *m
	if e = copyOwner.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	var zero RuntimeFixedBitVectorManager
	if _, e = zero.Allocate(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	before := r.State()
	physical := r.Stats()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(20, func() {
		for i := uint32(0); i < 65; i++ {
			if e = m.SetBool(a, i, i%2 == 0); e != nil {
				panic(e)
			}
		}
		if e = m.Neg(a); e != nil {
			panic(e)
		}
		if _, e = m.Hash(a); e != nil {
			panic(e)
		}
	})
	if allocs != 0 || r.State() != before || r.Stats() != physical {
		t.Fatal("warm actual plateau", allocs, r.State(), before, r.Stats(), physical)
	}
	b.FailAfter(-1)
	heapMust(t, m.Deallocate(a))
	heapMust(t, other.Deallocate(src))
	heapMust(t, m.Close())
	heapMust(t, other.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFixedBitVectorRuntimeActualTLSUnpublishedAllocationRoot(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	heapMust(t, e)
	m, e := NewRuntimeFixedBitVectorManager(65, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	heapMust(t, e)
	external, e := r.Allocate(99980)
	heapMust(t, e)
	heapMust(t, r.SetMaxSize(1))
	out, e := m.Allocate0()
	intHeapActualRefusal(t, r, e)
	if out.object != (ObjectRef{}) {
		t.Fatal("unpublished allocation became caller value", out)
	}
	fixedRuntimeAuditExtra(t, m, nil, []Block{external})
	pending, e := m.allocator.PendingRoots()
	heapMust(t, e)
	if pending[0] == (Block{}) || r.Stats().LiveBlocks != 2 {
		t.Fatal("actual pending raw", pending, r.Stats())
	}
	heapMust(t, r.SetMaxSize(0))
	heapMust(t, m.CleanupStopped())
	heapMust(t, r.Deallocate(&external))
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestFixedBitVectorRuntimeActualPostPhysicalDirectDeallocation(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	heapMust(t, e)
	m, e := NewRuntimeFixedBitVectorManager(1<<20, SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	heapMust(t, e)
	v, e := m.Allocate0()
	heapMust(t, e)
	span, e := m.Capture(v)
	heapMust(t, e)
	contextTLSNearMinimum(t, r)
	fixedRuntimeAudit(t, m, v)
	e = m.Deallocate(v)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed-overflow-domain late sync", e)
	}
	// DeallocateTracked's actual named receipt proves physical ownership consumed;
	// the caller scalar pointer and captured source span are left unchanged.
	loc := testFieldBySourceName(reflect.ValueOf(m.allocator).Elem(), "locations").Index(int(testFieldBySourceName(reflect.ValueOf(v.object), "slot").Uint()) - 1)
	if testFieldBySourceName(loc, "live").Bool() || testFieldBySourceName(testFieldBySourceName(loc, "release"), "runtimeStage").Uint() != 3 {
		t.Fatal("actual consumed frontier", loc)
	}
	fixedRuntimeAudit(t, m)
	if _, e = span.Word(0); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("captured raw pointer now dangling", e)
	}
	if e = span.ReleaseDirect(); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("no duplicate physical release", e)
	}
	heapMust(t, m.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
