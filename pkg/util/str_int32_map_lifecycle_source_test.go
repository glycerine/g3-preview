// Source: src/util/map.h:24-196; src/util/hashtable.h:44-63,146-148,259-322,613-623.
// Source author: Leonardo de Moura (2006-10-14; 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrInt32MapSourceReplacementConsumesOldBeforeRefusal(t *testing.T) {
	for _, finalize := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, InitialCapacity: 128})
		mapMust(t, e)
		ref, e := region.Allocate(4)
		mapMust(t, e)
		for i, x := range []byte("key\x00") {
			mapMust(t, region.Write(ref, uint32(i), x))
		}
		key, e := BorrowCString(region, ref, 0)
		mapMust(t, e)
		mapMust(t, h.Insert(key, 19))
		old := h.tableBytes
		before := b.Used()
		b.FailAfter(0)
		if finalize {
			e = h.Finalize()
		} else {
			e = h.Reset()
		}
		if e == nil || h.table != nil || h.tableBytes != 0 || b.Used() != before-old {
			t.Fatal("source delete publishes NULL before replacement refusal", finalize, e, len(h.table), h.tableBytes, b.Used(), before-old)
		}
		h.Close()
		b.FailAfter(-1)
		mapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func TestStrInt32MapSourceCopyCtorRetainsActualAcquiredDefaultEntries(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	foreign, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b})
	mapMust(t, e)
	ref, e := foreign.Allocate(4)
	mapMust(t, e)
	for i, x := range []byte("key\x00") {
		mapMust(t, foreign.Write(ref, uint32(i), x))
	}
	key, e := BorrowCString(foreign, ref, 0)
	mapMust(t, e)
	mapMust(t, h.InsertFrom(foreign, key, -19))
	b.FailAfter(2)
	out, cause := h.Clone()
	if cause == nil || out == nil || out.table == nil || out.tableBytes != 8*strInt32EntryBytes {
		t.Fatal("failed source copy body lost actual acquired default entries", cause, out)
	}
	b.FailAfter(-1)
	out.Close()
	h.Close()
	mapMust(t, foreign.Close())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
