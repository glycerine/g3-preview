package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestSATClauseSetSourceOrderAndGhostPrefixes(t *testing.T) {
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
	satOK(t, s.reserveIDs(3)) // Warm the source ID vector independently; fail m_set push.
	b.FailAfter(0)
	e = s.Insert(r3)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if has, e := s.Contains(r3); !has || e != nil {
		t.Fatal("position prefix", has, e)
	}
	if s.Size() != 2 {
		t.Fatal(s.Size())
	}
	b.FailAfter(-1)
	satOK(t, s.Insert(r3))
	if s.Size() != 2 {
		t.Fatal("retry sees ghost")
	}
	if e = s.Erase(r3); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if has, _ := s.Contains(r3); has {
		t.Fatal("erase must retain sentinel clear before OOB")
	}
	p, e := s.Pop()
	if p != r1 || e != nil {
		t.Fatal(p, e)
	}
	satOK(t, s.Reset())
	satOK(t, s.Finalize())
	s.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATClauseSetActualCopyMoveFailureOrder(t *testing.T) {
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
	satOK(t, dst.Insert(r))
	b.FailAfter(0)
	e = dst.Assign(src)
	if !errors.Is(e, arena.ErrInjectedFailure) || len(dst.ids) != 0 || len(dst.refs) != 1 {
		t.Fatal("ID-vector assignment first; old ref-vector not yet assigned", e, len(dst.ids), len(dst.refs))
	}
	b.FailAfter(-1)
	satOK(t, dst.Assign(src))
	copy, e := src.Clone(b)
	satOK(t, e)
	defer copy.Close()
	satOK(t, dst.MoveAssign(copy))
	if copy.Size() != 1 {
		t.Fatal("implicit clause_set move assignment copies ptr and unsigned svectors")
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
		t.Fatal("source raw copies do not dereference dead borrowed clause", got, e)
	}
}
