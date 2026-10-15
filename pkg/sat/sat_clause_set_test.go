package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATClauseSetNativeCollections(t *testing.T) {
	a := satStore(t, nil)
	b := arena.NewBudget(1 << 20)
	s, e := NewSATClauseSet(a, b)
	satOK(t, e)
	defer s.Close()
	r1 := satMk(t, a, satLs(0, 2), false)
	r2 := satMk(t, a, satLs(2, 4), false)
	r3 := satMk(t, a, satLs(4, 6), false)
	satOK(t, s.Insert(r1))
	satOK(t, s.Insert(r2))
	satOK(t, s.Insert(r1))
	if s.Size() != 2 {
		t.Fatal(s.Size())
	}
	satOK(t, s.Erase(r1))
	got, e := s.Get(0)
	if got != r2 || e != nil {
		t.Fatal(got, e)
	}
	satOK(t, s.Insert(r1))
	// Native map/slice growth does not consume a simulated C++ vector budget.
	b.FailAfter(0)
	satOK(t, s.Insert(r3))
	if has, e := s.Contains(r3); !has || e != nil || s.Size() != 3 {
		t.Fatal(has, e, s.Size())
	}
	if ok, e := s.CheckInvariant(); !ok || e != nil {
		t.Fatal(ok, e)
	}
	satOK(t, s.Erase(r3))
	if has, e := s.Contains(r3); has || e != nil {
		t.Fatal(has, e)
	}
	p, e := s.Pop()
	if p != r1 || e != nil {
		t.Fatal(p, e)
	}
	satOK(t, s.Reset())
	satOK(t, s.Finalize())
	satOK(t, s.Insert(r2))
	s.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestSATClauseSetCopyMoveAndHeaderFailure(t *testing.T) {
	a := satStore(t, nil)
	b := arena.NewBudget(1 << 20)
	src, e := NewSATClauseSet(a, b)
	satOK(t, e)
	defer src.Close()
	r := satMk(t, a, satLs(0, 2), false)
	satOK(t, src.Insert(r))
	dst, e := NewSATClauseSet(a, b)
	satOK(t, e)
	defer dst.Close()
	b.FailAfter(0)
	if _, e = src.Clone(b); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("clone must charge its owner header", e)
	}
	// Assignment copies native collections and needs no extra source allocation.
	satOK(t, dst.Assign(src))
	b.FailAfter(-1)
	copy, e := src.Clone(b)
	satOK(t, e)
	defer copy.Close()
	satOK(t, dst.MoveAssign(copy))
	if copy.Size() != 1 || dst.Size() != 1 {
		t.Fatal("move assignment copies the set")
	}
	moved, e := copy.MoveClone(b)
	satOK(t, e)
	defer moved.Close()
	if copy.Size() != 0 || moved.Size() != 1 {
		t.Fatal("move constructor")
	}
	satOK(t, a.Close())
	raw, e := src.Clone(b)
	satOK(t, e)
	defer raw.Close()
	if got, e := raw.Get(0); got != r || e != nil {
		t.Fatal("copy must not dereference a dead borrowed clause", got, e)
	}
}
