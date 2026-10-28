package numeral

// Source: src/util/scoped_numeral.h:160-183.

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedIntExpressionBodyFailureEndsTemporary(t *testing.T) {
	m, budget := integerManager(t)
	s, e := NewScopedInt(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Cleanup()) })
	serviceMust(t, s.SetInt64(7))
	baseline := budget.Used()
	zero := NewInt(0)
	result, e := s.Div(&zero)
	var pending *ScopedIntExpressionCleanupError
	if result != nil || !errors.Is(e, ErrDivideByZero) || errors.As(e, &pending) || budget.Used() != baseline {
		t.Fatal("failed arithmetic did not end completed temporary", result, e, budget.Used(), baseline)
	}
}

func TestScopedRationalExpressionBodyFailureEndsTemporary(t *testing.T) {
	q, budget := rationalManager(t, true)
	s, e := NewScopedRational(q)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Cleanup()) })
	serviceMust(t, s.SetInt64(7))
	baseline := budget.Used()
	invalid := NewRational()
	invalid.Den.Val = 0
	result, e := s.Div(&invalid)
	var pending *ScopedRationalExpressionCleanupError
	if result != nil || !errors.Is(e, ErrDivideByZero) || errors.As(e, &pending) || budget.Used() != baseline {
		t.Fatal("failed rational arithmetic did not end temporary", result, e, budget.Used(), baseline)
	}
}

func TestScopedExpressionStoppedDestructorsRetainSourceSuffix(t *testing.T) {
	t.Run("integer", func(t *testing.T) {
		q, _, _ := rawArithmeticGuardOwner(t)
		local, e := NewScopedInt(q.Z)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, local.Cleanup()) })
		serviceMust(t, q.Z.SetLimbs(&local.num, []uint32{7, 0, 1}))
		ref := local.num.Cell
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
		body := errors.New("integer expression body")
		end := finishScopedIntExpressionFailure(local, body)
		var pending *ScopedIntExpressionCleanupError
		if !errors.As(end, &pending) || !errors.Is(end, body) || !errors.Is(end, arena.ErrBounds) || pending.owner != local || local.closed {
			t.Fatal("stopped expression lost its body or integer", end)
		}
		if e := pending.Cleanup(); !errors.Is(e, arena.ErrBounds) || pending.owner != local {
			t.Fatal("owning refusal lost integer", e)
		}
		q.Z.Store(ref, original)
		corrupted = false
		serviceMust(t, pending.Cleanup())
		if pending.owner != nil || !local.closed {
			t.Fatal("owning cleanup retained integer expression")
		}
	})
	t.Run("rational", func(t *testing.T) {
		q, _, _ := rawArithmeticGuardOwner(t)
		local, e := NewScopedRational(q)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, local.Cleanup()) })
		serviceMust(t, q.Z.SetLimbs(&local.num.Num, []uint32{7, 0, 1}))
		serviceMust(t, q.Z.SetLimbs(&local.num.Den, []uint32{9, 0, 1}))
		ref := local.num.Den.Cell
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
		body := errors.New("rational expression body")
		end := finishScopedRationalExpressionFailure(local, body)
		var pending *ScopedRationalExpressionCleanupError
		if !errors.As(end, &pending) || !errors.Is(end, body) || !errors.Is(end, arena.ErrBounds) || pending.owner != local || local.closed || local.num.Num.Cell != 0 || local.num.Den.Cell != ref {
			t.Fatal("stopped expression lost its body or rational suffix", end)
		}
		if e := pending.Cleanup(); !errors.Is(e, arena.ErrBounds) || pending.owner != local {
			t.Fatal("owning refusal lost denominator", e)
		}
		q.Z.Store(ref, original)
		corrupted = false
		serviceMust(t, pending.Cleanup())
		if pending.owner != nil || !local.closed {
			t.Fatal("owning cleanup retained rational expression")
		}
	})
}
