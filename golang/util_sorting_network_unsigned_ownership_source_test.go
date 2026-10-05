// Source: src/util/sorting_network.h:28-115; src/util/vector.h:178-189,303-329,350-365,437-440.
// Source author: Nikolaj Bjorner (2013-11-07); Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func unsignedSortingGraph(t *testing.T, b *arena.Budget, r *memory.Runtime, sn *RuntimeUnsignedSortingNetwork, borrowed ...*RuntimeUnsignedVector) {
	t.Helper()
	vectors := append([]*RuntimeUnsignedVector{&sn.currentv, &sn.nextv}, borrowed...)
	roots := map[memory.Block]bool{}
	cold := sn.header
	seen := map[*RuntimeUnsignedVector]bool{}
	add := func(block memory.Block) {
		if block == (memory.Block{}) {
			return
		}
		if roots[block] {
			t.Fatal("duplicate actual vector root", block)
		}
		roots[block] = true
		_, e := r.Size(block)
		paramsMust(t, e)
	}
	for _, v := range vectors {
		if seen[v] {
			t.Fatal("duplicate actual column")
		}
		seen[v] = true
		cold += v.header
		add(v.column.raw.block)
		add(v.column.raw.pending)
		for a := v.growthAcquisitions; a != nil; a = a.next {
			cold += a.header
			add(a.block)
		}
		for a := v.copyAcquisitions; a != nil; a = a.next {
			cold += a.header
			add(a.block)
		}
	}
	stats := r.Stats()
	if uint32(len(roots)) != stats.LiveBlocks {
		t.Fatal("named source vector/retained token bijection", roots, stats)
	}
	if b.Used() != stats.TotalBytes+cold {
		t.Fatal("named raw/cold graph", b.Used(), stats.TotalBytes+cold)
	}
}
func TestUnsignedSortingNetworkEveryActualReservationAndCallerPrefix(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		first := -1
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: bits})
			paramsMust(t, e)
			sn, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
			paramsMust(t, e)
			in, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			out, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			values := []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1}
			for _, v := range values {
				paramsMust(t, in.Push(v))
			}
			b.FailAfter(ordinal)
			cause := sn.Sort(in, out)
			unsignedSortingGraph(t, b, r, sn, in, out)
			a, e := sn.currentv.Size()
			paramsMust(t, e)
			n, e := sn.nextv.Size()
			paramsMust(t, e)
			if a < n || a > n+1 {
				t.Fatal("source index alternating push prefix", a, n)
			}
			size, e := out.Size()
			paramsMust(t, e)
			if a != 0 && size != 16 {
				t.Fatal("indices start only after complete caller padding", a, n, size)
			}
			if a == 0 && size <= uint32(len(values)) {
				for i := uint32(0); i < size; i++ {
					x, e := out.At(i)
					paramsMust(t, e)
					if x != values[i] {
						t.Fatal("caller copied prefix", ordinal, i, x)
					}
				}
			}
			if cause == nil {
				sortingUnsignedOracle(t, out, values)
			}
			b.FailAfter(-1)
			paramsMust(t, sn.SourceFinalize())
			unsignedSortingGraph(t, b, r, sn, in, out)
			paramsMust(t, sn.Close())
			size, e = in.Size()
			paramsMust(t, e)
			if size != uint32(len(values)) {
				t.Fatal("borrowed source was not owned by network")
			}
			paramsMust(t, out.Close())
			paramsMust(t, in.Close())
			paramsMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(bits, ordinal, b.Used())
			}
			if cause == nil {
				first = int(ordinal)
				break
			}
		}
		if first != 19 {
			t.Fatal("missing first actual success", bits)
		}
		t.Log("actual sorting reservation firstsuccess", bits, first)
	}
}
func TestUnsignedSortingNetworkConstructorIdentityAndColdRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	before := b.Used()
	b.FailAfter(0)
	sn, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	if e == nil || sn != nil || b.Used() != before {
		t.Fatal("unacquired header refusal", sn, e)
	}
	b.FailAfter(-1)
	sn, e = NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	copy := *sn
	used := b.Used()
	if copy.Close() != ErrSATOwner || copy.SourceFinalize() != ErrSATOwner || b.Used() != used {
		t.Fatal("accidental owning Go copy")
	}
	paramsMust(t, sn.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestUnsignedSortingNetworkStoppedReverseDestructorActualReceipt(t *testing.T) {
	// Source signed-overflow free domain. Populate real constructed member
	// vectors through their public storage engine; no fabricated counters/fields.
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	sn, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	paramsMust(t, sn.currentv.Resize(30000, 0))
	paramsMust(t, sn.nextv.Resize(30000, 0))
	satHeapTLSNearMinimum(t, r)
	before := sn.currentv.column.raw.block
	cause := sn.SourceFinalize()
	if !errors.Is(cause, arena.ErrBounds) || sn.nextv.column.raw.block != (memory.Block{}) || sn.currentv.column.raw.block != before || sn.destructorStage != 0 {
		t.Fatal("next before current true DT stop", cause, sn.destructorStage)
	}
	unsignedSortingGraph(t, b, r, sn)
	if sn.Close() != cause {
		t.Fatal("source stop sticky")
	}
	first := sn.CleanupStopped()
	if !errors.Is(first, arena.ErrBounds) {
		t.Fatal("older current true consumed stop", first)
	}
	unsignedSortingGraph(t, b, r, sn)
	if sn.currentv.column.raw.block != (memory.Block{}) || sn.closed {
		t.Fatal("consumed root receipt before headerrefund")
	}
	paramsMust(t, sn.CleanupStopped())
	if sn.destructorCause != cause || sn.destructorStage != 0 {
		t.Fatal("owning cleanup rewrote source history")
	}
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedSortingNetworkActualAllocationTransportCannotCloseExecutingOwner(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var sn *RuntimeUnsignedSortingNetwork
	active := false
	calls := 0
	var ordinary, source error
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, Transport: func(f memory.Failure) error {
		if active {
			calls++
			ordinary = sn.Close()
			source = sn.SourceFinalize()
		}
		return f
	}})
	paramsMust(t, e)
	sn, e = NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	in, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	out, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, in.Resize(32768, 0))
	paramsMust(t, out.Resize(32768, 0))
	paramsMust(t, r.SetMaxSize(1))
	active = true
	cause := sn.Sort(in, out)
	active = false
	if cause == nil || calls != 1 {
		t.Fatal("actual internal vector postphysical OOM", cause, calls)
	}
	if ordinary != ErrUnsignedSortingNetworkExecuting || source != ErrUnsignedSortingNetworkExecuting || sn.closed || sn.header == 0 {
		t.Fatal("executing acquisition owner was closed before returned root", ordinary, source, sn.closed, sn.header)
	}
	unsignedSortingGraph(t, b, r, sn, in, out)
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, sn.Close())
	paramsMust(t, out.Close())
	paramsMust(t, in.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
