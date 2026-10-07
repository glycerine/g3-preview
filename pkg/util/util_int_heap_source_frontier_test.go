// Source: src/util/heap.h:27-28,272-285; src/util/vector.h:102-129,191-202.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntHeapAssignmentSourceDestroyedFirstMember(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	src := testIntHeap(t, 16, IntHeapConfig{Budget: b, Less: func(a, c int32) bool { return a > c }})
	dst := testIntHeap(t, 4, IntHeapConfig{Budget: b})
	heapMust(t, src.Insert(3))
	heapMust(t, dst.Insert(1))
	before := b.Used()
	oldValues := uint64(cap(dst.values)) * 4
	b.FailAfter(0)
	e := dst.Assign(src)
	// The old value vector is destroyed before copy_core's allocation. The
	// later inverse vector is still source-owned. No temporary heap exists.
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("allocation refusal: %v", e)
	}
	if got := b.Used(); got != before-oldValues {
		t.Fatalf("source first destruction frontier: used=%d want=%d (old values=%d)", got, before-oldValues, oldValues)
	}
	if !dst.lt(3, 1) {
		t.Fatal("base comparator must be assigned before vector destruction")
	}
	b.FailAfter(-1)
	heapMust(t, dst.CleanupStopped())
}

func TestIntHeapFindLESourceOutputBeforeChildGrowth(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h := testIntHeap(t, 16, IntHeapConfig{Budget: b})
	for i := int32(1); i <= 7; i++ {
		heapMust(t, h.Insert(i))
	}
	dst := make([]int32, 1, 16)
	dst[0] = 99
	before := b.Used()
	b.FailAfter(1)
	got, e := h.FindLE(7, dst)
	// Fresh todo capacity two accepts root's left/right. Processing right=3
	// publishes value3, then pushes left6 before right7 requires capacity3.
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("child growth refusal: %v", e)
	}
	if !reflect.DeepEqual(got, []int32{99, 1, 3}) {
		t.Fatalf("source immediate output prefix: %v", got)
	}
	if b.Used() != before {
		t.Fatalf("lexical todo did not destruct: %d -> %d", before, b.Used())
	}
	b.FailAfter(-1)
}
