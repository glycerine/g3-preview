// Source: src/util/heap.h:27-28; src/util/vector.h:102-129,191-202.
package heap

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHeapAssignmentSourceDestroyedFirstMember(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	src := testInt(t, 16, Config{Budget: b, Less: func(a, c int32) bool { return a > c }})
	dst := testInt(t, 4, Config{Budget: b})
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
