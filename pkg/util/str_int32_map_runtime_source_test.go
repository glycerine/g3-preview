// Source: src/util/map.h:24-196; src/util/hashtable.h:44-63,138-140,243-283,353-653.
package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrInt32MapRuntimeActualDefaultEntryStateAndKeyOnlyRead(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true})
	mapMust(t, e)
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	if h.table != nil || roots.Object == (Block{}) || roots.Table == (Block{}) {
		t.Fatal("real Runtime headers/payload, no shadow", roots)
	}
	n, e := r.Size(roots.Table)
	mapMust(t, e)
	if n != 8*40 {
		t.Fatal("actual concrete default-entry extent", n)
	}
	for i := uint64(0); i < 8; i++ {
		state, e := r.ReadUint32(roots.Table, i*40+4)
		mapMust(t, e)
		if state != 0 {
			t.Fatal(state)
		}
		if _, e = r.ReadUint32(roots.Table, i*40+32); e != ErrRuntimeUninitialized {
			t.Fatal("default key_data value initialized eagerly", i, e)
		}
	}
	ref, e := region.Allocate(4)
	mapMust(t, e)
	for i, x := range []byte("key\x00") {
		mapMust(t, region.Write(ref, uint32(i), x))
	}
	key, e := BorrowCString(region, ref, 0)
	mapMust(t, e)
	mapMust(t, h.Insert(key, -123))
	value, ok, e := h.Find(key)
	mapMust(t, e)
	if !ok || value != -123 {
		t.Fatal(value, ok)
	}
	if h.CollisionCount() != 0 {
		t.Fatal("matched source count")
	}
	mapMust(t, h.SourceFinalize())
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func runtimeMapGraph(t *testing.T, b *arena.Budget, r *Runtime, regions []*arena.BytesRegion, owners ...*StrInt32Map) {
	t.Helper()
	roots := map[Block]bool{}
	var cold uint64
	seen := map[*StrInt32Map]bool{}
	for _, h := range owners {
		if h == nil {
			continue
		}
		if seen[h] {
			t.Fatal("duplicate owner")
		}
		seen[h] = true
		rr, e := h.RuntimeRoots()
		mapMust(t, e)
		cold += h.rootBytes + h.directoryBytes + rr.AcquisitionBytes
		for _, token := range append([]Block{rr.Object, rr.Table}, rr.Acquisitions...) {
			if token == (Block{}) {
				continue
			}
			if roots[token] {
				t.Fatal("duplicate actual root", token)
			}
			roots[token] = true
			_, e = r.Size(token)
			mapMust(t, e)
		}
	}
	rs := r.Stats()
	if uint32(len(roots)) != rs.LiveBlocks {
		t.Fatal("named actual map root bijection", roots, rs)
	}
	for _, region := range regions {
		s := region.Stats()
		d := s.Descriptors
		cold += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	if b.Used() != rs.TotalBytes+cold {
		t.Fatal("actual raw/cold graph", b.Used(), rs.TotalBytes+cold)
	}
}

func TestStrInt32MapRuntimeEveryActualReservationAndDefaultEntryValues(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, action := range []string{"construct", "copy", "intersect", "grow", "reset", "finalize"} {
			success := -1
			for ordinal := int64(0); ordinal < 24; ordinal++ {
				b := arena.NewBudget(1 << 26)
				r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: bits})
				mapMust(t, e)
				region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
				mapMust(t, e)
				var h, other, out *StrInt32Map
				if action != "construct" {
					cap := uint32(128)
					if action == "grow" || action == "copy" || action == "intersect" {
						cap = 8
					}
					h, e = NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true, InitialCapacity: cap})
					mapMust(t, e)
					if action == "grow" {
						for i := 0; i < 7; i++ {
							mapMust(t, h.Insert(simKey(t, region, fmt.Sprintf("growth-%d", i)), int32(i-3)))
						}
					} else {
						mapMust(t, h.Insert(simKey(t, region, "existing"), -313))
					}
					if action == "intersect" {
						other, e = NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true})
						mapMust(t, e)
					}
				}
				var next StrKey
				if action == "grow" {
					next = simKey(t, region, "next")
				}
				b.FailAfter(ordinal)
				switch action {
				case "construct":
					out, e = NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true})
				case "copy":
					out, e = h.Clone()
				case "intersect":
					e = h.Intersect(other)
				case "grow":
					e = h.Insert(next, -991)
				case "reset":
					e = h.Reset()
				case "finalize":
					e = h.Finalize()
				}
				cause := e
				runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h, other, out)
				if cause == nil && out != nil && action == "copy" {
					v, ok, err := out.FindCString("existing")
					mapMust(t, err)
					if !ok {
						t.Fatal("copied key")
					}
					n, err := v.Value()
					mapMust(t, err)
					if n != -313 {
						t.Fatal("copied int field", n)
					}
				}
				b.FailAfter(-1)
				for _, owner := range []*StrInt32Map{out, other, h} {
					if owner != nil {
						mapMust(t, owner.CloseChecked())
					}
				}
				mapMust(t, region.Close())
				mapMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal(bits, action, ordinal, b.Used())
				}
				if cause == nil {
					success = int(ordinal)
					break
				}
			}
			if success < 0 {
				t.Fatal("missing actual first success", bits, action)
			}
			want := map[string]int{"construct": 6, "copy": 6, "intersect": 5, "grow": 3, "reset": 2, "finalize": 2}[action]
			if success != want {
				t.Fatal("actual Runtime reservation horizon", bits, action, success, want)
			}
			t.Log("actual first success", bits, action, success)
		}
	}
}

func TestStrInt32MapRuntimeIndependentFiniteValueIntersection(t *testing.T) {
	for left := 0; left < 16; left++ {
		for right := 0; right < 16; right++ {
			b := arena.NewBudget(1 << 24)
			r, e := NewRuntime(RuntimeConfig{Budget: b})
			mapMust(t, e)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r})
			mapMust(t, e)
			other, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r})
			mapMust(t, e)
			var keys [4]StrKey
			for i := range keys {
				keys[i] = simKey(t, region, fmt.Sprintf("finite-%d", i))
				if left&(1<<i) != 0 {
					mapMust(t, h.Insert(keys[i], int32(17*i-41)))
				}
				if right&(1<<i) != 0 {
					mapMust(t, other.Insert(keys[i], int32(991+i)))
				}
			}
			mapMust(t, h.Intersect(other))
			var want uint32
			for i, k := range keys {
				value, ok, err := h.Find(k)
				mapMust(t, err)
				present := left&right&(1<<i) != 0
				if ok != present || ok && value != int32(17*i-41) {
					t.Fatal(left, right, i, value, ok)
				}
				if present {
					want++
				}
			}
			if h.Size() != want {
				t.Fatal(left, right, h.Size(), want)
			}
			runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h, other)
			mapMust(t, h.CloseChecked())
			mapMust(t, other.CloseChecked())
			mapMust(t, region.Close())
			mapMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
}

func TestStrInt32MapRuntimeKeyOnlyLookupAndCopyReachedValueGuard(t *testing.T) {
	// Explicit partially initialized default-entry representation domain. The
	// key-only source predicates do not read m_value; value/copy then reach it.
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	mapMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	key := simKey(t, region, "partial-value")
	hash, e := h.Hash(key)
	mapMust(t, e)
	slot := hash & 7
	roots, e := h.RuntimeRoots()
	mapMust(t, e)
	mapMust(t, h.writeKeyAt(roots.Table, slot, key))
	mapMust(t, r.WriteUint32(roots.Table, uint64(slot)*40, hash))
	mapMust(t, r.WriteUint32(roots.Table, uint64(slot)*40+4, strHashUsed))
	mapMust(t, r.WriteUint32(roots.Object, 12, 1))
	yes, e := h.Contains(key)
	mapMust(t, e)
	if !yes {
		t.Fatal("key-only matched lookup")
	}
	if _, _, e = h.Find(key); e != ErrRuntimeUninitialized {
		t.Fatal("actual value read guard", e)
	}
	copy, cause := h.Clone()
	if !errors.Is(cause, ErrRuntimeUninitialized) || copy == nil || copy.raw.complete {
		t.Fatal("copy reached value before state/hash publication", copy, cause)
	}
	copied, e := copy.RuntimeRoots()
	mapMust(t, e)
	for j := uint64(0); j < 6; j++ {
		_, e = r.ReadUint32(copied.Table, uint64(slot)*40+8+j*4)
		mapMust(t, e)
	}
	state, e := r.ReadUint32(copied.Table, uint64(slot)*40+4)
	mapMust(t, e)
	if state != strHashUsed {
		t.Fatal("literal assigned state before source value guard", state)
	}
	runtimeMapGraph(t, b, r, []*arena.BytesRegion{region}, h, copy)
	if copy.SourceFinalize() == nil {
		t.Fatal("failed copy no completed core DT")
	}
	mapMust(t, copy.CloseChecked())
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestStrInt32MapLegacyEveryActualReservationAndDefaultEntryValues(t *testing.T) {
	for _, bits := range []uint8{0} {
		for _, action := range []string{"construct", "copy", "intersect", "grow", "reset", "finalize"} {
			success := -1
			for ordinal := int64(0); ordinal < 24; ordinal++ {
				b := arena.NewBudget(1 << 26)
				var r *Runtime
				var e error
				region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
				mapMust(t, e)
				var h, other, out *StrInt32Map
				if action != "construct" {
					cap := uint32(128)
					if action == "grow" || action == "copy" || action == "intersect" {
						cap = 8
					}
					h, e = NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true, InitialCapacity: cap})
					mapMust(t, e)
					if action == "grow" {
						for i := 0; i < 7; i++ {
							mapMust(t, h.Insert(simKey(t, region, fmt.Sprintf("growth-%d", i)), int32(i-3)))
						}
					} else {
						mapMust(t, h.Insert(simKey(t, region, "existing"), -313))
					}
					if action == "intersect" {
						other, e = NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true})
						mapMust(t, e)
					}
				}
				var next StrKey
				if action == "grow" {
					next = simKey(t, region, "next")
				}
				b.FailAfter(ordinal)
				switch action {
				case "construct":
					out, e = NewStrInt32Map(region, StrInt32MapConfig{Budget: b, Runtime: r, Statistics: true})
				case "copy":
					out, e = h.Clone()
				case "intersect":
					e = h.Intersect(other)
				case "grow":
					e = h.Insert(next, -991)
				case "reset":
					e = h.Reset()
				case "finalize":
					e = h.Finalize()
				}
				cause := e
				legacyMapGraph(t, b, []*arena.BytesRegion{region}, h, other, out)
				if cause == nil && out != nil && action == "copy" {
					v, ok, err := out.FindCString("existing")
					mapMust(t, err)
					if !ok {
						t.Fatal("copied key")
					}
					n, err := v.Value()
					mapMust(t, err)
					if n != -313 {
						t.Fatal("copied int field", n)
					}
				}
				b.FailAfter(-1)
				for _, owner := range []*StrInt32Map{out, other, h} {
					if owner != nil {
						mapMust(t, owner.CloseChecked())
					}
				}
				mapMust(t, region.Close())

				if b.Used() != 0 {
					t.Fatal(bits, action, ordinal, b.Used())
				}
				if cause == nil {
					success = int(ordinal)
					break
				}
			}
			if success < 0 {
				t.Fatal("missing actual first success", bits, action)
			}
			want := map[string]int{"construct": 2, "copy": 2, "intersect": 3, "grow": 2, "reset": 1, "finalize": 1}[action]
			if success != want {
				t.Fatal("actual legacy reservation horizon", action, success, want)
			}
			t.Log("actual first success", bits, action, success)
		}
	}
}
