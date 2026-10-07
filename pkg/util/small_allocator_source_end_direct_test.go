// Source: src/util/small_object_allocator.cpp:72-96.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSmallAllocatorDirectSourceEndWithForeignPending(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	smallObjectMust(t, e)
	object, e := a.Allocate(32)
	smallObjectMust(t, e)
	raw, _, e := a.RawLocation(object)
	smallObjectMust(t, e)
	padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	smallObjectMust(t, e)
	smallObjectMust(t, r.SetMaxSize(1))
	_, body := a.Allocate(32)
	if body == nil || a.pending == (Block{}) {
		t.Fatal(body)
	}
	pending := a.pending
	smallObjectMust(t, r.SetMaxSize(0))
	if e = a.Close(); !errors.Is(e, ErrAllocatorPending) || !a.destructorDone || a.closed {
		t.Fatal(e)
	}
	ended, e := a.SourceObjectOwnerEnded()
	smallObjectMust(t, e)
	if ended {
		t.Fatal("pending source destructor confused with directory end")
	}
	before := r.Stats().LiveBlocks
	handled, e := a.CleanupSourceRetainedDirect(object, &raw)
	smallObjectMust(t, e)
	if !handled || raw != (Block{}) || a.pending != pending || r.Stats().LiveBlocks != before-1 {
		t.Fatal("exact direct retirement touched foreign pending", handled, raw, a.pending)
	}
	if _, e = a.object(object); !errors.Is(e, ErrAllocatorOwner) {
		t.Fatal("directory retained released object", e)
	}
	smallObjectMust(t, a.CleanupPending())
	smallObjectMust(t, a.Close())
	smallObjectMust(t, r.Deallocate(&padding))
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
