// Source: src/util/bit_vector.h:69-114,191-214; src/util/bit_vector.cpp:27-247.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func bitVectorRuntimeAudit(t *testing.T, r *Runtime, vs ...*RuntimeBitVector) {
	t.Helper()
	seen := map[Block]bool{}
	var physical, bitmap, cold uint64
	for _, v := range vs {
		if v == nil || v.closed {
			continue
		}
		cold += v.ownerBytes
		hs := []Block{v.block, v.pending}
		if v.destroyed || v.pendingReplaces {
			hs[0] = Block{}
		}
		for _, h := range hs {
			if h == (Block{}) {
				continue
			}
			if seen[h] {
				t.Fatal("duplicate owning raw", h)
			}
			seen[h] = true
			n, e := r.Size(h)
			heapMust(t, e)
			physical += n + uint64(r.SizeTBits()/8)
			bitmap += ((n + uint64(r.SizeTBits()/8) + 63) / 64) * 8
		}
	}
	st := r.Stats()
	if st.LiveBlocks != uint32(len(seen)) || st.PhysicalBytes != physical || st.InitializationBytes != bitmap {
		t.Fatal("independent named physical graph", st, len(seen), physical, bitmap)
	}
	if r.Budget().Used() != st.TotalBytes+cold {
		t.Fatal("once-only graph", r.Budget().Used(), st.TotalBytes, cold)
	}
}
func TestBitVectorRuntimeRawLayoutUnsignedAndNullAssignment(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorReserved(r, b, 65)
	heapMust(t, e)
	if v.capacity != 3 || v.numBits != 0 {
		t.Fatal(v.capacity, v.numBits)
	}
	n, e := r.Size(v.block)
	heapMust(t, e)
	if n != 12 {
		t.Fatal("payload only no svector header", n)
	}
	heapMust(t, v.Resize(65, true))
	old := v.block
	c, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	heapMust(t, v.Assign(c))
	if v.numBits != 0 || v.capacity != 3 || v.block != old {
		t.Fatal("source NULL assignment preserves array")
	}
	for _, n := range []uint32{0, 1, 31, 32, 33, 0xffffffe0, 0xffffffe1, 0xffffffff} {
		if bitVectorNumWords(n) != (n+31)/32 {
			t.Fatal(n)
		}
	}
	bitVectorRuntimeAudit(t, r, v, c)
	heapMust(t, v.Close())
	heapMust(t, c.Close())
	heapMust(t, r.Close())
}
func TestBitVectorRuntimeIncompleteConstructorHasNoSourceDestructor(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorWords(r, b, []uint32{7}, 65)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if v == nil || v.capacity != 3 || v.numBits != 65 {
		t.Fatal(v)
	}
	bitVectorRuntimeAudit(t, r, v)
	old := v.block
	if e = v.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("failed class constructor has no source destructor", e)
	}
	if v.block != old {
		t.Fatal("source dtor consumed failed class payload")
	}
	heapMust(t, v.CleanupStopped())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestBitVectorRuntimeEveryActualRefusal(t *testing.T) {
	for route := 0; route < 4; route++ {
		success, failures := 0, 0
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(1 << 24)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			heapMust(t, e)
			a, e := NewRuntimeBitVector(r, b)
			heapMust(t, e)
			c, e := NewRuntimeBitVectorReserved(r, b, 513)
			heapMust(t, e)
			heapMust(t, a.Resize(1, true))
			heapMust(t, c.Resize(513, true))
			old := a.block
			b.FailAfter(ordinal)
			var out *RuntimeBitVector
			switch route {
			case 0:
				e = a.Resize(513, false)
			case 1:
				e = a.Union(c)
			case 2:
				out, e = c.Clone(b)
			case 3:
				e = a.Assign(c)
			}
			bitVectorRuntimeAudit(t, r, a, c, out)
			if e != nil {
				failures++
				intHeapActualRefusal(t, r, e)
				if route == 3 {
					if a.numBits != 513 || a.block != old || !a.destroyed {
						t.Fatal("assignment count/destroy frontier", ordinal, a.numBits, a.block, a.destroyed)
					}
				}
			} else {
				success++
			}
			b.FailAfter(-1)
			for _, v := range []*RuntimeBitVector{out, a, c} {
				if v == nil {
					continue
				}
				if v.stopped != nil || v.pendingReplaces || v.pending != (Block{}) {
					heapMust(t, v.CleanupStopped())
				} else {
					heapMust(t, v.Close())
				}
			}
			heapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(route, ordinal, b.Used())
			}
			if e == nil {
				t.Logf("route%d firstsuccess%d", route, ordinal)
				break
			}
		}
		if failures == 0 || success != 1 {
			t.Fatal(route, failures, success)
		}
	}
}
func TestBitVectorRuntimeReleaseLogicalGuardsAndScalarReads(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorReserved(r, b, 64)
	heapMust(t, e)
	got, e := v.Get(33)
	if sourceDebug {
		if !errors.Is(e, ErrRuntimeVectorAssertion) {
			t.Fatal(e)
		}
	} else {
		heapMust(t, e)
		if got {
			t.Fatal(got)
		}
		heapMust(t, v.SetBit(33))
		got, e = v.Get(33)
		heapMust(t, e)
		if !got {
			t.Fatal(got)
		}
	}
	heapMust(t, v.Close())
	heapMust(t, r.Close())
}

func TestBitVectorRuntimeSourceDestructorPostPhysicalAndOwningAbort(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorReserved(r, b, 1600000)
	heapMust(t, e)
	old := v.block
	contextTLSNearMinimum(t, r)
	e = v.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || v.block != (Block{}) || v.closed {
		t.Fatal("actual source stop", e, v.closed)
	}
	if _, e = r.Size(old); e == nil {
		t.Fatal("old block live")
	}
	bitVectorRuntimeAudit(t, r, v)
	cause := v.destructorStop
	if e = v.Close(); e != cause {
		t.Fatal(e, cause)
	}
	heapMust(t, v.CleanupStopped())
	if v.destructorStop != cause {
		t.Fatal("lost stopped history")
	}
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestBitVectorRuntimeAllLiteralScalarAndBoundaryRoutes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorReserved(r, b, 0)
	heapMust(t, e)
	if v.block == (Block{}) || v.capacity != 0 {
		t.Fatal("reserved0 still actual allocation")
	}
	n, e := r.Size(v.block)
	heapMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	heapMust(t, v.Push(true))
	heapMust(t, v.Push(false))
	back, e := v.Back()
	heapMust(t, e)
	if back {
		t.Fatal(back)
	}
	heapMust(t, v.Pop())
	back, e = v.Back()
	heapMust(t, e)
	if !back {
		t.Fatal(back)
	}
	heapMust(t, v.Reserve(33, true))
	size, e := v.Size()
	heapMust(t, e)
	if size != 33 {
		t.Fatal(size)
	}
	nw, e := v.NumWords()
	heapMust(t, e)
	if nw != 2 {
		t.Fatal(nw)
	}
	heapMust(t, v.Fill0())
	empty, e := v.Empty()
	heapMust(t, e)
	if empty {
		t.Fatal("fill0 retained size")
	}
	heapMust(t, v.Reset())
	empty, e = v.Empty()
	heapMust(t, e)
	if !empty {
		t.Fatal(empty)
	}
	e = v.Pop()
	if sourceDebug {
		if !errors.Is(e, ErrRuntimeVectorAssertion) || v.numBits != 0 {
			t.Fatal(e, v.numBits)
		}
	} else {
		heapMust(t, e)
		if v.numBits != ^uint32(0) {
			t.Fatal(v.numBits)
		}
	}
	bitVectorRuntimeAudit(t, r, v)
	heapMust(t, v.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestBitVectorRuntimeSizeTRequestsAndReachedWrappedExpansion(t *testing.T) {
	for _, width := range []uint8{32, 64} {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: width})
		heapMust(t, e)
		want := uint64(0x100000000)
		if width == 32 {
			want = 0
		}
		if got := bitVectorRequest(r, 0x40000000); got != want {
			t.Fatal(width, got, want)
		}
		v, e := NewRuntimeBitVector(r, b)
		heapMust(t, e)
		heapMust(t, v.Push(true))
		old := v.block
		v.capacity = 0xffffffff
		// A corrupted raw capacity is a source-undefined memset range. Physical
		// reallocation/pointer publication precedes the reached suffix-write guard.
		e = v.expandTo(2)
		if !errors.Is(e, arena.ErrBounds) || v.block == old || v.capacity != 0xffffffff {
			t.Fatal(width, e, v.block == old, v.capacity)
		}
		bitVectorRuntimeAudit(t, r, v)
		heapMust(t, v.Close())
		heapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func TestBitVectorRuntimeColdSelfZeroAndClosedGuards(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	foreign := arena.NewBudget(1 << 24)
	if _, e = NewRuntimeBitVector(r, foreign); !errors.Is(e, arena.ErrConfig) || foreign.Used() != 0 {
		t.Fatal(e)
	}
	var zero RuntimeBitVector
	if e = zero.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	var fr RuntimeFastResetBitVector
	if e = fr.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	v, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	f, e := NewRuntimeFastResetBitVector(r, b)
	heapMust(t, e)
	copy := *f
	if e = copy.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal(e)
	}
	heapMust(t, v.Close())
	if _, e = v.Size(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	heapMust(t, f.Close())
	if e = f.Reset(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestBitVectorRuntimeTLSConstructorPendingActualHeaderCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	heapMust(t, e)
	heapMust(t, r.SetMaxSize(1))
	v, e := NewRuntimeBitVectorReserved(r, b, 1600000)
	if e == nil || v == nil || v.block != (Block{}) || v.pending == (Block{}) || !v.constructorFailed {
		t.Fatal("TLS allocation before caller publication", e, v)
	}
	bitVectorRuntimeAudit(t, r, v)
	if e = v.SourceFinalize(); e != v.stopped {
		t.Fatal(e, v.stopped)
	}
	heapMust(t, v.CleanupStopped())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestBitVectorRuntimeSignedRawConstructorWrappedZeroWords(t *testing.T) {
	for _, width := range []uint8{32, 64} {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: width})
		heapMust(t, e)
		v, e := NewRuntimeBitVectorSpan(r, b, RuntimeBitVectorSpan{}, -1)
		heapMust(t, e)
		if v.numBits != ^uint32(0) || v.capacity != 0 || v.block == (Block{}) {
			t.Fatal(width, v.numBits, v.capacity, v.block)
		}
		n, e := r.Size(v.block)
		heapMust(t, e)
		if n != 0 {
			t.Fatal(n)
		}
		bitVectorRuntimeAudit(t, r, v)
		heapMust(t, v.Close())
		heapMust(t, r.Close())
	}
}
