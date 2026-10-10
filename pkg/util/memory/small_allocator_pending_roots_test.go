// Source: src/util/small_object_allocator.cpp; src/util/memory_manager.cpp.
package memory

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSmallAllocatorActualPendingRootsIdentityOnly(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	smallObjectMust(t, e)
	a, e := NewSmallAllocator(SmallAllocatorConfig{Runtime: r, Profile: SmallAllocatorProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	smallObjectMust(t, r.SetMaxSize(1))
	ref, cause := a.Allocate(100000)
	var memout Failure
	if ref != (ObjectRef{}) || !errors.As(cause, &memout) || memout.ErrorCode() != 101 {
		t.Fatal("actual postphysical allocation refusal", ref, cause)
	}
	state := r.State()
	stats := r.Stats()
	used := b.Used()
	roots, e := a.PendingRoots()
	smallObjectMust(t, e)
	if roots[0] == (Block{}) || roots[1] != (Block{}) || roots[2] != (Block{}) || a.Stats().PendingBlocks != 1 {
		t.Fatal("actual three named allocation/lexical roots", roots, a.Stats())
	}
	size, e := r.Size(roots[0])
	smallObjectMust(t, e)
	if size != 100000 || r.State() != state || r.Stats() != stats || b.Used() != used {
		t.Fatal("diagnostic mutated owner/source state", size)
	}
	copyOwner := *a
	if _, e = copyOwner.PendingRoots(); e != ErrAllocatorOwner {
		t.Fatal("copied owner root exposure", e)
	}
	var nilOwner *SmallAllocator
	if _, e = nilOwner.PendingRoots(); e != ErrAllocatorOwner {
		t.Fatal("nil owner root exposure", e)
	}
	smallObjectMust(t, r.SetMaxSize(0))
	smallObjectMust(t, a.CleanupPending())
	roots, e = a.PendingRoots()
	smallObjectMust(t, e)
	if roots != ([3]Block{}) || r.Stats().LiveBlocks != 0 {
		t.Fatal("actual consumed physical root not acknowledged", roots, r.Stats())
	}
	smallObjectMust(t, a.Close())
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual whole graph refund", b.Used())
	}
}
