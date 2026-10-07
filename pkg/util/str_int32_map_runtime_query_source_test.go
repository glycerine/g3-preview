// Source: src/util/str_hashtable.h:28-30; src/util/hashtable.h:375-435,465-522.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrInt32MapRuntimeCStringBorrowedSourceQuery(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	h, e := NewStrInt32Map(region, StrInt32MapConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	p, e := region.Allocate(6)
	if e != nil {
		t.Fatal(e)
	}
	for i, c := range []byte{'a', 'b', 'c', 0, 'x', 0} {
		if e = region.Write(p, uint32(i), c); e != nil {
			t.Fatal(e)
		}
	}
	k, e := h.CString(p, 0)
	if e != nil {
		t.Fatal(e)
	}
	e = h.Insert(k, 17)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := r.Allocate(6)
	if e != nil {
		t.Fatal(e)
	}
	for i, c := range []byte{'a', 'b', 'c', 0, 'x', 0} {
		if e = r.Write(raw, uint64(i), c); e != nil {
			t.Fatal(e)
		}
	}
	b.FailAfter(0)
	entry, yes, e := h.PrepareFindCStringRuntime(r, raw, 0, 6)
	if e != nil || !yes {
		t.Fatal("actual warm borrowed query", yes, e)
	}
	v, e := entry.Value()
	if e != nil || v != 17 {
		t.Fatal(v, e)
	}
	b.FailAfter(-1)
	if e = r.Deallocate(&raw); e != nil {
		t.Fatal(e)
	}
	if _, _, e = h.FindCStringRuntime(r, raw, 0, 6); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("stale reached borrowed storage", e)
	}
	if e = h.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = region.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
