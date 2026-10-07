// Source: src/util/hashtable.h:159-215,249; src/util/debug.h:60-65,76-89.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableSourceProbeIndexIndependentLinearLoops(t *testing.T) {
	for capacity := uint32(1); capacity < 19; capacity++ {
		for start := uint32(0); start < capacity; start++ {
			var want []uint32
			for i := start; i < capacity; i++ {
				want = append(want, i)
			}
			for i := uint32(0); i < start; i++ {
				want = append(want, i)
			}
			for step, index := range want {
				if got := sourceProbeIndex(start, uint32(step), capacity); got != index {
					t.Fatal(capacity, start, step, got, index)
				}
			}
		}
	}
}
func TestStrHashtableRuntimeDefaultAssertionConstructorActualWrapperPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, InitialCapacity: 3})
	if strHashtableSourceDebug {
		if h != nil || !errors.Is(e, ErrStrHashtableAssertion) || r.State().GlobalCount != 1 {
			t.Fatal(h, e, r.State())
		}
	} else {
		mapMust(t, e)
		if h.Capacity() != 3 || r.State().GlobalCount != 2 {
			t.Fatal(h.Stats(), r.State())
		}
	}
	runtimeStringGraph(t, b, r, region, h)
	if h != nil {
		mapMust(t, h.CloseChecked())
	}
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestStrHashtableRuntimeMoveTableActualDeletedTargetPredicate(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	src, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	dst, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	var keys []StrKey
	for n := 0; len(keys) < 3; n++ {
		text := fmt.Sprintf("static-collision-%d", n)
		if String(text, 17)&7 != 0 {
			continue
		}
		keys = append(keys, strStore(t, src, text, 0))
	}
	mapMust(t, src.Insert(keys[0]))
	mapMust(t, dst.Insert(keys[1]))
	mapMust(t, dst.Insert(keys[2]))
	mapMust(t, dst.Remove(keys[1]))
	from, e := src.RuntimeRoots()
	mapMust(t, e)
	to, e := dst.RuntimeRoots()
	mapMust(t, e)
	// The real static source helper receives explicit arrays; a legitimately
	// deleted target violates its asserted precondition before the free test.
	e = dst.moveRuntimeTable(from.Table, 8, to.Table, 8)
	if strHashtableSourceDebug {
		if !errors.Is(e, ErrStrHashtableAssertion) {
			t.Fatal(e)
		}
		if dst.Deleted() != 1 || src.Size() != 1 {
			t.Fatal("stopped source fields", src.Stats(), dst.Stats())
		}
	} else {
		mapMust(t, e)
	}
	runtimeStringGraph(t, b, r, region, src, dst)
	mapMust(t, src.CloseChecked())
	mapMust(t, dst.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
