// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func familyFixture(t *testing.T, b *arena.Budget) (*SymbolManager, *FamilyManager) {
	t.Helper()
	m, e := NewSymbolManager(SymbolConfig{})
	if e != nil {
		t.Fatal(e)
	}
	f, e := NewFamilyManager(m, b)
	if e != nil {
		m.Close()
		t.Fatal(e)
	}
	t.Cleanup(func() { f.Close(); m.Close() })
	return m, f
}

// These are added assertions on real family_manager routes. The source has no
// standalone family assertion test. This ordering is ast_manager::init's actual
// registration sequence, not a simulation of its manager or plugin construction.
func TestFamilySourceRegistrationAndQueries(t *testing.T) {
	m, f := familyFixture(t, nil)
	if f.Stats().NextID != 0 || f.Stats().Names != 0 {
		t.Fatal(f.Stats())
	}
	for i, name := range []string{"basic", "label", "pattern", "model-value", "user-sort", "arith"} {
		s, e := m.Intern(name)
		if e != nil {
			t.Fatal(e)
		}
		before, e := f.GetID(s)
		if e != nil || before != NullFamilyID {
			t.Fatal(before, e)
		}
		id, e := f.MkID(s)
		if e != nil || id != FamilyID(i) {
			t.Fatal(id, e)
		}
		again, e := f.MkID(s)
		if e != nil || again != id {
			t.Fatal(again, e)
		}
		got, e := f.GetName(id)
		if e != nil || got != s {
			t.Fatal(got, e)
		}
		has, e := f.HasID(id)
		if e != nil || !has {
			t.Fatal(has, e)
		}
		has, e = f.HasSymbol(s)
		if e != nil || !has {
			t.Fatal(has, e)
		}
	}
	for _, id := range []FamilyID{-1, -2, 6, math.MaxInt32} {
		s, e := f.GetName(id)
		if e != nil || s != (Symbol{}) {
			t.Fatal(s, e)
		}
		has, e := f.HasID(id)
		if e != nil || has {
			t.Fatal(has, e)
		}
	}
}

func TestFamilySourceEnumerationOrderAndPrefixes(t *testing.T) {
	_, f := familyFixture(t, nil)
	for _, k := range []uint32{6, 1, 4} {
		if _, e := f.MkID(NumericalSymbol(k)); e != nil {
			t.Fatal(e)
		}
	}
	dom, e := f.AppendDom([]Symbol{NumericalSymbol(99)})
	if e != nil || !reflect.DeepEqual(dom, []Symbol{NumericalSymbol(99), NumericalSymbol(1), NumericalSymbol(4), NumericalSymbol(6)}) {
		t.Fatal(dom, e)
	}
	rangeIDs, e := f.AppendRange([]FamilyID{-99})
	if e != nil || !reflect.DeepEqual(rangeIDs, []FamilyID{-99, 1, 2, 0}) {
		t.Fatal(rangeIDs, e)
	}
}

func TestFamilyNameFailureRetainsPublishedMap(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	_, f := familyFixture(t, b)
	for i := uint32(0); i < 2; i++ {
		if _, e := f.MkID(NumericalSymbol(i)); e != nil {
			t.Fatal(e)
		}
	}
	before := b.Used()
	b.FailAfter(0)
	if _, e := f.MkID(NumericalSymbol(2)); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	// Source increments next, inserts mapping, then attempts names.push_back.
	s := f.Stats()
	if s.NextID != 3 || s.Names != 2 || s.Table.Size != 3 || b.Used() != before {
		t.Fatal(s, b.Used(), before)
	}
	id, e := f.GetID(NumericalSymbol(2))
	if e != nil || id != 2 {
		t.Fatal(id, e)
	}
	name, e := f.GetName(2)
	if e != nil || name != (Symbol{}) {
		t.Fatal(name, e)
	}
	has, e := f.HasID(2)
	if e != nil || has {
		t.Fatal(has, e)
	}
	id, e = f.MkID(NumericalSymbol(2))
	if e != nil || id != 2 || f.Stats().Names != 2 {
		t.Fatal(id, e)
	}
	b.FailAfter(-1)
	f.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFamilyMapFailureRetainsNextIDAndRetryGap(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	_, f := familyFixture(t, b)
	for i := uint32(0); i < 7; i++ {
		if _, e := f.MkID(NumericalSymbol(i)); e != nil {
			t.Fatal(e)
		}
	}
	before := b.Used()
	b.FailAfter(0)
	if _, e := f.MkID(NumericalSymbol(7)); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	s := f.Stats()
	if s.NextID != 8 || s.Names != 7 || s.Table.Size != 7 || b.Used() != before {
		t.Fatal(s, b.Used(), before)
	}
	id, e := f.GetID(NumericalSymbol(7))
	if e != nil || id != NullFamilyID {
		t.Fatal(id, e)
	}
	b.FailAfter(-1)
	id, e = f.MkID(NumericalSymbol(7))
	if e != nil || id != 8 {
		t.Fatal(id, e)
	}
	// Source failure is not transactional: appended position7 need not equal ID8.
	name, e := f.GetName(7)
	if e != nil || name != NumericalSymbol(7) {
		t.Fatal(name, e)
	}
	has, e := f.HasID(8)
	if e != nil || has {
		t.Fatal(has, e)
	}
	f.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFamilyBudgetConstructorAndNameGrowth(t *testing.T) {
	m, e := NewSymbolManager(SymbolConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	for prefix := int64(0); prefix < 3; prefix++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(prefix)
		f, e := NewFamilyManager(m, b)
		if !errors.Is(e, arena.ErrInjectedFailure) || f != nil || b.Used() != 0 {
			t.Fatal(prefix, f, e, b.Used())
		}
	}
	b := arena.NewBudget(1 << 20)
	f, e := NewFamilyManager(m, b)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	for i, capacity := range []uint32{2, 2, 3, 5, 5, 8, 8, 8, 12} {
		if _, e := f.MkID(NumericalSymbol(uint32(i))); e != nil {
			t.Fatal(e)
		}
		s := f.Stats()
		if s.NamesCapacity != capacity || s.NamesBytes != uint64(capacity)*uint64(reflect.TypeOf(Symbol{}).Size()) || b.Used() != s.OwnerBytes+s.NamesBytes+s.Table.OwnerBytes+s.Table.TableBytes {
			t.Fatal(s, b.Used())
		}
	}
	f.Close()
	f.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFamilyWarmGuardsAndSignedCounterBound(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m, f := familyFixture(t, b)
	s, e := m.Intern("basic")
	if e != nil {
		t.Fatal(e)
	}
	id, e := f.MkID(s)
	if e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	var failure error
	if n := testing.AllocsPerRun(100, func() {
		got, e := f.MkID(s)
		if e != nil {
			failure = e
		}
		if got != id {
			t.Fatal(got)
		}
	}); n != 0 || failure != nil {
		t.Fatal(n, failure)
	}
	b.FailAfter(-1)
	// Reconstruct a copied cold identity without copying a live mutex.
	copyOwner := FamilyManager{self: f, symbols: m, budget: b, table: f.table, names: f.names, next: f.next}
	if _, e := copyOwner.MkID(s); !errors.Is(e, ErrFamilyOwner) {
		t.Fatal(e)
	}
	if _, e := f.MkID(Symbol{}); !errors.Is(e, ErrInvalidSymbol) {
		t.Fatal(e)
	}
	if _, e := f.MkID(DummySymbol()); !errors.Is(e, ErrInvalidSymbol) {
		t.Fatal(e)
	}
	other, e := NewSymbolManager(SymbolConfig{})
	if e != nil {
		t.Fatal(e)
	}
	foreign, e := other.Intern("basic")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := f.GetID(foreign); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal(e)
	}
	other.Close()
	f.next = math.MaxInt32
	if _, e := f.MkID(NumericalSymbol(88)); !errors.Is(e, arena.ErrBounds) || f.next != math.MaxInt32 {
		t.Fatal(e, f.next)
	}
	f.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	if _, e := f.GetName(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	var nilOwner *FamilyManager
	if _, e := nilOwner.GetID(s); !errors.Is(e, ErrFamilyOwner) {
		t.Fatal(e)
	}
}

func TestFamilyExactNameGrowthOverlapAndBorrowedLifetime(t *testing.T) {
	m, e := NewSymbolManager(SymbolConfig{})
	if e != nil {
		t.Fatal(e)
	}
	width := uint64(reflect.TypeOf(Symbol{}).Size())
	base := uint64(reflect.TypeOf(FamilyManager{}).Size()) + uint64(reflect.TypeOf(SymbolInt32Table{}).Size()) + symbolInt32EntryBytes(8)
	for _, delta := range []uint64{0, 1} {
		b := arena.NewBudget(base + 5*width - delta)
		f, e := NewFamilyManager(m, b)
		if e != nil {
			t.Fatal(e)
		}
		for i := uint32(0); i < 2; i++ {
			if _, e = f.MkID(NumericalSymbol(i)); e != nil {
				t.Fatal(e)
			}
		}
		_, e = f.MkID(NumericalSymbol(2))
		if delta == 1 {
			if !errors.Is(e, arena.ErrBudget) || b.Used() != base+2*width || f.Stats().Names != 2 || f.Stats().Table.Size != 3 {
				t.Fatal(e, f.Stats(), b.Used())
			}
		} else if e != nil || b.Used() != base+3*width || f.Stats().Names != 3 {
			t.Fatal(e, f.Stats(), b.Used())
		}
		f.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	f, e := NewFamilyManager(m, nil)
	if e != nil {
		t.Fatal(e)
	}
	m.Close()
	if _, e = f.GetName(-1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
}
