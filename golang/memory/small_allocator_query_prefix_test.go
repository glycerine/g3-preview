// Source: src/util/small_object_allocator.h:48; src/util/small_object_allocator.cpp:147-171.
// Licensed under the MIT license.
// Source author: Nikolaj bjorner (nbjorner) 2007-08-06.
// Source revision: Leonardo de Moura (leonardo) 2011-04-27; rewrote/simplified allocator.
package memory

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestSmallAllocatorQueriesReadPublishedPrefixDuringRawStop(t *testing.T) {
	for _, profile := range []SmallAllocatorProfile{{PointerBits: 64}, {PointerBits: 32}, {PointerBits: 64, Debug: true, Windows: true}} {
		t.Run(profileName(profile), func(t *testing.T) {
			a, d, r, b := rawAllocatorPair(t, profile)
			av := smallObjectAlloc(t, a, 17)
			free := smallObjectAlloc(t, a, 1)
			extra := smallObjectAlloc(t, a, 17)
			dv := smallObjectAlloc(t, d, 17)
			ap := rawAllocatorPointer(t, a, av)
			fp := rawAllocatorPointer(t, a, free)
			dp := rawAllocatorPointer(t, d, dv)
			smallObjectMust(t, a.Deallocate(1, free))
			smallObjectMust(t, a.Deallocate(17, extra))
			b.FailAfter(0)
			_, e := a.DeallocateRawTracked(17, dp)
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("actual sidecar refusal", e)
			}
			frame := a.rawRelease
			state := r.State()
			astats := a.Stats()
			dstats := d.Stats()
			rstats := r.Stats()
			used := b.Used()
			n, e := a.AllocationSize()
			if e != nil || n != 0 {
				t.Error("source scalar published count", n, e)
			}
			n, e = a.NumFreeObjects()
			if e != nil || n != 2 {
				t.Error("source old head actual traversal", n, e)
			}
			n, e = a.WastedSize()
			if e != nil || n != a.pointerBytes()+(17+a.pointerBytes()-1)/a.pointerBytes()*a.pointerBytes() {
				t.Error("source old head slot extent", n, e)
			}
			if a.rawRelease != frame || r.State() != state || a.Stats() != astats || d.Stats() != dstats || r.Stats() != rstats || b.Used() != used {
				t.Fatal("queries changed interrupted physical graph")
			}
			rawAllocatorGraph(t, a, d, r, b, ap.Block, fp.Block, dp.Block)
			if _, e = a.Allocate(1); !errors.Is(e, ErrAllocatorPending) {
				t.Fatal("mutation bypassed active invocation", e)
			}
			copy := *a
			if _, e = copy.AllocationSize(); !errors.Is(e, ErrAllocatorOwner) {
				t.Fatal("copied scalar owner", e)
			}
			b.FailAfter(-1)
			_, _, e = a.AbortRawInvocation()
			smallObjectMust(t, e)
			smallObjectMust(t, d.Deallocate(17, dv))
			rawAllocatorFinish(t, a, d, r, b)
		})
	}
}
