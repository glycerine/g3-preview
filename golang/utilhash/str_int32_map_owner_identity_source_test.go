// Source: src/util/hashtable.h:243-275; src/util/memory_manager.h:112-119.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestStrInt32MapLegacyAccidentalGoCopyCannotOwnSourceActions(t *testing.T) {
	for _, action := range []string{"close", "source", "reset", "insert"} {
		t.Run(action, func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b})
			mapMust(t, e)
			key := simKey(t, region, "live")
			mapMust(t, h.Insert(key, -19))
			copied := *h
			before := b.Used()
			table := h.table
			size := h.size
			epoch := h.epoch
			switch action {
			case "close":
				e = copied.CloseChecked()
			case "source":
				e = copied.SourceFinalize()
			case "reset":
				e = copied.Reset()
			case "insert":
				e = copied.Insert(key, -19)
			}
			if !errors.Is(e, arena.ErrConfig) || b.Used() != before || h.size != size || h.epoch != epoch || h.closed || h.table == nil || &h.table[0] != &table[0] || copied.closed || copied.raw.destructing {
				t.Fatal("copied Go wrapper gained source allocation authority", action, e, b.Used(), before)
			}
			clone, e := h.Clone()
			mapMust(t, e)
			mapMust(t, clone.CloseChecked())
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
	}
}
func TestStrInt32MapZeroGoWrapperCannotOwnCleanup(t *testing.T) {
	var h StrInt32Map
	if e := h.CloseChecked(); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("unprepared wrapper cleanup", e)
	}
	if e := h.SourceFinalize(); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("unprepared source object", e)
	}
}
