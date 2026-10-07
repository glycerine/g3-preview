// Source: src/util/hashtable.h:222-230; src/util/memory_manager.cpp:269-285.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableRuntimeAllocationReentrantOwningClose(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var h *StrHashtable
	active := false
	callbacks := 0
	var closeCause, sourceCause error
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, Transport: func(f Failure) error {
		if active {
			callbacks++
			closeCause = h.CloseChecked()
			sourceCause = h.SourceFinalize()
		}
		return f
	}})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 2048})
	mapMust(t, e)
	mapMust(t, r.SetMaxSize(1))
	active = true
	cause := h.rehashRuntime(4096)
	active = false
	if cause == nil || callbacks != 1 {
		t.Fatal("actual postphysical transport not reached", cause, callbacks)
	}
	if closeCause != ErrStrHashtableExecuting || sourceCause != ErrStrHashtableExecuting {
		t.Fatal("destructive callback did not guard executing actual leaf", closeCause, sourceCause)
	}
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	named := uint32(len(roots.Acquisitions))
	if roots.Object != (Block{}) {
		named++
	}
	if roots.Table != (Block{}) {
		named++
	}
	if uint32(r.Stats().LiveBlocks) != named {
		t.Fatalf("reentrant close lost actual returned acquisition: live=%d named=%d roots=%+v close=%v body=%v", r.Stats().LiveBlocks, named, roots, closeCause, cause)
	}
	mapMust(t, r.SetMaxSize(0))
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("named graph retained", b.Used())
	}
}

func TestStrHashtableRuntimeLexicalCopyReentrantClosePreservesReturnedOwner(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var h *StrHashtable
	active := false
	calls := 0
	var own, source error
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, Transport: func(f Failure) error {
		if active {
			calls++
			own = h.CloseChecked()
			source = h.SourceFinalize()
		}
		return f
	}})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 2048})
	mapMust(t, e)
	other, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	mapMust(t, r.SetMaxSize(1))
	active = true
	cause := h.Intersect(other)
	active = false
	if cause == nil || calls != 1 {
		t.Fatal("actual lexical copy allocation transport", cause, calls)
	}
	runtimeStringGraph(t, b, r, region, h, other)
	if own != ErrStrHashtableExecuting || source != ErrStrHashtableExecuting {
		t.Fatal("executing lexical context was destructively retired", own, source)
	}
	runtimeStringGraph(t, b, r, region, h, other)
	mapMust(t, r.SetMaxSize(0))
	mapMust(t, h.CloseChecked())
	mapMust(t, other.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
