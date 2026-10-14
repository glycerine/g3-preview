package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedBinarySourceDeletionAndOwningRetry(t *testing.T) {
	q, budget, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(q.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	s, e := NewScopedBinary(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Cleanup()) })
	value, e := s.Get()
	serviceMust(t, e)
	serviceMust(t, q.Z.SetLimbs(&value.Num, []uint32{7, 0, 1}))
	ref := value.Num.Cell
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
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Num.Cell != ref {
		t.Fatal("source destructor lost binary numerator", e)
	}
	if e := s.Close(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Num.Cell != ref {
		t.Fatal("source destructor replayed", e)
	}
	if e := s.Cleanup(); !errors.Is(e, arena.ErrBounds) || s.closed || value.Num.Cell != ref {
		t.Fatal("owning refusal lost binary numerator", e)
	}
	q.Z.Store(ref, original)
	corrupted = false
	serviceMust(t, s.Cleanup())
	if !s.closed || value.Num.Cell != 0 {
		t.Fatal("owning retry retained binary value")
	}
	serviceMust(t, s.Close())
}
