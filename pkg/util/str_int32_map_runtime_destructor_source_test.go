// Source: src/util/hashtable.h:146-148,259-261,287-322;
// src/util/memory_manager.h:112-119; src/util/memory_manager.cpp:274-325.
package util

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrInt32MapRuntimeNullReplacementDestructorRetainsUnjoinedNew(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, InitialCapacity: 8192})
	mapMust(t, e)
	mapMust(t, h.Insert(simKey(t, region, "live"), -117))
	before, e := h.RuntimeRoots()
	mapMust(t, e)
	mapMust(t, r.SetMaxSize(1))
	cause := h.Reset()
	if cause == nil || !r.State().OutOfMemory {
		t.Fatal("actual postphysical replacement refusal", cause, r.State())
	}
	after, e := h.RuntimeRoots()
	mapMust(t, e)
	if after.Table != (Block{}) || after.Object != before.Object || len(after.Acquisitions) != 1 {
		t.Fatal("NULL current, real unjoined NEW", after)
	}
	runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
	pending := after.Acquisitions[0]
	// DT skips actual NULL; pending NEW is not a source completed member.
	mapMust(t, h.SourceFinalize())
	after, e = h.RuntimeRoots()
	mapMust(t, e)
	if after.Object != (Block{}) || after.Table != (Block{}) || len(after.Acquisitions) != 1 || after.Acquisitions[0] != pending || h.raw.sourceStage != 2 {
		t.Fatal("core DT vs separate owning root", after, h.raw.sourceStage)
	}
	runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
	mapMust(t, h.SourceFinalize())
	if _, e = h.ContainsFrom(region, StrKey{}); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ended source lifetime guard", e)
	}
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrInt32MapRuntimeBodyReleaseStopReachesOwnDestructorBoundary(t *testing.T) {
	// Actual signed-overflow-domain Runtime free physically consumes OLD, then
	// refuses synchronization. Source body cannot publish NULL; later native DT
	// would dereference the dangling source pointer (undefined domain). Its Go
	// reached pointer guard must be distinct from the earlier body cause.
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, InitialCapacity: 8192})
	mapMust(t, e)
	first, e := r.Allocate(8)
	mapMust(t, e)
	second, e := r.Allocate(8)
	mapMust(t, e)
	delta := int64(-1000000)
	if e = r.Reallocate(&first, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	mapMust(t, r.Deallocate(&first))
	state := r.State()
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	if e = r.Reallocate(&second, uint64(target)+16-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	mapMust(t, r.Deallocate(&second))
	body := h.Finalize()
	if !errors.Is(body, arena.ErrBounds) || h.raw.sourceCause != body || h.raw.table != (Block{}) {
		t.Fatal("actual body release stop", body, h.raw)
	}
	runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
	dt := h.SourceFinalize()
	if !errors.Is(dt, ErrRuntimeOwner) || dt == body || !h.raw.destructing || h.raw.sourceStage != 0 {
		t.Fatal("DT must reach stale source pointer, not replay body cause", dt, body, h.raw.sourceStage)
	}
	if h.raw.sourceCause != body {
		t.Fatal("body history replaced", h.raw.sourceCause)
	}
	if again := h.SourceFinalize(); again != dt {
		t.Fatal("actual DT cause sticky", again, dt)
	}
	runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
	// Exact zero token records consumed OLD. Owning abort frees remaining wrapper,
	// whose own postphysical sync may stop; retry acknowledges its zero receipt.
	e = h.CloseChecked()
	if e != nil && !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
	mapMust(t, h.CloseChecked())
	if h.raw.sourceCause != body || h.raw.destructorCause != dt || h.SourceFinalize() != dt {
		t.Fatal("owning cleanup altered source stop history")
	}
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrInt32MapRuntimeReplacementEveryRefusalThenSourceDestructor(t *testing.T) {
	for _, finalize := range []bool{false, true} {
		success := -1
		for ordinal := int64(0); ordinal < 16; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, InitialCapacity: 128})
			mapMust(t, e)
			mapMust(t, h.Insert(simKey(t, region, "existing"), -313))
			b.FailAfter(ordinal)
			if finalize {
				e = h.Finalize()
			} else {
				e = h.Reset()
			}
			body := e
			runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
			if e != nil && h.raw.table != (Block{}) {
				t.Fatal("replacement refused after actual NULL publication", ordinal, e)
			}
			// No reservation is necessary for real source table/object destruction.
			mapMust(t, h.SourceFinalize())
			roots, e := h.RuntimeRoots()
			mapMust(t, e)
			if roots.Object != (Block{}) || roots.Table != (Block{}) || h.raw.sourceStage != 2 {
				t.Fatal("actual DT complete", ordinal, roots)
			}
			runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h)
			mapMust(t, h.CloseChecked())
			b.FailAfter(-1)
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(ordinal, b.Used())
			}
			if body == nil {
				success = int(ordinal)
				break
			}
		}
		if success < 0 {
			t.Fatal("first success missing", finalize)
		}
		t.Log("actual replacement first success", finalize, success)
	}
}
