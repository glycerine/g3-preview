// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestGlobalParamsDictionariesSourceThresholdAndTombstoneProbe(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var d gparamsRefDictionary
	paramsMust(t, d.allocate(b, 8))
	defer func() {
		d.close(b)
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	// Seven insertions fill 7/8: source checks OLD load strictly greater 3/4.
	for i := uint32(1); i <= 7; i++ {
		paramsMust(t, d.insert(b, gparamsRefEntry{key: NumericalSymbol(i), hash: 0, cell: i}))
	}
	old := b.Used()
	b.FailAfter(0)
	e := d.insert(b, gparamsRefEntry{key: NumericalSymbol(1), hash: 0, cell: 99})
	if !errors.Is(e, arena.ErrInjectedFailure) || len(d.entries) != 8 || b.Used() != old || d.entries[0].cell != 1 {
		t.Fatal("growth precedes overwrite", e, d, b.Used())
	}
	b.FailAfter(-1)
	paramsMust(t, d.insert(b, gparamsRefEntry{key: NumericalSymbol(1), hash: 0, cell: 99}))
	if len(d.entries) != 16 || d.size != 7 || d.entries[0].cell != 99 {
		t.Fatal(d)
	}
	// Probe remembers LAST deleted bucket, not first, before the first FREE.
	d.entries[1].state = 1
	d.entries[2].state = 1
	d.size -= 2
	d.deleted += 2
	paramsMust(t, d.insert(b, gparamsRefEntry{key: NumericalSymbol(20), hash: 0, cell: 20}))
	if d.entries[2].cell != 20 || d.entries[2].state != 2 || d.entries[1].state != 1 || d.deleted != 1 {
		t.Fatal(d)
	}
	// Reset retains old payload records and does not shrink cap16.
	retained := d.entries[0].cell
	paramsMust(t, d.reset(b))
	if d.size != 0 || d.deleted != 0 || len(d.entries) != 16 || d.entries[0].cell != retained || d.entries[0].state != 0 {
		t.Fatal(d)
	}
}
func TestGlobalParamsThreeDictionaryLayoutsAndPhysicalBucketOrder(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(gparamsDescriptorEntry{}), reflect.TypeOf(gparamsRefEntry{}), reflect.TypeOf(gparamsDescriptionEntry{}), reflect.TypeOf(GlobalParamCString{})} {
		paramsMust(t, arena.ValidateRecordType(typ))
	}
	b := arena.NewBudget(^uint64(0))
	var d gparamsDescriptorDictionary
	var s gparamsDescriptionDictionary
	paramsMust(t, d.allocate(b, 8))
	paramsMust(t, s.allocate(b, 8))
	defer func() {
		d.close(b)
		s.close(b)
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	// Independent expected placement from first FREE linear buckets, all hash7.
	expected := []uint32{2, 3, 4, 5, 6, 7, 0, 1}
	for i := uint32(1); i <= 7; i++ {
		key := NumericalSymbol(i)
		paramsMust(t, d.insert(b, gparamsDescriptorEntry{key: key, hash: 7, descriptor: i}))
		paramsMust(t, s.insert(b, gparamsDescriptionEntry{key: key, hash: 7, text: GlobalParamCString{Offset: i}}))
	}
	for i, want := range expected {
		if d.entries[i].descriptor != want || s.entries[i].text.Offset != want {
			t.Fatal(i, d.entries[i], s.entries[i])
		}
	}
	paramsMust(t, d.insert(b, gparamsDescriptorEntry{key: NumericalSymbol(8), hash: 7, descriptor: 8}))
	paramsMust(t, s.insert(b, gparamsDescriptionEntry{key: NumericalSymbol(8), hash: 7, text: GlobalParamCString{Offset: 8}}))
	if len(d.entries) != 16 || len(s.entries) != 16 {
		t.Fatal("source growth")
	}
	// Rehash scans old physical order, so new bucket7 starts with old bucket0's2.
	for i := 0; i < 7; i++ {
		want := uint32(i + 2)
		if i == 6 {
			want = 1
		}
		if d.entries[7+i].descriptor != want || s.entries[7+i].text.Offset != want {
			t.Fatal(i, d.entries, s.entries)
		}
	}
}
func TestGlobalParamsDictionaryResetReleaseBeforeFailedShrink(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var d gparamsRefDictionary
	paramsMust(t, d.allocate(b, 64))
	paramsMust(t, d.insert(b, gparamsRefEntry{key: NumericalSymbol(1), hash: 7, cell: 4}))
	b.FailAfter(0)
	e := d.reset(b)
	if !errors.Is(e, arena.ErrInjectedFailure) || len(d.entries) != 0 || b.Used() != 0 {
		t.Fatal(e, d, b.Used())
	}
	// Source pointer/count after allocation failure has no valid native recovery;
	// this adapter exposes a guarded absent backing and permits safe owner Close.
	d.close(b)
}
