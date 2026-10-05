// Source: src/util/str_hashtable.h:28-30; src/util/hash.cpp:26-102.
// Source author: Leonardo de Moura (2006-09-13; 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestStrHashtableCStringHashAndEqualRereadActualUnstoredPointer(t *testing.T) {
	for _, physical := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "runtime"}[physical], func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
			mapMust(t, e)
			var r *memory.Runtime
			if physical {
				r, e = memory.NewRuntime(memory.RuntimeConfig{Budget: b})
				mapMust(t, e)
			}
			h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
			mapMust(t, e)
			a := strStore(t, h, "abcdef", 0)
			other := strStore(t, h, "ab", 0)
			// Neither pointer has been inserted. Changing caller-owned bytes cannot
			// invalidate a cached table hash or any table iterator.
			mapMust(t, region.Write(a.ref, a.offset+2, 0))
			b.FailAfter(0)
			hash, e := h.Hash(a)
			mapMust(t, e)
			if hash != String("ab", 17) {
				t.Fatalf("source strlen at hash call: got %08x want %08x", hash, String("ab", 17))
			}
			same, e := h.Equal(a, other)
			mapMust(t, e)
			if !same {
				t.Fatal("source strcmp ignores borrow-time length")
			}
			b.FailAfter(-1)
			mapMust(t, h.CloseChecked())
			mapMust(t, region.Close())
			if r != nil {
				mapMust(t, r.Close())
			}
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
	}
}

func TestStrHashtableCStringBorrowChangedBeforeInsertionQueryEngine(t *testing.T) {
	for _, physical := range []bool{false, true} {
		b := arena.NewBudget(1 << 24)
		region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		mapMust(t, e)
		var r *memory.Runtime
		if physical {
			r, e = memory.NewRuntime(memory.RuntimeConfig{Budget: b})
			mapMust(t, e)
		}
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
		mapMust(t, e)
		key := strStore(t, h, "abcdef", 0)
		mapMust(t, region.Write(key.ref, key.offset+2, 0))
		mapMust(t, h.Insert(key))
		// All subsequent bytes are unchanged. The table cached hash and actual
		// spelling agree at insertion, regardless of the earlier borrow descriptor.
		b.FailAfter(0)
		_, ok, e := h.FindCString("ab")
		mapMust(t, e)
		if !ok {
			t.Fatal("source strcmp query did not find actual inserted spelling")
		}
		_, ok, e = h.FindCStringBytes([]byte("ab"))
		mapMust(t, e)
		if !ok {
			t.Fatal("byte-query strcmp did not find actual inserted spelling")
		}
		b.FailAfter(-1)
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

func TestStrHashtableCStringFiniteLengthAndEqualityOracle(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	mapMust(t, e)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	mapMust(t, e)
	for n := 0; n <= 40; n++ {
		bytes := make([]byte, n)
		for i := range bytes {
			bytes[i] = byte((i*73+129)%255 + 1)
		}
		key := strStore(t, h, string(bytes), 0)
		other := strStore(t, h, string(bytes[:n/2]), 0)
		// A real caller-owned string shortens before every source Hash/Eq/Insert.
		mapMust(t, region.Write(key.ref, key.offset+uint32(n/2), 0))
		b.FailAfter(0)
		got, e := h.Hash(key)
		mapMust(t, e)
		if got != String(string(bytes[:n/2]), 17) {
			t.Fatal(n, got)
		}
		same, e := h.Equal(key, other)
		mapMust(t, e)
		if !same {
			t.Fatal(n, "finite strcmp equality")
		}
		runtimeStringGraph(t, b, r, region, h)
		b.FailAfter(-1)
	}
	mapMust(t, h.CloseChecked())
	mapMust(t, region.Close())
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestRegionBytesActualAscendingWordsDescendingTailAndZeroUnread(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	ref, e := region.Allocate(6)
	mapMust(t, e)
	// Checked invalid raw hash extent. BytesRegion uses zeroed Go backing;
	// its initialization profile differs from Runtime. The shared hash kernel
	// retains the previously proved ascending-word/descending-tail schedule.
	_, e = RegionBytes(region, ref, 0, 11, 17)
	if e != arena.ErrBounds {
		t.Fatal("tail first read", e)
	}
	_, e = RegionBytes(region, ref, 0, 12, 17)
	if e != arena.ErrBounds {
		t.Fatal("reached second whole-word extent", e)
	}
	first, e := (runtimeHashSource{region: region, regionRef: ref, regionMode: true}).wordAt(0)
	mapMust(t, e)
	if first != 0 {
		t.Fatal("actual default backing", first)
	}
	got, e := RegionBytes(nil, arena.BytesRegionRef{}, ^uint64(0), 0, 17)
	mapMust(t, e)
	if got != String("", 17) {
		t.Fatal(got)
	}
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestStrInt32MapSharedCStringHashEqualityAndInsertion(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b})
	mapMust(t, e)
	a := simKey(t, region, "abcdef")
	other := simKey(t, region, "ab")
	mapMust(t, region.Write(a.ref, a.offset+2, 0))
	hash, e := h.Hash(a)
	mapMust(t, e)
	if hash != String("ab", 17) {
		t.Fatal(hash)
	}
	same, e := h.Equal(a, other)
	mapMust(t, e)
	if !same {
		t.Fatal("map uses literal common strcmp")
	}
	mapMust(t, h.Insert(a, 29))
	b.FailAfter(0)
	out, ok, e := h.FindCore(other)
	mapMust(t, e)
	if !ok {
		t.Fatal("map source lookup")
	}
	v, e := out.Value()
	mapMust(t, e)
	if v != 29 {
		t.Fatal(v)
	}
	b.FailAfter(-1)
	h.Close()
	mapMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRegionBytesReachedOffsetOverflowBeforeTokenRead(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	mapMust(t, e)
	// Guarded Go raw-reader domain: native pointer arithmetic is undefined.
	// The first descending tail offset must fail before the invalid token read.
	_, e = RegionBytes(r, arena.BytesRegionRef{}, ^uint64(0), 2, 17)
	if e != arena.ErrBounds {
		t.Fatal("wrapped offset reached token", e)
	}
	got, e := RegionBytes(nil, arena.BytesRegionRef{}, ^uint64(0), 0, 17)
	mapMust(t, e)
	if got != String("", 17) {
		t.Fatal("zero-n erased", got)
	}
	mapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
