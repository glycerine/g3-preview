// Source: src/util/heap.h:123-135,152-166,195-218; src/util/vector.h:420-433.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-14.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntHeapEmptySourceZeroSizeSentinelPop(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	e = h.Erase(0)
	n, read := h.runtimeQueue.runtimeValues.size()
	heapMust(t, read)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || n != 1 || !h.Empty() {
			t.Fatal("source pre-mutation contains assertion", e, n)
		}
	} else {
		heapMust(t, e)
		if n != 0 || h.Empty() {
			t.Fatal("source empty must compare actual SIZE with one", n, h.Empty())
		}
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{h})
	heapMust(t, h.Close())
	heapMust(t, r.Close())
}
func TestIntHeapSetBoundsSignedConversionBeforeGrowthRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	old := h.runtimeQueue.runtimeIndices.raw.block
	size, e := h.runtimeQueue.runtimeIndices.size()
	heapMust(t, e)
	capacity, e := h.runtimeQueue.runtimeIndices.capacity()
	heapMust(t, e)
	b.FailAfter(0)
	// Reserve's signed comparison is distinct from SetBounds's unsigned resize.
	heapMust(t, h.Reserve(-1))
	e = h.SetBounds(-1)
	var failure memory.Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !r.State().OutOfMemory {
		t.Fatal("first actual resize expansion refusal", e, r.State())
	}
	n, read := h.runtimeQueue.runtimeIndices.size()
	heapMust(t, read)
	cap, read := h.runtimeQueue.runtimeIndices.capacity()
	heapMust(t, read)
	if h.runtimeQueue.runtimeIndices.raw.block != old || n != size || cap != capacity {
		t.Fatal("source SIZE/capacity prefix on first refusal", n, cap, size, capacity)
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{h})
	b.FailAfter(-1)
	heapMust(t, h.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("bounded refusal leaked owners")
	}
}
func TestIntHeapConstructorSignedBoundsReachedAfterSentinel(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	// Reflected heap header, descriptor directory and the real combined
	// sentinel payload/bitmap reservation complete before index growth refuses.
	b.FailAfter(3)
	h, e := NewIntHeap(-1, IntHeapConfig{Budget: b, Runtime: r})
	var failure memory.Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !r.State().OutOfMemory {
		t.Fatal("source ctor did not reach sentinel then bound growth", e, r.State())
	}
	if h != nil || r.State().GlobalCount != 2 || r.State().GlobalSize != 24 || r.Stats().LiveBlocks != 0 || b.Used() != r.Stats().TotalBytes {
		t.Fatal("completed ctor failure/unwind prefix", h, r.State(), r.Stats(), b.Used())
	}
	b.FailAfter(-1)
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("ctor prefix ownership")
	}
}
