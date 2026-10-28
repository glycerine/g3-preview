package heap

// Native storage checks for the retained integer heap.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHeapNativeInsertUnderRefusedArena(t *testing.T) {
	b := arena.NewBudget(ownerBytes)
	h := testInt(t, 130, Config{Budget: b})
	before, stale := b.Used(), h.Begin()
	b.FailAfter(0)
	for i := int32(0); i < 129; i++ {
		heapMust(t, h.Insert(i))
	}
	if h.Size() != 129 || b.Used() != before || !h.CheckInvariant() {
		t.Fatal("native growth", h.Size(), b.Used(), before)
	}
	if _, e := stale.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale iterator", e)
	}
	for i := int32(0); i < 129; i++ {
		got, e := h.EraseMin()
		heapMust(t, e)
		if got != i {
			t.Fatal("heap order", got, i)
		}
	}
	b.FailAfter(-1)
	heapMust(t, h.Close())
	if b.Used() != 0 {
		t.Fatal("owner leak", b.Used())
	}
}

func TestHeapNativeEmptyResetAndNoGhost(t *testing.T) {
	b := arena.NewBudget(ownerBytes)
	h := testInt(t, 8, Config{Budget: b})
	heapMust(t, h.Insert(1))
	_, e := h.EraseMin()
	heapMust(t, e)
	before := h.epoch
	heapMust(t, h.Reset())
	if h.epoch != before || h.Contains(1) || h.Contains(2) {
		t.Fatal("empty reset or stale membership")
	}
	b.FailAfter(0)
	heapMust(t, h.Insert(2))
	if !h.Contains(2) || h.Size() != 1 || b.Used() != ownerBytes {
		t.Fatal("native insertion", h.values, h.indices)
	}
	b.FailAfter(-1)
}

func TestHeapNativePriorityUpdates(t *testing.T) {
	priority := []int32{30, 20, 10}
	h := testInt(t, 3, Config{Less: func(a, b int32) bool { return priority[a] < priority[b] }})
	for _, v := range []int32{0, 1, 2} {
		heapMust(t, h.Insert(v))
	}
	priority[0] = 1
	heapMust(t, h.Decreased(0))
	if got, e := h.Min(); e != nil || got != 0 {
		t.Fatal(got, e)
	}
	priority[0] = 40
	heapMust(t, h.Increased(0))
	if got, e := h.Min(); e != nil || got != 2 {
		t.Fatal(got, e)
	}
}
