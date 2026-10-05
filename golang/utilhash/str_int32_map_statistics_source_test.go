// Source: src/util/map.h:24-196; src/util/hashtable.h:44-63,138-140,254-283,359-424,481-543,649-653.
// Source author: Leonardo de Moura (2006-10-14; 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestStrInt32MapStatisticsIndependentDefaultEntryProbeCounts(t *testing.T) {
	for _, physical := range []bool{false, true} {
		for mask := 0; mask < 8; mask++ {
			b := arena.NewBudget(1 << 24)
			var r *memory.Runtime
			var e error
			if physical {
				r, e = memory.NewRuntime(memory.RuntimeConfig{Budget: b})
				mapMust(t, e)
			}
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true})
			mapMust(t, e)
			keys := simCollisionKeys(t, region, 0, 7, 4)
			for i, k := range keys[:3] {
				mapMust(t, h.Insert(k, int32(-31-i)))
			}
			deleted := uint64(0)
			for i := 0; i < 3; i++ {
				if mask&(1<<i) != 0 {
					mapMust(t, h.Remove(keys[i]))
					deleted++
				}
			}
			want := uint64(6) - deleted
			if mask&4 != 0 {
				want--
			}
			text, e := h.Text(keys[3])
			mapMust(t, e)
			before := h.CollisionCount()
			_, ok, e := h.FindCStringBytes([]byte(text))
			mapMust(t, e)
			if ok || h.CollisionCount()-before != want {
				t.Fatal(physical, mask, "byte query", ok, h.CollisionCount()-before, want)
			}
			before = h.CollisionCount()
			_, ok, e = h.FindCString(text)
			mapMust(t, e)
			if ok || h.CollisionCount()-before != want {
				t.Fatal(physical, mask, "string query", ok, h.CollisionCount()-before, want)
			}
			before = h.CollisionCount()
			mapMust(t, h.Remove(keys[3]))
			if h.CollisionCount()-before != want {
				t.Fatal(physical, mask, "remove", h.CollisionCount()-before, want)
			}
			out, e := h.Clone()
			mapMust(t, e)
			if out.CollisionCount() != 0 {
				t.Fatal("copy constructor st_init", out.CollisionCount())
			}
			count := h.CollisionCount()
			mapMust(t, h.Swap(out))
			if h.CollisionCount() != 0 || out.CollisionCount() != count {
				t.Fatal("source scalar member swap")
			}
			for i, k := range keys[:3] {
				if mask&(1<<i) == 0 {
					v, ok, e := h.Find(k)
					mapMust(t, e)
					if !ok || v != int32(-31-i) {
						t.Fatal("value member preserved", i, v, ok)
					}
				}
			}
			if physical {
				runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h, out)
			} else {
				legacyMapGraph(t, b, []*arena.BytesRegion{region}, h, out)
			}
			mapMust(t, out.CloseChecked())
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			if r != nil {
				mapMust(t, r.Close())
			}
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
}

func legacyMapGraph(t *testing.T, b *arena.Budget, regions []*arena.BytesRegion, owners ...*StrInt32Map) {
	t.Helper()
	var total uint64
	seen := map[*StrInt32Map]bool{}
	var add func(*StrInt32Map)
	add = func(h *StrInt32Map) {
		if h == nil {
			return
		}
		if seen[h] {
			t.Fatal("duplicate named map")
		}
		seen[h] = true
		total += h.rootBytes + h.tableBytes + h.directoryBytes
		for a := h.raw.legacyAcquisitions; a != nil; a = a.next {
			if a.bytes == 0 {
				t.Fatal("retired node retained")
			}
			total += a.bytes + a.tableBytes
		}
		for f := h.raw.lexical; f != nil; f = f.next {
			total += f.bytes
			add(f.copy)
		}
	}
	for _, h := range owners {
		add(h)
	}
	for _, r := range regions {
		s := r.Stats()
		d := s.Descriptors
		total += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	if total != b.Used() {
		t.Fatal("named actual default-entry cold graph", total, b.Used())
	}
}
