package ast

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func familyFixture(t *testing.T, b *arena.Budget) (*usymbol.Manager, *FamilyManager) {
	t.Helper()
	m, e := usymbol.NewManager(usymbol.Config{})
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
		if e != nil || s != (usymbol.Symbol{}) {
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
		if _, e := f.MkID(usymbol.Numerical(k)); e != nil {
			t.Fatal(e)
		}
	}
	dom, e := f.AppendDom([]usymbol.Symbol{usymbol.Numerical(99)})
	if e != nil || !reflect.DeepEqual(dom, []usymbol.Symbol{usymbol.Numerical(99), usymbol.Numerical(6), usymbol.Numerical(1), usymbol.Numerical(4)}) {
		t.Fatal(dom, e)
	}
	rangeIDs, e := f.AppendRange([]FamilyID{-99})
	if e != nil || !reflect.DeepEqual(rangeIDs, []FamilyID{-99, 0, 1, 2}) {
		t.Fatal(rangeIDs, e)
	}
}

func TestFamilyBudgetConstructorAndNativeNameGrowth(t *testing.T) {
	m, e := usymbol.NewManager(usymbol.Config{})
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	for prefix := int64(0); prefix < 1; prefix++ {
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
	for i := uint32(0); i < 9; i++ {
		if _, e := f.MkID(usymbol.Numerical(uint32(i))); e != nil {
			t.Fatal(e)
		}
		s := f.Stats()
		if s.Names != i+1 || s.TableSize != i+1 || b.Used() != s.OwnerBytes {
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
	if _, e := f.MkID(usymbol.Symbol{}); !errors.Is(e, usymbol.ErrInvalid) {
		t.Fatal(e)
	}
	if _, e := f.MkID(usymbol.Dummy()); !errors.Is(e, usymbol.ErrInvalid) {
		t.Fatal(e)
	}
	other, e := usymbol.NewManager(usymbol.Config{})
	if e != nil {
		t.Fatal(e)
	}
	foreign, e := other.Intern("basic")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := f.GetID(foreign); !errors.Is(e, usymbol.ErrOwner) {
		t.Fatal(e)
	}
	other.Close()
	f.next = math.MaxInt32
	if _, e := f.MkID(usymbol.Numerical(88)); !errors.Is(e, arena.ErrBounds) || f.next != math.MaxInt32 {
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

func TestFamilyBorrowedSymbolManagerLifetime(t *testing.T) {
	m, e := usymbol.NewManager(usymbol.Config{})
	if e != nil {
		t.Fatal(e)
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
