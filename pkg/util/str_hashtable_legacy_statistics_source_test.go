// Source: src/util/hashtable.h:138-140,254-256,267-269,281-283,359-373,409-424,481-491,533-543,649-653.
package util

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableLegacyStatisticsActualScalarAndLiteralMissCounts(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
	mapMust(t, e)
	if h.CollisionCount() != 0 || h.raw.initialized&0xC0 != 0xC0 || h.raw.object.Slot() != 0 || h.table == nil {
		t.Fatal("actual charged scalar, no Runtime ticket", h.raw)
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
	legacyStatisticsGraph(t, b, region, h, copy)
	mapMust(t, h.CloseChecked())
	mapMust(t, copy.CloseChecked())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableLegacyStatisticsIndependentFiniteProbeCounts(t *testing.T) {
	// Three occupied slots in a known physical cluster give a finite independent
	// miss oracle: absent FIND/REMOVE contributes two for each occupied slot,
	// one for each deleted slot, and zero for the first free slot.
	for mask := 0; mask < 8; mask++ {
		b := arena.NewBudget(1 << 24)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
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
		legacyStatisticsGraph(t, b, region, h)
		mapMust(t, h.CloseChecked())
		mapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}

func legacyStatisticsGraph(t *testing.T, b *arena.Budget, region *arena.BytesRegion, owners ...*StrHashtable) {
	legacyStatisticsRegionsGraph(t, b, []*arena.BytesRegion{region}, owners...)
}
func legacyStatisticsRegionsGraph(t *testing.T, b *arena.Budget, regions []*arena.BytesRegion, owners ...*StrHashtable) {
	t.Helper()
	var sum uint64
	seen := map[*StrHashtable]bool{}
	for _, h := range owners {
		if h != nil {
			if seen[h] {
				t.Fatal("duplicate named owner")
			}
			seen[h] = true
			sum += legacyAssertionOwnedGraph(t, h)
		}
	}
	rs := map[*arena.BytesRegion]bool{}
	for _, region := range regions {
		if rs[region] {
			t.Fatal("duplicate region")
		}
		rs[region] = true
		s := region.Stats()
		d := s.Descriptors
		sum += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	if b.Used() != sum {
		t.Fatal("named scalar/table/cold graph", b.Used(), sum)
	}
}

func TestStrHashtableLegacyStatisticsEveryConstructorCopyAndReplacementReservation(t *testing.T) {
	for _, action := range []string{"construct", "copy", "reset", "finalize", "grow"} {
		first := -1
		for ordinal := int64(0); ordinal < 16; ordinal++ {
			b := arena.NewBudget(1 << 26)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			var src, out *StrHashtable
			if action != "construct" {
				cap := uint32(128)
				if action == "copy" || action == "grow" {
					cap = 8
				}
				src, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true, InitialCapacity: cap})
				mapMust(t, e)
				if action == "grow" {
					for i := 0; i < 7; i++ {
						mapMust(t, src.Insert(strStore(t, src, fmt.Sprintf("growth-%d", i), 0)))
					}
				} else {
					mapMust(t, src.Insert(strStore(t, src, "existing", 0)))
				}
				mapMust(t, src.writeCollision(29))
			}
			var next StrKey
			if action == "grow" {
				next = strStore(t, src, "next", 0)
			}
			b.FailAfter(ordinal)
			switch action {
			case "construct":
				out, e = NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
			case "copy":
				out, e = src.Clone()
			case "reset":
				e = src.Reset()
			case "finalize":
				e = src.Finalize()
			case "grow":
				e = src.Insert(next)
			}
			cause := e
			legacyStatisticsGraph(t, b, region, src, out)
			if out != nil && e == nil && out.CollisionCount() != 0 {
				t.Fatal("actual completed ctor counter init", action, ordinal, out.CollisionCount())
			}
			if src != nil && (action != "grow" || e != nil) && src.CollisionCount() != 29 {
				t.Fatal("body/copy never resets older statistics", action, ordinal, src.CollisionCount())
			}
			if src != nil {
				mapMust(t, src.SourceFinalize())
			}
			if out != nil && out.raw.complete {
				mapMust(t, out.SourceFinalize())
			}
			legacyStatisticsGraph(t, b, region, src, out)
			if out != nil {
				mapMust(t, out.CloseChecked())
			}
			if src != nil {
				mapMust(t, src.CloseChecked())
			}
			b.FailAfter(-1)
			mapMust(t, region.Close())
			if b.Used() != 0 {
				t.Fatal(action, ordinal, b.Used())
			}
			if cause == nil {
				first = int(ordinal)
				break
			}
		}
		want := 2
		if action == "reset" || action == "finalize" {
			want = 1
		}
		if first != want {
			t.Fatal("actual prefix horizon", action, first, want)
		}
		t.Log(action, "first success", first)
	}
}

func TestStrHashtableLegacyStatisticsUnsignedScalarAndCopyBodyOrder(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	foreign, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
	mapMust(t, e)
	storage, e := foreign.Allocate(8)
	mapMust(t, e)
	for i, x := range []byte("foreign\x00") {
		mapMust(t, foreign.Write(storage, uint32(i), x))
	}
	key, e := BorrowCString(foreign, storage, 0)
	mapMust(t, e)
	mapMust(t, h.InsertFrom(foreign, key))
	// Concrete field/arithmetic codec proof, not a claim of UINT64_MAX collisions
	// generated by this bounded source campaign.
	mapMust(t, h.writeCollision(^uint64(0)))
	mapMust(t, h.incrementCollision())
	if h.CollisionCount() != 0 {
		t.Fatal("unsigned64 field wrap")
	}
	mapMust(t, h.writeCollision(31))
	b.FailAfter(2)
	failed, cause := h.Clone()
	if cause == nil || failed == nil || failed.table == nil || failed.raw.complete || failed.raw.initialized&0xC0 != 0 {
		t.Fatal("copy stops before source st_init", cause, failed)
	}
	legacyStatisticsRegionsGraph(t, b, []*arena.BytesRegion{region, foreign}, h, failed)
	if failed.SourceFinalize() == nil {
		t.Fatal("failed ctor has no completed core DT")
	}
	b.FailAfter(-1)
	mapMust(t, failed.CloseChecked())
	copy, e := h.Clone()
	mapMust(t, e)
	if copy.CollisionCount() != 0 || h.CollisionCount() != 31 {
		t.Fatal("copy counter st_init after payload copy")
	}
	legacyStatisticsRegionsGraph(t, b, []*arena.BytesRegion{region, foreign}, h, copy)
	mapMust(t, copy.CloseChecked())
	mapMust(t, h.CloseChecked())
	mapMust(t, foreign.Close())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableLegacyStatisticsBothInsertMacrosAndProfileBinding(t *testing.T) {
	for _, core := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
		mapMust(t, e)
		ordinary, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
		mapMust(t, e)
		before := b.Used()
		if e = h.Swap(ordinary); e == nil || b.Used() != before || h.CollisionCount() != 0 || h.Size() != 0 || ordinary.Size() != 0 {
			t.Fatal("compile-profile domain must not exchange unequal fields", e)
		}
		keys := collisionStrKeys(t, h, 7, 7, 4)
		for i, k := range keys[:3] {
			beforeCount := h.CollisionCount()
			if core {
				added, _, err := h.InsertIfNotThereCore(k)
				mapMust(t, err)
				if !added {
					t.Fatal("new actual entry")
				}
			} else {
				mapMust(t, h.Insert(k))
			}
			if delta := h.CollisionCount() - beforeCount; delta != uint64(i) {
				t.Fatal("one increment per used miss", core, i, delta)
			}
		}
		beforeCount := h.CollisionCount()
		if core {
			added, _, err := h.InsertIfNotThereCore(keys[1])
			mapMust(t, err)
			if added {
				t.Fatal("duplicate")
			}
		} else {
			mapMust(t, h.Insert(keys[1]))
		}
		if delta := h.CollisionCount() - beforeCount; delta != 1 {
			t.Fatal("preceding used miss, matched row unread HS", core, delta)
		}
		mapMust(t, h.Remove(keys[0]))
		beforeCount = h.CollisionCount()
		if core {
			added, _, err := h.InsertIfNotThereCore(keys[3])
			mapMust(t, err)
			if !added {
				t.Fatal("replacement")
			}
		} else {
			mapMust(t, h.Insert(keys[3]))
		}
		if delta := h.CollisionCount() - beforeCount; delta != 3 || h.Deleted() != 0 || h.Size() != 3 {
			t.Fatal("deleted then used misses before actual free publication", core, delta, h.Deleted(), h.Size())
		}
		legacyStatisticsGraph(t, b, region, h, ordinary)
		mapMust(t, h.CloseChecked())
		mapMust(t, ordinary.CloseChecked())
		mapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
