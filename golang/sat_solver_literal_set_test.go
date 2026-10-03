package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func literalSetTestOwner(t *testing.T) (*satSolverLiteralSet, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverLiteralSet(b)
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("literal set leaked", b.Used())
		}
	})
	return s, b
}
func literalSetActualCharges(t *testing.T, s *satSolverLiteralSet, b *arena.Budget) {
	t.Helper()
	want := uint64(reflect.TypeOf(*s).Size()) + uint64(reflect.TypeOf(*s.set).Size()) + uint64(cap(s.set.inSet)) + uint64(cap(s.set.values))*4
	if b.Used() != want {
		t.Fatalf("physical charge %d want %d", b.Used(), want)
	}
}
func literalSetValues(t *testing.T, s *satSolverLiteralSet) []satLiteral {
	t.Helper()
	it, e := s.Iter()
	paramsMust(t, e)
	var result []satLiteral
	for {
		v, ok, e := it.Next()
		paramsMust(t, e)
		if !ok {
			return result
		}
		result = append(result, v)
	}
}
func TestSATSolverLiteralSetSourceVectorRoutesAndIndependence(t *testing.T) {
	store, _ := satConverterTestOwner(t)
	r, e := store.store.NewList()
	paramsMust(t, e)
	for _, v := range []satLiteral{3, 2, 3, 7, 0, 2} {
		paramsMust(t, store.store.Push(r, v))
	}
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverLiteralSetFromVector(b, store.store, r)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	if got := literalSetValues(t, s); !reflect.DeepEqual(got, []satLiteral{3, 2, 7, 0}) {
		t.Fatal(got)
	}
	literalSetActualCharges(t, s, b)
	paramsMust(t, store.store.Write(r, 0, 9))
	if !s.Contains(3) || s.Contains(9) {
		t.Fatal("borrowed input was retained")
	}
	out, e := s.ToVector(store.store)
	paramsMust(t, e)
	for i, v := range []satLiteral{3, 2, 7, 0} {
		got, e := store.store.Read(out, uint32(i))
		paramsMust(t, e)
		if got != v {
			t.Fatal(got)
		}
	}
	capacity, e := store.store.Capacity(out)
	paramsMust(t, e)
	if capacity != 5 {
		t.Fatal("result must push geometrically", capacity)
	}
	paramsMust(t, s.AssignVector(store.store, r))
	if got := literalSetValues(t, s); !reflect.DeepEqual(got, []satLiteral{9, 2, 3, 7, 0}) {
		t.Fatal(got)
	}
	paramsMust(t, store.Close())
	if !s.Contains(9) {
		t.Fatal("result depends on closed input owner")
	}
}
func TestSATSolverLiteralSetIndependentOrderedOracle(t *testing.T) {
	s, b := literalSetTestOwner(t)
	var member [32]bool
	var order []satLiteral
	rng := uint32(19)
	for step := 0; step < 1500; step++ {
		rng = rng*1664525 + 1013904223
		v := satLiteral(rng >> 16 & 31)
		switch rng & 3 {
		case 0, 1:
			paramsMust(t, s.Insert(v))
			if !member[v] {
				member[v] = true
				order = append(order, v)
			}
		case 2:
			paramsMust(t, s.Remove(v))
			if member[v] {
				member[v] = false
				for i, x := range order {
					if x == v {
						order[i] = order[len(order)-1]
						order = order[:len(order)-1]
						break
					}
				}
			}
		case 3:
			if len(order) > 0 {
				got, e := s.Pop()
				paramsMust(t, e)
				want := order[len(order)-1]
				if got != want {
					t.Fatal(step, got, want)
				}
				member[want] = false
				order = order[:len(order)-1]
			}
		}
		if s.Size() != uint32(len(order)) || s.Empty() != (len(order) == 0) {
			t.Fatal(step)
		}
		got := literalSetValues(t, s)
		if len(got) != len(order) {
			t.Fatal(step, got, order)
		}
		for i := range order {
			if got[i] != order[i] {
				t.Fatal(step, got, order)
			}
		}
		for i := range member {
			if s.Contains(satLiteral(i)) != member[i] {
				t.Fatal(step, i)
			}
		}
		literalSetActualCharges(t, s, b)
	}
}
func TestSATSolverLiteralSetCopyUnionIntersectionAndRetainedIterator(t *testing.T) {
	s, b := literalSetTestOwner(t)
	other, _ := literalSetTestOwner(t)
	for _, v := range []satLiteral{3, 1, 6} {
		paramsMust(t, s.Insert(v))
	}
	for _, v := range []satLiteral{6, 4, 3} {
		paramsMust(t, other.Insert(v))
	}
	cb := arena.NewBudget(^uint64(0))
	clone, e := s.Clone(cb)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, clone.Close())
		if cb.Used() != 0 {
			t.Fatal(cb.Used())
		}
	}()
	if cap(clone.set.inSet) != cap(s.set.inSet) || cap(clone.set.values) != cap(s.set.values) {
		t.Fatal("copy lost source capacity")
	}
	paramsMust(t, s.Union(other))
	if got := literalSetValues(t, s); !reflect.DeepEqual(got, []satLiteral{3, 1, 6, 4}) {
		t.Fatal(got)
	}
	paramsMust(t, s.Intersect(other))
	if got := literalSetValues(t, s); !reflect.DeepEqual(got, []satLiteral{3, 6, 4}) {
		t.Fatal(got)
	}
	it, e := s.Iter()
	paramsMust(t, e)
	before := b.Used()
	paramsMust(t, s.Reset())
	if !s.Empty() || s.Contains(3) || b.Used() != before {
		t.Fatal("reset did not retain storage")
	}
	v, ok, e := it.Next()
	paramsMust(t, e)
	if !ok || v != 3 {
		t.Fatal("raw iterator lost retained backing", v, ok)
	}
	paramsMust(t, s.Assign(clone))
	if got := literalSetValues(t, s); !reflect.DeepEqual(got, []satLiteral{3, 1, 6}) {
		t.Fatal(got)
	}
	if _, _, e = it.Next(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("assignment must replace backing", e)
	}
	paramsMust(t, s.Finalize())
	literalSetActualCharges(t, s, b)
}
func TestSATSolverLiteralSetSourceFlagBeforeListRefusalAndResetBeforeInputRead(t *testing.T) {
	s, b := literalSetTestOwner(t)
	paramsMust(t, s.set.ReserveMembership(8))
	b.FailAfter(0)
	if e := s.Insert(3); !errors.Is(e, arena.ErrInjectedFailure) || !s.Contains(3) || !s.Empty() {
		t.Fatal("source ghost prefix", e)
	}
	literalSetActualCharges(t, s, b)
	e := s.Remove(3)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if s.Contains(3) {
		t.Fatal("remove must clear flag before missing-list guard")
	}
	b.FailAfter(-1)
	paramsMust(t, s.Insert(1))
	store, _ := satConverterTestOwner(t)
	r, e := store.store.NewList()
	paramsMust(t, e)
	paramsMust(t, store.store.Free(r))
	if e = s.AssignVector(store.store, r); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if !s.Empty() || s.Contains(1) {
		t.Fatal("assignment must reset before input access")
	}
}
func TestSATSolverLiteralSetConstructorAndResultFailureOwnership(t *testing.T) {
	store, _ := satConverterTestOwner(t)
	r, e := store.store.NewList()
	paramsMust(t, e)
	for _, v := range []satLiteral{3, 11, 4, 21} {
		paramsMust(t, store.store.Push(r, v))
	}
	success := false
	for fail := int64(0); fail < 32; fail++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(fail)
		s, e := newSATSolverLiteralSetFromVector(b, store.store, r)
		if e == nil {
			success = true
			literalSetActualCharges(t, s, b)
			paramsMust(t, s.Close())
		} else if !errors.Is(e, arena.ErrInjectedFailure) || s != nil {
			t.Fatal(fail, e)
		}
		if b.Used() != 0 {
			t.Fatal("unpublished constructor retained roots", fail, b.Used())
		}
	}
	if !success {
		t.Fatal("no constructor success")
	}
	s, _ := literalSetTestOwner(t)
	for _, v := range []satLiteral{3, 11, 4, 21} {
		paramsMust(t, s.Insert(v))
	}
	for fail := int64(0); fail < 25; fail++ {
		b := arena.NewBudget(^uint64(0))
		out, e := newSATSolverLiteralStore(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		paramsMust(t, e)
		b.FailAfter(fail)
		result, e := s.ToVector(out)
		if e == nil {
			paramsMust(t, out.Free(result))
		} else {
			if !errors.Is(e, arena.ErrInjectedFailure) || result != (satSolverLiteralListRef{}) {
				t.Fatal(fail, e)
			}
			for _, record := range out.records {
				if record.live {
					t.Fatal("failed lexical result retained a root", fail)
				}
			}
		}
		w := out.words.Stats()
		d := w.Descriptors
		want := uint64(reflect.TypeOf(*out).Size()) + uint64(cap(out.records))*uint64(reflect.TypeOf(satSolverLiteralListRecord{}).Size()) + w.OwnerBytes + w.PayloadBytes + w.BlockMetadataBytes + w.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
		if want != b.Used() {
			t.Fatal("result physical accounting", fail, want, b.Used())
		}
		paramsMust(t, out.Close())
		if b.Used() != 0 {
			t.Fatal(fail, b.Used())
		}
	}
}
func TestSATSolverLiteralSetWarmOperationsAndColdOwnerGuards(t *testing.T) {
	s, b := literalSetTestOwner(t)
	paramsMust(t, s.Insert(1))
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := s.Insert(1); e != nil {
			panic(e)
		}
		it, e := s.Iter()
		if e != nil {
			panic(e)
		}
		if _, ok, e := it.Next(); e != nil || !ok {
			panic("iterator")
		}
		if e = s.Assign(s); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	literalSetActualCharges(t, s, b)
	copied := *s
	if e := copied.Insert(2); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	paramsMust(t, s.Close())
	if e := s.Insert(1); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestSATSolverLiteralSetActualIteratorOperators(t *testing.T) {
	s, _ := literalSetTestOwner(t)
	other, _ := literalSetTestOwner(t)
	begin, e := s.Begin()
	paramsMust(t, e)
	end, e := other.End()
	paramsMust(t, e)
	if !begin.Equal(end) {
		t.Fatal("both source null backing pointers must compare equal")
	}
	for _, v := range []satLiteral{2, 5, 3} {
		paramsMust(t, s.Insert(v))
	}
	begin, e = s.Begin()
	paramsMust(t, e)
	end, e = s.End()
	paramsMust(t, e)
	if begin.Equal(end) {
		t.Fatal("begin equals nonempty end")
	}
	previous, e := begin.PostAdvance()
	paramsMust(t, e)
	v, e := previous.Value()
	paramsMust(t, e)
	if v != 2 {
		t.Fatal(v)
	}
	v, e = begin.Value()
	paramsMust(t, e)
	if v != 5 {
		t.Fatal(v)
	}
	paramsMust(t, begin.Advance())
	paramsMust(t, begin.Advance())
	if !begin.Equal(end) {
		t.Fatal("increment did not reach captured end")
	}
	// Source equality only copies/compares raw pointer values. Freeing backing
	// does not introduce a dereference in equality; Value guards that access.
	paramsMust(t, s.Finalize())
	if !begin.Equal(end) {
		t.Fatal("scalar pointer equality added a dereference")
	}
	if _, e = previous.Value(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
}
