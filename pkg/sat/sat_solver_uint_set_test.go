// Licensed under the MIT license.
// sat::uint_set is local char-membership/insertion vectors, not global bitmap.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satSolverSetOwner(t *testing.T) (*satSolverUintSet, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverUintSet(b)
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("set retained%d", b.Used())
		}
	})
	return s, b
}
func TestSATSolverUintSetSourceOrderAndOperations(t *testing.T) {
	s, _ := satSolverSetOwner(t)
	other, _ := satSolverSetOwner(t)
	for _, v := range []uint32{4, 1, 7, 4} {
		paramsMust(t, s.Insert(v))
	}
	if s.Size() != 3 || s.MembershipSize() != 8 || s.MembershipCapacity() != 8 || s.ValuesCapacity() != 3 {
		t.Fatalf("source two vector policies%+v", s.Stats())
	}
	paramsMust(t, s.Remove(1))
	v, e := s.At(1)
	paramsMust(t, e)
	if v != 7 || s.Contains(1) {
		t.Fatal("source swaplast removal")
	}
	paramsMust(t, other.Insert(7))
	paramsMust(t, other.Insert(10))
	paramsMust(t, s.Intersect(other))
	if s.Size() != 1 || s.Contains(4) {
		t.Fatal("source intersection")
	}
	paramsMust(t, s.Union(other))
	v, e = s.At(1)
	paramsMust(t, e)
	if v != 10 {
		t.Fatal("source insertionlist union order")
	}
	v, e = s.Erase()
	paramsMust(t, e)
	if v != 10 || s.Contains(10) {
		t.Fatal("source erase last")
	}
	paramsMust(t, s.Reset())
	if !s.Empty() || s.Contains(7) || s.MembershipSize() != 0 || s.MembershipCapacity() == 0 {
		t.Fatal("source reset retains both capacities")
	}
	paramsMust(t, s.Finalize())
	if s.MembershipCapacity() != 0 || s.ValuesCapacity() != 0 {
		t.Fatal("source finalize frees both arrays")
	}
}
func TestSATSolverUintSetSourceGhostAndRefusalPrefixes(t *testing.T) {
	s, b := satSolverSetOwner(t)
	paramsMust(t, s.ReserveMembership(4))
	b.FailAfter(0)
	e := s.Insert(3)
	if !errors.Is(e, arena.ErrInjectedFailure) || !s.Contains(3) || s.Size() != 0 {
		t.Fatalf("source flag BEFORE push failure %v %+v", e, s.Stats())
	}
	before := b.Used()
	paramsMust(t, s.Insert(3))
	if b.Used() != before || s.Size() != 0 {
		t.Fatal("source duplicate ghost returns without repairing list")
	}
	e = s.Remove(3)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("source assertion after flagclear%v", e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("source actual empty back danger%v", e)
	}
	if s.Contains(3) {
		t.Fatal("source remove flag-clear prefix")
	}
	b.FailAfter(-1)
	paramsMust(t, s.Insert(2))
	before = b.Used()
	b.FailAfter(1)
	e = s.Insert(20)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.MembershipCapacity() != 8 || s.MembershipSize() != 4 || s.Contains(20) {
		t.Fatalf("source first growth then failed next%v %+v", e, s.Stats())
	}
	if b.Used() != before+3 {
		t.Fatalf("actual retained first char growth %d/%d", b.Used(), before)
	}
	b.FailAfter(-1)
}
func TestSATSolverUintSetDeepCopyAssignmentFailureAndWarmAlloc(t *testing.T) {
	s, b := satSolverSetOwner(t)
	d, db := satSolverSetOwner(t)
	for _, v := range []uint32{5, 2, 9} {
		paramsMust(t, s.Insert(v))
	}
	paramsMust(t, d.Insert(1))
	paramsMust(t, d.Assign(s))
	paramsMust(t, s.Remove(2))
	if !d.Contains(2) {
		t.Fatal("source assign deep copy")
	}
	before := db.Used()
	db.FailAfter(0)
	e := d.Assign(s)
	if !errors.Is(e, arena.ErrInjectedFailure) || d.MembershipCapacity() != 0 || d.Size() != 3 {
		t.Fatalf("source membership destroy before copy failure%v", e)
	}
	if db.Used() >= before {
		t.Fatal("source destroyed membership charge remains")
	}
	db.FailAfter(-1)
	paramsMust(t, d.Assign(s))
	clone, e := s.Clone(nil)
	paramsMust(t, e)
	defer clone.Close()
	if clone.MembershipCapacity() != s.MembershipCapacity() || clone.ValuesCapacity() != s.ValuesCapacity() {
		t.Fatal("source exact copy capacities")
	}
	before = b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		if e := s.Insert(5); e != nil {
			panic(e)
		}
	})
	if allocs != 0 || b.Used() != before {
		t.Fatal("warm duplicate allocations")
	}
	b.FailAfter(-1)
}
func TestSATSolverModelConverterActualCollectVars(t *testing.T) {
	c, _ := satConverterTestOwner(t)
	s, b := satSolverSetOwner(t)
	paramsMust(t, s.Insert(2))
	for _, v := range []BoolVar{2, 8, 1} {
		_, e := c.Mk(satModelBlockLit, v)
		paramsMust(t, e)
	}
	paramsMust(t, c.CollectVars(s))
	want := []uint32{2, 8, 1}
	for i, v := range want {
		got, e := s.At(uint32(i))
		paramsMust(t, e)
		if got != v {
			t.Fatal("actual source traversal and prepopulated dedup")
		}
	}
	paramsMust(t, s.Finalize())
	paramsMust(t, s.ReserveMembership(3))
	b.FailAfter(0)
	e := c.CollectVars(s)
	if !errors.Is(e, arena.ErrInjectedFailure) || !s.Contains(2) || s.Size() != 0 {
		t.Fatalf("actual collect_vars flag before list allocation%v", e)
	}
	b.FailAfter(-1)
	paramsMust(t, s.Reset())
	paramsMust(t, s.Insert(0))
	paramsMust(t, s.Insert(2))
	paramsMust(t, s.Insert(1))
	b.FailAfter(0)
	e = c.CollectVars(s)
	if !errors.Is(e, arena.ErrInjectedFailure) || !s.Contains(2) || s.Contains(8) {
		t.Fatalf("source CollectVars completed earlier duplicate then growth refusal%v", e)
	}
	b.FailAfter(-1)
}
func TestSATSolverUintSetBoundsConstructorAndIndependentModel(t *testing.T) {
	b := arena.NewBudget(0)
	if _, e := newSATSolverUintSet(b); !errors.Is(e, arena.ErrBudget) || b.Used() != 0 {
		t.Fatal("actual constructor budget")
	}
	s, b := satSolverSetOwner(t)
	model := map[uint32]bool{}
	state := uint32(1)
	for i := 0; i < 1000; i++ {
		state = state*1664525 + 1013904223
		v := (state >> 8) % 37
		if state&1 != 0 {
			paramsMust(t, s.Insert(v))
			model[v] = true
		} else {
			paramsMust(t, s.Remove(v))
			delete(model, v)
		}
		for n := uint32(0); n < 37; n++ {
			if s.Contains(n) != model[n] {
				t.Fatalf("independent set membership step%d value%d", i, n)
			}
		}
		if s.Size() != uint32(len(model)) {
			t.Fatal("independent size")
		}
	}
	want := uint64(reflect.TypeOf(satSolverUintSet{}).Size()) + s.MembershipCapacity() + 4*s.ValuesCapacity()
	if b.Used() != want {
		t.Fatalf("actual headers/backings %d/%d", b.Used(), want)
	}
	e := s.Insert(^uint32(0))
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("source wrapped size debug%v", e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("source wrapped actual access%v", e)
	}
}

var satSolverEscapedSetIterator satSolverUintSetIterator

func TestSATSolverUintSetValueIteratorRetainedLifetime(t *testing.T) {
	s, b := satSolverSetOwner(t)
	for _, v := range []uint32{0, 1} {
		paramsMust(t, s.Insert(v))
	}
	var factory func() (satSolverUintSetIterator, error) = s.Iter
	it, e := factory()
	paramsMust(t, e)
	paramsMust(t, s.Reset())
	// The source iterator is a pointer into retained svector<unsigned> data;
	// Reset changes logical size, not scalar lifetime/backing or captured end.
	for _, want := range []uint32{0, 1} {
		v, ok, e := it.Next()
		paramsMust(t, e)
		if !ok || v != want {
			t.Fatal("source retained scalar iterator")
		}
	}
	_, ok, e := it.Next()
	paramsMust(t, e)
	if ok {
		t.Fatal("source captured end")
	}
	paramsMust(t, s.Insert(2))
	paramsMust(t, s.Insert(3))
	before := b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		v, e := s.Iter()
		if e != nil {
			panic(e)
		}
		satSolverEscapedSetIterator = v
	})
	if allocs != 0 || b.Used() != before {
		t.Fatalf("escaping value iterator allocs%v delta%d", allocs, b.Used()-before)
	}
	b.FailAfter(-1)
	paramsMust(t, s.Insert(4))
	_, _, e = satSolverEscapedSetIterator.Next()
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("actual backing relocation must invalidate pointer adapter %v", e)
	}
}
func TestSATSolverUintSetGhostDefinedSpareTailRelease(t *testing.T) {
	s, b := satSolverSetOwner(t)
	paramsMust(t, s.ReserveMembership(8))
	paramsMust(t, s.Insert(0))
	paramsMust(t, s.Insert(1))
	b.FailAfter(0)
	e := s.Insert(3)
	if !errors.Is(e, arena.ErrInjectedFailure) || !s.Contains(3) {
		t.Fatal("source ghost fixture")
	}
	b.FailAfter(-1)
	paramsMust(t, s.Insert(5))
	paramsMust(t, s.Insert(6))
	if s.Size() != 4 || s.ValuesCapacity() != 5 {
		t.Fatal("source defined spare tail fixture")
	}
	e = s.Remove(3)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || s.Size() != 4 {
			t.Fatalf("debug assertion before tail write%v", e)
		}
	} else {
		paramsMust(t, e)
		if s.Size() != 3 || !s.Contains(6) {
			t.Fatal("source defined ghost-tail write/pop side effects")
		}
		v, e := s.At(4)
		paramsMust(t, e)
		if v != 6 {
			t.Fatal("source retained spare tail assignment")
		}
	}
	if s.Contains(3) {
		t.Fatal("membership clear precedes source diagnostic/write")
	}
}

func TestSATSolverUintSetSourceCharCapacityOverflowMath(t *testing.T) {
	old := uint32(0)
	for i := 0; i < 128; i++ {
		want := uint32(2)
		if old != 0 {
			want = (3*old + 1) >> 1
		}
		overflow := want <= old || want+8 <= old+8
		got, e := satSolverMembershipNextCapacity(uint64(old))
		if overflow {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatalf("source uint32 char growth old%d wrappednext%d: got%d error%v", old, want, got, e)
			}
			return
		}
		if uint64(want) > uint64(^uint(0)>>1) {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("actual host backing bounds")
			}
			return
		}
		paramsMust(t, e)
		if got != uint64(want) {
			t.Fatalf("source growth %d/%d", got, want)
		}
		old = want
	}
	t.Fatal("source finite uint32 growth did not overflow")
}
