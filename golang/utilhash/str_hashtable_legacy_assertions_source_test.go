// Source: src/util/hashtable.h:159-215,249,371,395,422,451,559,643-644.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func legacyAssertionOwnedGraph(t *testing.T, h *StrHashtable) uint64 {
	t.Helper()
	total := h.rootBytes + h.tableBytes + h.directoryBytes
	retained := uint64(0)
	for a := h.raw.legacyAcquisitions; a != nil; a = a.next {
		if a.bytes != uint64(reflect.TypeOf(strHashtableLegacyAllocation{}).Size()) || a.tableBytes != uint64(len(a.table))*strHashEntryBytes {
			t.Fatal("actual acquisition header/backing")
		}
		retained += a.bytes + a.tableBytes
	}
	for frame := h.raw.lexical; frame != nil; frame = frame.next {
		retained += frame.bytes
		if frame.copy != nil {
			retained += legacyAssertionOwnedGraph(t, frame.copy)
		}
	}
	if h.Stats().RetainedBytes != retained {
		t.Fatal("retained owner census", h.Stats(), retained)
	}
	return total + retained
}

func TestStrHashtableLegacyGrowthEveryActualReservationBeforeSuccess(t *testing.T) {
	success := false
	for fail := int64(0); fail < 8; fail++ {
		b := arena.NewBudget(1 << 24)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
		mapMust(t, e)
		var keys []StrKey
		for n := 0; n < 8; n++ {
			keys = append(keys, strStore(t, h, fmt.Sprintf("growth-prefix-%d", n), 0))
			if n < 7 {
				mapMust(t, h.Insert(keys[n]))
			}
		}
		old := h.table
		b.FailAfter(fail)
		e = h.Insert(keys[7])
		if b.Used() != legacyAssertionOwnedGraph(t, h) {
			t.Fatal("immediate named growth graph", fail, b.Used())
		}
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || h.Capacity() != 8 || &h.table[0] != &old[0] || h.Size() != 7 {
				t.Fatal("growth failed before OLD delete", fail, e, h.Stats())
			}
		} else {
			success = true
			if h.Capacity() != 16 || h.Size() != 8 || h.raw.legacyAcquisitions != nil {
				t.Fatal("publication/retirement", h.Stats())
			}
		}
		b.FailAfter(-1)
		for _, key := range keys[:7] {
			yes, e := h.Contains(key)
			mapMust(t, e)
			if !yes {
				t.Fatal("old completed payload lost")
			}
		}
		mapMust(t, h.CloseChecked())
		mapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("own growth cleanup")
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("actual growth horizon")
	}
}

func TestStrHashtableLegacyCopyMoveActualDeletedTargetAssertion(t *testing.T) {
	for _, move := range []bool{false, true} {
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
		mapMust(t, e)
		src, e := NewStrHashtable(region, StrHashtableConfig{})
		mapMust(t, e)
		dst, e := NewStrHashtable(region, StrHashtableConfig{})
		mapMust(t, e)
		var keys []StrKey
		for n := 0; len(keys) < 3; n++ {
			text := fmt.Sprintf("legacy-deleted-%d", n)
			if String(text, 17)&7 == 0 {
				keys = append(keys, strStore(t, src, text, 0))
			}
		}
		mapMust(t, src.Insert(keys[0]))
		mapMust(t, dst.Insert(keys[1]))
		mapMust(t, dst.Insert(keys[2]))
		mapMust(t, dst.Remove(keys[1]))
		// Actual explicit-array helper precondition domain: source-owned target has
		// a legitimately deleted slot. No header corruption or artificial fault.
		e = dst.copyLegacyEntries(src, src.table, dst.table, 8, 8, move)
		if strHashtableSourceDebug {
			if !errors.Is(e, ErrStrHashtableAssertion) || dst.table[0].state != strHashDeleted {
				t.Fatal("literal deleted predicate before free", move, e)
			}
		} else {
			mapMust(t, e)
			if dst.table[2].key != keys[0] {
				t.Fatal("release erased predicate before reached free")
			}
		}
		src.Close()
		dst.Close()
		mapMust(t, region.Close())
	}
}

func TestStrHashtableLegacyInvariantDefaultSourceAssertion(t *testing.T) {
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{})
	mapMust(t, e)
	// Explicit malformed source count-domain: this is assertion ordering proof,
	// not an ordinary table algorithm failure or an allocator refusal.
	h.size = 1
	h.deleted = 1
	yes, e := h.CheckInvariantChecked()
	if strHashtableSourceDebug {
		if !errors.Is(e, ErrStrHashtableAssertion) || yes {
			t.Fatal("first deleted-count source assertion", yes, e)
		}
	} else if e != nil || yes {
		t.Fatal("release inspection counterpart", yes, e)
	}
	h.Close()
	mapMust(t, region.Close())
}

func TestStrHashtableLegacyActualExhaustedProbeUnreachableProfiles(t *testing.T) {
	for _, core := range []bool{false, true} {
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{InitialCapacity: 1})
		mapMust(t, e)
		mapMust(t, h.Insert(strStore(t, h, "used", 0)))
		// Malformed source count-domain only: the actual full array remains owned.
		// This suppresses expansion to reach the real exhausted probe guard.
		h.size = 0
		key := strStore(t, h, "missing", 0)
		if core {
			_, _, e = h.InsertIfNotThereCore(key)
		} else {
			e = h.Insert(key)
		}
		if !errors.Is(e, ErrStrHashtableUnreachable) || h.table[0].state != strHashUsed || h.size != 0 {
			t.Fatal("actual source exhausted branch", core, e)
		}
		h.Close()
		mapMust(t, region.Close())
	}
}

func TestStrHashtableLegacyCapturedReachedPointerRetainsLiveBacking(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
	mapMust(t, e)
	a, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	z, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	var key StrKey
	for n := 0; ; n++ {
		text := fmt.Sprintf("captured-%d", n)
		if String(text, 17)&7 == 0 {
			key = strStore(t, a, text, 0)
			break
		}
	}
	mapMust(t, a.Insert(key))
	captured := a.table
	pointer, e := legacyEntryAt(captured, 0)
	mapMust(t, e)
	before := b.Used()
	mapMust(t, a.Swap(z))
	if pointer != &z.table[0] || pointer == &a.table[0] || pointer.key != key || a.table[0].state != strHashFree {
		t.Fatal("reached pointer was recaptured from changed receiver")
	}
	if b.Used() != before || b.Used() != legacyAssertionOwnedGraph(t, a)+legacyAssertionOwnedGraph(t, z) {
		t.Fatal("actual disjoint owning ranges")
	}
	// Pointer-representation proof only: both old/new owning arrays remain live;
	// no fake debugger callback or claim of a first allocator/source failure.
	pointer.hash ^= 1
	if z.table[0].hash != pointer.hash || a.table[0].hash != 0 {
		t.Fatal("write must address captured actual array")
	}
	// Restore the actual cached hash before ordinary class use.
	pointer.hash ^= 1
	yes, e := z.Contains(key)
	mapMust(t, e)
	if !yes {
		t.Fatal("actual payload owner")
	}
	if _, e := legacyEntryAt(captured, 8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached physical array extent", e)
	}
	a.Close()
	z.Close()
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal("capture was never an owning root")
	}
}
