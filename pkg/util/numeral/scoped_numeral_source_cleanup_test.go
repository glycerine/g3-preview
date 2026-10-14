// Source: src/util/scoped_numeral.h:31-33; src/util/mpz.cpp and mpq.cpp.
package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedIntSourceDeletionAndOwningRetry(t *testing.T) {
	q, _, _ := rawArithmeticGuardOwner(t)
	s, e := NewScopedInt(q.Z)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Cleanup()) })
	value, e := s.Get()
	serviceMust(t, e)
	serviceMust(t, q.Z.SetLimbs(value, []uint32{7, 0, 1}))
	ref := value.Cell
	original := q.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	q.Z.Store(ref, bad)
	corrupted := true
	t.Cleanup(func() {
		if corrupted {
			q.Z.Store(ref, original)
		}
	})
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Cell != ref {
		t.Fatal("source destructor lost integer", e)
	}
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Cell != ref {
		t.Fatal("stopped source destructor replayed", e)
	}
	if e := s.Cleanup(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Cell != ref {
		t.Fatal("owning refusal lost integer", e)
	}
	q.Z.Store(ref, original)
	corrupted = false
	serviceMust(t, s.Cleanup())
	if !s.closed || value.Cell != 0 {
		t.Fatal("owning retry retained integer")
	}
	serviceMust(t, s.Close())
}

func TestScopedRationalSourceDeletionAndOwningSuffix(t *testing.T) {
	q, _, _ := rawArithmeticGuardOwner(t)
	s, e := NewScopedRational(q)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Cleanup()) })
	value, e := s.Get()
	serviceMust(t, e)
	serviceMust(t, q.Z.SetLimbs(&value.Num, []uint32{7, 0, 1}))
	serviceMust(t, q.Z.SetLimbs(&value.Den, []uint32{9, 0, 1}))
	denRef := value.Den.Cell
	original := q.Z.Cell(denRef)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	q.Z.Store(denRef, bad)
	corrupted := true
	t.Cleanup(func() {
		if corrupted {
			q.Z.Store(denRef, original)
		}
	})
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Num.Cell != 0 || value.Den.Cell != denRef {
		t.Fatal("source destructor lost rational suffix", e)
	}
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || value.Den.Cell != denRef {
		t.Fatal("stopped source destructor replayed", e)
	}
	if e := s.Cleanup(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Den.Cell != denRef {
		t.Fatal("owning refusal lost denominator", e)
	}
	q.Z.Store(denRef, original)
	corrupted = false
	serviceMust(t, s.Cleanup())
	if !s.closed || value.Den.Cell != 0 {
		t.Fatal("owning retry retained rational")
	}
	serviceMust(t, s.Close())
}
