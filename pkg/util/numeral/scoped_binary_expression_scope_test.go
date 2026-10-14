package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedBinaryExpressionSourceStopRetainsReturnedValue(t *testing.T) {
	q, budget, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(q.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	local, e := NewScopedBinary(m)
	serviceMust(t, e)
	returned, e := NewScopedBinary(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, local.Cleanup()); serviceMust(t, returned.Cleanup()) })
	serviceMust(t, q.Z.SetLimbs(&local.num.Num, []uint32{7, 0, 1}))
	serviceMust(t, q.Z.SetLimbs(&returned.num.Num, []uint32{9, 0, 1}))
	ref := local.num.Num.Cell
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
	result := returned
	var cause error
	finishScopedBinaryExpressionSource(&result, &cause, local)
	var pending *ScopedBinaryExpressionCleanupError
	if !errors.As(cause, &pending) || !errors.Is(cause, arena.ErrBounds) || result != nil || local.closed || returned.closed {
		t.Fatal("source expression lost local or returned value", cause, result)
	}
	if e := pending.Cleanup(); !errors.Is(e, arena.ErrBounds) || local.closed || returned.closed {
		t.Fatal("owning refusal lost expression suffix", e)
	}
	q.Z.Store(ref, original)
	corrupted = false
	serviceMust(t, pending.Cleanup())
	if !local.closed || !returned.closed || local.num.Num.Cell != 0 || returned.num.Num.Cell != 0 {
		t.Fatal("owning retry did not drain expression owners")
	}
	serviceMust(t, pending.Cleanup())
}

func TestScopedBinaryCopyConstructorFailureRetainsUnwoundValue(t *testing.T) {
	m, budget := binaryManager(t)
	a, e := NewScopedBinary(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.Cleanup()) })
	var big Int
	serviceMust(t, m.Z.SetString(&big, "340282366920938463463374607431768211457"))
	t.Cleanup(func() { serviceMust(t, m.Z.Del(&big)) })
	serviceMust(t, a.SetInteger(&big))
	budget.FailAfter(1)
	copy, e := a.Clone()
	var pending *ScopedBinaryExpressionCleanupError
	if copy != nil || !errors.As(e, &pending) || !numericPhysicalReservationRefusal(e) || pending.local == nil || pending.local.closed {
		t.Fatal("failed source copy constructor lost partial owner", copy, e)
	}
	budget.FailAfter(-1)
	serviceMust(t, pending.Cleanup())
	if pending.local != nil {
		t.Fatal("owning copy-constructor cleanup retained local")
	}
	serviceMust(t, pending.Cleanup())
}
