// Licensed under the MIT license.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
		paramsMust(t, d.insert(b, GparamsRefEntry{key: NumericalSymbol(i), hash: 0, cell: i}))
	}
	old := b.Used()
	b.FailAfter(0)
	e := d.insert(b, GparamsRefEntry{key: NumericalSymbol(1), hash: 0, cell: 99})
	if !errors.Is(e, arena.ErrInjectedFailure) || len(d.Entries) != 8 || b.Used() != old || d.Entries[0].cell != 1 {
		t.Fatal("growth precedes overwrite", e, d, b.Used())
	}
	b.FailAfter(-1)
	paramsMust(t, d.insert(b, GparamsRefEntry{key: NumericalSymbol(1), hash: 0, cell: 99}))
	if len(d.Entries) != 16 || d.size != 7 || d.Entries[0].cell != 99 {
		t.Fatal(d)
	}
	// Probe remembers LAST deleted bucket, not first, before the first FREE.
	d.Entries[1].state = 1
	d.Entries[2].state = 1
	d.size -= 2
	d.deleted += 2
	paramsMust(t, d.insert(b, GparamsRefEntry{key: NumericalSymbol(20), hash: 0, cell: 20}))
	if d.Entries[2].cell != 20 || d.Entries[2].state != 2 || d.Entries[1].state != 1 || d.deleted != 1 {
		t.Fatal(d)
	}
	// Reset retains old payload records and does not shrink cap16.
	retained := d.Entries[0].cell
	paramsMust(t, d.reset(b))
	if d.size != 0 || d.deleted != 0 || len(d.Entries) != 16 || d.Entries[0].cell != retained || d.Entries[0].state != 0 {
		t.Fatal(d)
	}
}
func TestGlobalParamsThreeDictionaryLayoutsAndPhysicalBucketOrder(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(GparamsDescriptorEntry{}), reflect.TypeOf(GparamsRefEntry{}), reflect.TypeOf(GparamsDescriptionEntry{}), reflect.TypeOf(GlobalParamCString{})} {
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
		paramsMust(t, d.insert(b, GparamsDescriptorEntry{key: key, hash: 7, descriptor: i}))
		paramsMust(t, s.insert(b, GparamsDescriptionEntry{key: key, hash: 7, text: GlobalParamCString{Offset: i}}))
	}
	for i, want := range expected {
		if d.Entries[i].descriptor != want || s.Entries[i].text.Offset != want {
			t.Fatal(i, d.Entries[i], s.Entries[i])
		}
	}
	paramsMust(t, d.insert(b, GparamsDescriptorEntry{key: NumericalSymbol(8), hash: 7, descriptor: 8}))
	paramsMust(t, s.insert(b, GparamsDescriptionEntry{key: NumericalSymbol(8), hash: 7, text: GlobalParamCString{Offset: 8}}))
	if len(d.Entries) != 16 || len(s.Entries) != 16 {
		t.Fatal("source growth")
	}
	// Rehash scans old physical order, so new bucket7 starts with old bucket0's2.
	for i := 0; i < 7; i++ {
		want := uint32(i + 2)
		if i == 6 {
			want = 1
		}
		if d.Entries[7+i].descriptor != want || s.Entries[7+i].text.Offset != want {
			t.Fatal(i, d.Entries, s.Entries)
		}
	}
}
func TestGlobalParamsDictionaryResetReleaseBeforeFailedShrink(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var d gparamsRefDictionary
	paramsMust(t, d.allocate(b, 64))
	paramsMust(t, d.insert(b, GparamsRefEntry{key: NumericalSymbol(1), hash: 7, cell: 4}))
	b.FailAfter(0)
	e := d.reset(b)
	if !errors.Is(e, arena.ErrInjectedFailure) || len(d.Entries) != 0 || b.Used() != 0 {
		t.Fatal(e, d, b.Used())
	}
	// Source pointer/count after allocation failure has no valid native recovery;
	// this adapter exposes a guarded absent backing and permits safe owner Close.
	d.close(b)
}
