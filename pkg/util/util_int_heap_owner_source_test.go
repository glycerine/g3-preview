// Source: src/util/heap.h:27-28; src/util/vector.h:102-129,191-202.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-14.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntHeapLegacyCopiedCloseRejectsBeforeRefund(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	h, e := NewIntHeap(4, IntHeapConfig{Budget: b})
	heapMust(t, e)
	copied := *h
	before := b.Used()
	if e = copied.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal("copied cold owner Close admitted", e)
	}
	if b.Used() != before {
		t.Fatal("copy guard refunded actual original owner")
	}
	heapMust(t, h.Close())
	if b.Used() != 0 {
		t.Fatal("original owner not refunded")
	}
}
func TestIntHeapSourceScalarPlacementReadsWholeIntBeforeWrite(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	src, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	dst, e := NewIntHeap(4, IntHeapConfig{Budget: b, Runtime: r})
	heapMust(t, e)
	// Checked source-undefined domain: only one byte of the second source int
	// is initialized. The actual T(*it) scalar read must complete before the
	// destination placement store; this is not ordinary numeric OOM evidence.
	heapMust(t, r.Write(src.runtimeQueue.runtimeValues.Raw.Block, 12, 42))
	heapMust(t, r.WriteUint32(src.runtimeQueue.runtimeValues.Raw.Block, 4, 2))
	e = dst.Assign(src)
	if !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal("actual incomplete scalar read", e)
	}
	intHeapRuntimeAudit(t, r, []*IntHeap{src, dst})
	if _, e = r.Read(dst.runtimeQueue.runtimeValues.Raw.Block, 12); !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal("destination int written before source scalar read completed", e)
	}
	heapMust(t, dst.CleanupStopped())
	heapMust(t, src.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("partial publication owner lost")
	}
}
