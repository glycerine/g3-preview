// Source: src/util/hashtable.h:33-36,138-140,254-256,267-269,281-283,359-373,409-424,481-491,533-543,649-653.
package util

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableRuntimeStatisticsActualFieldAndLiteralMissCounts(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, Statistics: true})
	mapMust(t, e)
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	extent, e := r.Size(roots.Object)
	mapMust(t, e)
	if extent != 32 || h.CollisionCount() != 0 {
		t.Fatal(extent, h.CollisionCount())
	}
	var keys []StrKey
	for n := 0; len(keys) < 3; n++ {
		text := fmt.Sprintf("count-collision-%d", n)
		if String(text, 17)&7 != 0 {
			continue
		}
		keys = append(keys, strStore(t, h, text, 0))
	}
	mapMust(t, h.Insert(keys[0]))
	mapMust(t, h.Insert(keys[1]))
	if h.CollisionCount() != 1 {
		t.Fatal("insert USED miss once", h.CollisionCount())
	}
	found, e := h.Contains(keys[2])
	mapMust(t, e)
	if found || h.CollisionCount() != 5 {
		t.Fatal("source FIND each USED miss twice", found, h.CollisionCount())
	}
	mapMust(t, h.Remove(keys[0]))
	if h.CollisionCount() != 5 {
		t.Fatal("matched remove zero", h.CollisionCount())
	}
	found, e = h.Contains(keys[2])
	mapMust(t, e)
	if found || h.CollisionCount() != 8 {
		t.Fatal("deleted1 + used2", h.CollisionCount())
	}
	mapMust(t, h.Remove(keys[2]))
	if h.CollisionCount() != 11 {
		t.Fatal("REMOVE deleted1 + USED2", h.CollisionCount())
	}
	copy, e := h.Clone()
	mapMust(t, e)
	if copy.CollisionCount() != 0 {
		t.Fatal("copy constructor resets", copy.CollisionCount())
	}
	mapMust(t, h.Swap(copy))
	if h.CollisionCount() != 0 || copy.CollisionCount() != 11 {
		t.Fatal("actual member swap", h.CollisionCount(), copy.CollisionCount())
	}
	runtimeStringGraph(t, b, r, region, h, copy)
	mapMust(t, h.CloseChecked())
	mapMust(t, copy.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeStatisticsEveryConstructorAndCopyReservation(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, copying := range []bool{false, true} {
			success := -1
			for ordinal := int64(0); ordinal < 32; ordinal++ {
				b := arena.NewBudget(1 << 26)
				r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: bits})
				mapMust(t, e)
				region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
				mapMust(t, e)
				var src *StrHashtable
				if copying {
					src, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, Statistics: true})
					mapMust(t, e)
					mapMust(t, src.Insert(strStore(t, src, "copy", 0)))
					mapMust(t, src.writeCollision(19))
				}
				b.FailAfter(ordinal)
				var out *StrHashtable
				if copying {
					out, e = src.Clone()
				} else {
					out, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, Statistics: true})
				}
				runtimeStringGraph(t, b, r, region, src, out)
				if e == nil {
					success = int(ordinal)
					if out.CollisionCount() != 0 {
						t.Fatal("constructor counter init", out.CollisionCount())
					}
					rr, re := out.RuntimeRoots()
					mapMust(t, re)
					n, re := r.Size(rr.Object)
					mapMust(t, re)
					if n != 32 {
						t.Fatal(n)
					}
				}
				b.FailAfter(-1)
				if out != nil {
					mapMust(t, out.CloseChecked())
				}
				if src != nil {
					mapMust(t, src.CloseChecked())
				}
				mapMust(t, region.Close())
				mapMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal(bits, copying, ordinal, b.Used())
				}
				if e == nil {
					break
				}
			}
			if success != 6 {
				t.Fatal("actual constructor/copy horizon", bits, copying, success)
			}
		}
	}
}

func TestStrHashtableRuntimeStatisticsResetFinalizeInlineAndUnsignedWrap(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, Statistics: true, InitialCapacity: 128})
	mapMust(t, e)
	// Reach the actual field through its physical word codec, not a cold mirror.
	mapMust(t, h.writeCollision(^uint64(0)))
	mapMust(t, h.incrementCollision())
	if h.CollisionCount() != 0 {
		t.Fatal("unsigned wrap")
	}
	mapMust(t, h.writeCollision(37))
	mapMust(t, h.Reset())
	if h.CollisionCount() != 37 {
		t.Fatal("reset changed source counter")
	}
	mapMust(t, h.Finalize())
	if h.CollisionCount() != 37 {
		t.Fatal("finalize changed source counter")
	}
	inline, e := h.cloneRuntime(false)
	mapMust(t, e)
	rr, e := inline.RuntimeRoots()
	mapMust(t, e)
	if rr.Object != (Block{}) || inline.CollisionCount() != 0 {
		t.Fatal("literal lexical inline counter", rr)
	}
	mapMust(t, inline.writeCollision(41))
	mapMust(t, inline.incrementCollision())
	if inline.CollisionCount() != 42 || h.CollisionCount() != 37 {
		t.Fatal("separate authoritative fields")
	}
	runtimeStringGraph(t, b, r, region, h, inline)
	mapMust(t, inline.CloseChecked())
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableRuntimeStatisticsEveryReplacementAndGrowthPrefix(t *testing.T) {
	for _, action := range []string{"reset", "finalize", "grow"} {
		first := -1
		for ordinal := int64(0); ordinal < 8; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			capacity := uint32(128)
			if action == "grow" {
				capacity = 8
			}
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, Statistics: true, InitialCapacity: capacity})
			mapMust(t, e)
			var key StrKey
			if action == "grow" {
				for i := 0; i < 7; i++ {
					mapMust(t, h.Insert(strStore(t, h, fmt.Sprintf("grow-%d", i), 0)))
				}
				key = strStore(t, h, "grow-next", 0)
			} else {
				mapMust(t, h.Insert(strStore(t, h, "one", 0)))
			}
			mapMust(t, h.writeCollision(29))
			before, e := h.RuntimeRoots()
			mapMust(t, e)
			b.FailAfter(ordinal)
			switch action {
			case "reset":
				e = h.Reset()
			case "finalize":
				e = h.Finalize()
			case "grow":
				e = h.Insert(key)
			}
			runtimeStringGraph(t, b, r, region, h)
			if e != nil && h.CollisionCount() != 29 {
				t.Fatal("getter must read actual counter despite stopped body", action, e, h.CollisionCount())
			}
			if action != "grow" && h.CollisionCount() != 29 {
				t.Fatal("replacement must retain statistics", action, h.CollisionCount())
			}
			if action == "grow" && e != nil {
				after, re := h.RuntimeRoots()
				mapMust(t, re)
				if after.Table != before.Table {
					t.Fatal("old field before growth publication", after, before)
				}
			}
			if e == nil {
				first = int(ordinal)
			}
			b.FailAfter(-1)
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(action, ordinal, b.Used())
			}
			if e == nil {
				break
			}
		}
		want := 2
		if action == "grow" {
			want = 3
		}
		if first != want {
			t.Fatal("actual replacement horizon", action, first, want)
		}
	}
}

func TestStrHashtableRuntimeStatisticsIndependentFiniteProbeCounts(t *testing.T) {
	// Three occupied slots in a known physical cluster give a finite independent
	// miss oracle: absent FIND/REMOVE contributes two for each occupied slot,
	// one for each deleted slot, and zero for the first free slot.
	for mask := 0; mask < 8; mask++ {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		mapMust(t, e)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r, Statistics: true})
		mapMust(t, e)
		keys := collisionStrKeys(t, h, 0, 7, 4)
		for _, k := range keys[:3] {
			mapMust(t, h.Insert(k))
		}
		deleted := uint64(0)
		for i := 0; i < 3; i++ {
			if mask&(1<<i) != 0 {
				mapMust(t, h.Remove(keys[i]))
				deleted++
			}
		}
		before := h.CollisionCount()
		want := uint64(6) - deleted
		// The last cluster entry is followed by a free slot: REMOVE marks it
		// free rather than deleted, ending the subsequent search there.
		if mask&4 != 0 {
			want--
		}
		_, ok, e := h.FindCStringBytes([]byte(mustStatisticsText(t, h, keys[3])))
		mapMust(t, e)
		if ok || h.CollisionCount()-before != want {
			t.Fatal(mask, "byte FIND", ok, h.CollisionCount()-before, want)
		}
		before = h.CollisionCount()
		_, ok, e = h.FindCString(mustStatisticsText(t, h, keys[3]))
		mapMust(t, e)
		if ok || h.CollisionCount()-before != want {
			t.Fatal(mask, "string FIND", ok, h.CollisionCount()-before, want)
		}
		before = h.CollisionCount()
		mapMust(t, h.Remove(keys[3]))
		if h.CollisionCount()-before != want {
			t.Fatal(mask, "REMOVE", h.CollisionCount()-before, want)
		}
		runtimeStringGraph(t, b, r, region, h)
		mapMust(t, h.CloseChecked())
		mapMust(t, region.Close())
		mapMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func mustStatisticsText(t *testing.T, h *StrHashtable, k StrKey) string {
	t.Helper()
	s, e := h.Text(k)
	mapMust(t, e)
	return s
}
