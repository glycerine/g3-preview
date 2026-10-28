package heap

// Source: src/util/heap.h:27-28; native typed storage for the live int heap.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHeapNativeAssignmentAcrossBudgets(t *testing.T) {
	sourceBudget := arena.NewBudget(ownerBytes)
	destBudget := arena.NewBudget(ownerBytes)
	src := testInt(t, 16, Config{Budget: sourceBudget, Less: func(a, b int32) bool { return a > b }})
	dst := testInt(t, 4, Config{Budget: destBudget})
	heapMust(t, src.Insert(3))
	heapMust(t, dst.Insert(1))
	before := destBudget.Used()
	destBudget.FailAfter(0)
	heapMust(t, dst.Assign(src))
	if destBudget.Used() != before || !dst.Contains(3) || !dst.lt(3, 1) || !src.Contains(3) {
		t.Fatal("native assignment or borrowed comparator", destBudget.Used(), before)
	}
	got, e := dst.EraseMin()
	heapMust(t, e)
	if got != 3 || !src.Contains(3) {
		t.Fatal("copied value independence", got)
	}
	destBudget.FailAfter(-1)
}
