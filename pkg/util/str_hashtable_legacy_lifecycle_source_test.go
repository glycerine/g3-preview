// Source: src/util/hashtable.h:259-275,287-322,613-623.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableLegacyResetFinalizeActualDeleteBeforeReplacementRefusal(t *testing.T) {
	for _, finalize := range []bool{false, true} {
		t.Run(map[bool]string{false: "reset", true: "finalize"}[finalize], func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, InitialCapacity: 128})
			mapMust(t, e)
			mapMust(t, h.Insert(strStore(t, h, "one", 0)))
			before := b.Used()
			oldBytes := h.tableBytes
			b.FailAfter(0)
			if finalize {
				e = h.Finalize()
			} else {
				e = h.Reset()
			}
			if e == nil {
				t.Fatal("actual first replacement reservation did not fail")
			}
			if h.tableBytes != 0 || h.table != nil || b.Used() != before-oldBytes {
				t.Fatal("source delete_table must consume old owning backing before replacement refusal", h.tableBytes, b.Used(), before-oldBytes)
			}
			want := uint32(64)
			if finalize {
				want = 128
			}
			if h.Capacity() != want || h.Size() != 1 || h.Deleted() != 0 {
				t.Fatal("literal scalar publication", h.Capacity(), h.Size(), h.Deleted(), want)
			}
			b.FailAfter(-1)
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
	}
}

func TestStrHashtableLegacyCopyCtorForeignBindingRefusalRetainsActualTable(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	foreign, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	storage, e := foreign.Allocate(8)
	mapMust(t, e)
	for i, c := range []byte("foreign\x00") {
		mapMust(t, foreign.Write(storage, uint32(i), c))
	}
	key, e := BorrowCString(foreign, storage, 0)
	mapMust(t, e)
	mapMust(t, h.InsertFrom(foreign, key))
	b.FailAfter(2)
	copy, e := h.Clone()
	if e == nil {
		t.Fatal("real foreign-binding directory reservation did not refuse")
	}
	if copy == nil || copy.tableBytes != 8*strHashEntryBytes || len(copy.table) != 8 {
		t.Fatal("uncompleted source copy ctor lost acquired raw table", copy, e)
	}
	if copy.raw.complete {
		t.Fatal("failed outer constructor claimed completion")
	}
	b.FailAfter(-1)
	mapMust(t, copy.CloseChecked())
	mapMust(t, h.CloseChecked())
	mapMust(t, foreign.Close())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestStrHashtableLegacyLexicalCopyCtorFailureRetainedByActualFrame(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	foreign, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	other, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	storage, e := foreign.Allocate(8)
	mapMust(t, e)
	for i, c := range []byte("foreign\x00") {
		mapMust(t, foreign.Write(storage, uint32(i), c))
	}
	key, e := BorrowCString(foreign, storage, 0)
	mapMust(t, e)
	mapMust(t, h.InsertFrom(foreign, key))
	b.FailAfter(3)
	e = h.Intersect(other)
	if e == nil {
		t.Fatal("actual lexical directory reservation did not refuse")
	}
	frame := h.raw.lexical
	if frame == nil || frame.copy == nil || frame.complete || frame.copy.raw.complete || frame.copy.tableBytes != 8*strHashEntryBytes {
		t.Fatal("actual failed lexical ctor owner was not retained", frame, e)
	}
	if h.Size() != 1 {
		t.Fatal("body entered after failed copy constructor", h.Size())
	}
	b.FailAfter(-1)
	mapMust(t, h.CloseChecked())
	mapMust(t, other.CloseChecked())
	mapMust(t, foreign.Close())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// Independent census of the actual Go representation, not a scan of provider
// allocations. Every wrapper and lexical frame occurs once in the owning tree.
func legacyOwnedGraph(t *testing.T, seen map[*StrHashtable]bool, h *StrHashtable) uint64 {
	t.Helper()
	if seen[h] {
		t.Fatal("duplicate owning wrapper")
	}
	seen[h] = true
	if h.tableBytes != uint64(len(h.table))*strHashEntryBytes {
		t.Fatal("actual backing extent")
	}
	total := h.rootBytes + h.directoryBytes + h.tableBytes
	var retained uint64
	for frame := h.raw.lexical; frame != nil; frame = frame.next {
		retained += frame.bytes
		if frame.copy != nil {
			retained += legacyOwnedGraph(t, seen, frame.copy)
		}
	}
	if h.Stats().RetainedBytes != retained {
		t.Fatal("observer omitted or double-counted a lexical root")
	}
	return total + retained
}

func TestStrHashtableLegacyCopyAndIntersectEveryReservationPrefix(t *testing.T) {
	for _, intersection := range []bool{false, true} {
		firstSuccess := false
		for ordinal := int64(0); ordinal < 16; ordinal++ {
			b := arena.NewBudget(1 << 24)
			regionBudget := arena.NewBudget(1 << 24)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: regionBudget}})
			mapMust(t, e)
			foreign, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: regionBudget}})
			mapMust(t, e)
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
			mapMust(t, e)
			other, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
			mapMust(t, e)
			storage, e := foreign.Allocate(2)
			mapMust(t, e)
			mapMust(t, foreign.Write(storage, 0, 'x'))
			mapMust(t, foreign.Write(storage, 1, 0))
			key, e := BorrowCString(foreign, storage, 0)
			mapMust(t, e)
			mapMust(t, h.InsertFrom(foreign, key))
			b.FailAfter(ordinal)
			var copied *StrHashtable
			if intersection {
				e = h.Intersect(other)
			} else {
				copied, e = h.Clone()
			}
			seen := make(map[*StrHashtable]bool)
			owned := legacyOwnedGraph(t, seen, h) + legacyOwnedGraph(t, seen, other)
			if copied != nil {
				owned += legacyOwnedGraph(t, seen, copied)
			}
			if b.Used() != owned {
				t.Fatal("immediate named cold/backing graph", intersection, ordinal, b.Used(), owned)
			}
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if e != nil && intersection && h.Size() != 1 {
				t.Fatal("failed constructor entered body")
			}
			if copied != nil && e != nil && copied.raw.complete {
				t.Fatal("failed constructor destructor authority")
			}
			if intersection && e == nil && (h.Size() != 0 || h.raw.lexical != nil) {
				t.Fatal("completed copy source destructor was not drained")
			}
			if e == nil {
				firstSuccess = true
			}
			// Borrowed payload is independently live before any owning abort.
			if c, err := foreign.Read(storage, 0); err != nil || c != 'x' {
				t.Fatal("borrowed payload lost", err)
			}
			b.FailAfter(-1)
			if copied != nil {
				mapMust(t, copied.CloseChecked())
			}
			mapMust(t, h.CloseChecked())
			mapMust(t, other.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("once-only owning refund", ordinal, b.Used())
			}
			mapMust(t, foreign.Close())
			mapMust(t, region.Close())
			if regionBudget.Used() != 0 {
				t.Fatal("foreign provider refund")
			}
			if firstSuccess {
				break
			}
		}
		if !firstSuccess {
			t.Fatal("reservation horizon did not reach success", intersection)
		}
	}
}

func TestStrHashtableLegacyIntersectionIndependentFiniteSets(t *testing.T) {
	for left := 0; left < 16; left++ {
		for right := 0; right < 16; right++ {
			b := arena.NewBudget(1 << 24)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
			mapMust(t, e)
			a, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
			mapMust(t, e)
			z, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
			mapMust(t, e)
			keys := make([]StrKey, 4)
			for i := range keys {
				keys[i] = strStore(t, a, string([]byte{byte('a' + i)}), 0)
				if left&(1<<i) != 0 {
					mapMust(t, a.Insert(keys[i]))
				}
				if right&(1<<i) != 0 {
					mapMust(t, z.Insert(keys[i]))
				}
			}
			mapMust(t, a.Intersect(z))
			count := uint32(0)
			for i, key := range keys {
				want := left&right&(1<<i) != 0
				got, e := a.Contains(key)
				mapMust(t, e)
				if got != want {
					t.Fatal(left, right, i)
				}
				if want {
					count++
				}
			}
			if a.Size() != count || a.raw.lexical != nil {
				t.Fatal("finite intersection or completed lexical lifetime")
			}
			if b.Used() != legacyOwnedGraph(t, make(map[*StrHashtable]bool), a)+legacyOwnedGraph(t, make(map[*StrHashtable]bool), z) {
				t.Fatal("bounded completed metadata")
			}
			mapMust(t, a.CloseChecked())
			mapMust(t, z.CloseChecked())
			mapMust(t, region.Close())
			if b.Used() != 0 {
				t.Fatal("finite oracle refund")
			}
		}
	}
}

func TestStrHashtableLegacyCompletedCopyDestructorBeforeOwningHeaderAbort(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	key := strStore(t, h, "kept", 0)
	mapMust(t, h.Insert(key))
	copy, e := h.Clone()
	mapMust(t, e)
	before := b.Used()
	tableBytes := copy.tableBytes
	mapMust(t, copy.SourceFinalize())
	if !copy.raw.complete || copy.raw.sourceStage != 2 || copy.table != nil || copy.rootBytes == 0 || b.Used() != before-tableBytes {
		t.Fatal("source destructor did not consume exactly table backing")
	}
	mapMust(t, copy.SourceFinalize())
	if b.Used() != before-tableBytes {
		t.Fatal("completed destructor replayed release")
	}
	if got, e := h.Contains(key); e != nil || !got {
		t.Fatal("source borrowed owner changed", e)
	}
	mapMust(t, copy.CloseChecked())
	mapMust(t, copy.CloseChecked())
	if b.Used() != legacyOwnedGraph(t, make(map[*StrHashtable]bool), h) {
		t.Fatal("owning header abort consumed foreign root")
	}
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal("remaining graph")
	}
}

func TestStrHashtableLegacyEmptyResetDoesNotReadDestroyedBacking(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, InitialCapacity: 128})
	mapMust(t, e)
	b.FailAfter(0)
	cause := h.Finalize()
	if !errors.Is(cause, arena.ErrInjectedFailure) || h.table != nil || h.Size() != 0 || h.Capacity() != 128 {
		t.Fatal("actual empty finalize prefix", cause)
	}
	before := b.Used()
	epoch := h.epoch
	if e = h.Reset(); e != nil {
		t.Fatal("source empty reset must return before reading dangling table", e)
	}
	if b.Used() != before || h.epoch != epoch || h.raw.sourceCause != cause || h.Capacity() != 128 {
		t.Fatal("empty no-op changed stopped history")
	}
	b.FailAfter(-1)
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal("empty source abort refund")
	}
}

func TestStrHashtableLegacyReplacementBodyFailureNullSourceDestructor(t *testing.T) {
	for _, finalize := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
		mapMust(t, e)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, InitialCapacity: 128})
		mapMust(t, e)
		mapMust(t, h.Insert(strStore(t, h, "source-null-dtor", 0)))
		b.FailAfter(0)
		if finalize {
			e = h.Finalize()
		} else {
			e = h.Reset()
		}
		cause := e
		if !errors.Is(cause, arena.ErrInjectedFailure) || h.table != nil || h.tableBytes != 0 {
			t.Fatal("actual body refusal after null publication", e, h.Stats())
		}
		before := b.Used()
		// src/util/hashtable.h:146-148 publishes NULL; memory_manager.h:112-114
		// skips its destruction. Body cause is history, not a failed DT action.
		if e = h.SourceFinalize(); e != nil {
			t.Fatal("defined source NULL destructor must complete after body refusal", finalize, e)
		}
		if h.raw.sourceStage != 2 || !h.raw.destructing || h.raw.sourceCause != cause || b.Used() != before {
			t.Fatal("null DT prefix/history")
		}
		mapMust(t, h.SourceFinalize())
		if _, e = h.Contains(StrKey{}); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("ended source object guard", e)
		}
		b.FailAfter(-1)
		mapMust(t, h.CloseChecked())
		mapMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("only explicit owning abort refunds Go header")
		}
	}
}

func TestStrHashtableLegacyGrowthBodyFailureNonNullSourceDestructor(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b})
	mapMust(t, e)
	var keys []StrKey
	for n := 0; n < 8; n++ {
		keys = append(keys, strStore(t, h, string([]byte{byte('a' + n)}), 0))
		if n < 7 {
			mapMust(t, h.Insert(keys[n]))
		}
	}
	oldBytes := h.tableBytes
	b.FailAfter(1) // Actual table reservation after the new Go acquisition header.
	cause := h.Insert(keys[7])
	if !errors.Is(cause, arena.ErrInjectedFailure) || h.table == nil || h.tableBytes != oldBytes || h.Size() != 7 {
		t.Fatal("actual OLD remains after failed NEW allocation", cause, h.Stats())
	}
	before := b.Used()
	mapMust(t, h.SourceFinalize())
	if b.Used() != before-oldBytes || h.table != nil || h.raw.sourceStage != 2 {
		t.Fatal("DT consumes actual nonnull OLD after method failure")
	}
	mapMust(t, h.SourceFinalize())
	b.FailAfter(-1)
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal("non-null DT owning header cleanup")
	}
}
