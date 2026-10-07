// Regression fixtures follow the statement order in src/util/heap.h.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntHeapSourceInsertFailurePrefix(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h := testIntHeap(t, 8, IntHeapConfig{Budget: b})
	heapMust(t, h.Insert(1)) // sentinel plus one value exhausts capacity two
	before, epoch, it := b.Used(), h.epoch, h.Begin()
	b.FailAfter(0)
	if e := h.Insert(2); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if h.Size() != 1 || !h.Contains(2) || h.indices[2] != 2 || len(h.values) != 2 || b.Used() != before {
		t.Fatal("source publishes inverse index before failing value push", h.values, h.indices, b.Used())
	}
	if !h.CheckInvariant() {
		t.Fatal("source invariant visits live values only")
	}
	if h.epoch != epoch+1 {
		t.Fatal("published membership must invalidate checked traversal")
	}
	if _, e := it.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale iterator", e)
	}
	b.FailAfter(-1)
	heapMust(t, h.Reset()) // nonempty source reset clears the whole inverse vector
	if h.Contains(2) {
		t.Fatal("nonempty reset did not clear failed publication")
	}
	heapMust(t, h.Insert(2))
	heapMust(t, h.Close())
	if b.Used() != 0 {
		t.Fatal("failure graph leaked", b.Used())
	}
}

func TestIntHeapSourceEmptyResetAndGhostGuards(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h := testIntHeap(t, 8, IntHeapConfig{Budget: b})
	heapMust(t, h.Insert(1))
	b.FailAfter(0)
	if e := h.Insert(2); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	heapMust(t, h.Erase(1)) // leaves an empty heap with the failed inverse index
	if !h.Empty() || !h.Contains(2) || h.indices[2] != 2 || !h.CheckInvariant() {
		t.Fatal("empty source failure prefix", h.indices)
	}
	before, epoch := b.Used(), h.epoch
	for _, action := range []func() error{h.Reset, h.Clear} {
		heapMust(t, action())
		if !h.Contains(2) || !h.Empty() || h.epoch != epoch || b.Used() != before {
			t.Fatal("empty reset changed prefix")
		}
	}
	for _, action := range []func() error{func() error { return h.Erase(2) }, func() error { return h.Increased(2) }, func() error { return h.Decreased(2) }} {
		if e := action(); !errors.Is(e, ErrHeapValue) {
			t.Fatal("out-of-bounds guard", e)
		}
		if !h.Contains(2) || !h.Empty() || h.epoch != epoch || b.Used() != before {
			t.Fatal("out-of-bounds guard changed prefix")
		}
	}
	h.epoch = ^uint64(0)
	heapMust(t, h.Reset())
	heapMust(t, h.Close())
	if b.Used() != 0 {
		t.Fatal("empty failure graph leaked", b.Used())
	}
}

func TestIntHeapSourceGhostInRangeContinuation(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	h := testIntHeap(t, 8, IntHeapConfig{Budget: b})
	heapMust(t, h.Insert(1))
	b.FailAfter(0)
	if e := h.Insert(2); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	heapMust(t, h.Insert(3)) // the failed index now names an actual, different value
	if !h.Contains(2) || !h.Contains(3) || h.indices[2] != 2 || h.values[2] != 3 {
		t.Fatal("source continuation", h.values, h.indices)
	}
	// Source priority notifications follow the stored index without comparing the
	// requested value with that slot. These accesses are defined, not out of bounds.
	heapMust(t, h.Increased(2))
	heapMust(t, h.Decreased(2))
	heapMust(t, h.Erase(2)) // removes slot three but clears the requested inverse index
	if h.Contains(2) || !h.Contains(3) || h.Size() != 1 || h.indices[3] != 2 || !h.CheckInvariant() {
		t.Fatal("source erase prefix", h.values, h.indices)
	}
	heapMust(t, h.Reset())
	if h.Contains(3) {
		t.Fatal("nonempty reset")
	}
}
