// Source: src/math/polynomial/polynomial.cpp:5659-5831; src/util/obj_ref.h:31-112.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// Each fixture enters the same actual constructor prefix used by the body.
// A real last DEC invokes the handler while that prefix is being destroyed.
func TestPolynomialPSCActualCleanupDeletionReentry(t *testing.T) {
	for actor := 0; actor < 3; actor++ {
		for _, owning := range []bool{false, true} {
			full := polynomialPSCFullFixture(t)
			m := full.core
			var root *polynomialReference
			var finish func() error
			var cursor func() uint8
			var cause func() error
			switch actor {
			case 0:
				s, e := m.beginLazardInvocation(nil)
				serviceMust(t, e)
				root = &s.x
				finish = func() error { return m.finishLazardPhase(owning, owning) }
				cursor = func() uint8 { return s.cursor }
				cause = func() error { return s.cleanupCause }
				s.executing = false
			case 1:
				s, e := m.beginPSCNextInvocation(nil)
				serviceMust(t, e)
				root = &s.refs[3]
				finish = func() error { return m.finishPSCNextPhase(owning, owning) }
				cursor = func() uint8 { return s.cursor }
				cause = func() error { return s.cleanupCause }
				s.executing = false
			case 2:
				s, e := m.beginPSCInvocation(nil)
				serviceMust(t, e)
				root = &s.refs[5].root
				finish = func() error { return m.finishPSCPhase(owning, owning) }
				cursor = func() uint8 { return s.cursor }
				cause = func() error { return s.cleanupCause }
				s.executing = false
			}
			m.pscLeave()
			p := polynomialICCPUnownedTerms(t, m, []int32{73}, [][]MonomialPower{nil})
			serviceMust(t, root.assign(m, p))
			calls := 0
			var nested error
			h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, v Polynomial) error {
				if v != p {
					return nil
				}
				calls++
				beforeCursor, beforeRoot, beforeCause, beforeState := cursor(), root.value, cause(), m.memory.State()
				nested = finish()
				if cursor() != beforeCursor || root.value != beforeRoot || cause() != beforeCause || m.memory.State() != beforeState {
					t.Errorf("reentrant cleanup mutated current destructor actor=%d owning=%v", actor, owning)
				}
				return arena.ErrBounds
			})
			serviceMust(t, e)
			serviceMust(t, m.AddDeletionHandler(h))
			returned := finish()
			serviceMust(t, m.RemoveDeletionHandler(h))
			serviceMust(t, h.CloseChecked())
			// A callback error is a real source/own stop, independent of reentry refusal.
			if !errors.Is(returned, arena.ErrBounds) || calls != 1 || !errors.Is(nested, ErrPolynomialPending) {
				t.Errorf("actual finalizer reentry actor=%d owning=%v returned=%v calls=%d nested=%v", actor, owning, returned, calls, nested)
			}
			first := cause()
			serviceMust(t, m.finishLazardPhase(true, true))
			serviceMust(t, m.finishPSCNextPhase(true, true))
			serviceMust(t, m.finishPSCPhase(true, true))
			if owning && cause() != first {
				t.Error("first owning cause lost across acknowledgment")
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, nil, nil, 0, nil)
		}
	}
}

func TestPolynomialPSCHActualCleanupDeletionReentry(t *testing.T) {
	for _, owning := range []bool{false, true} {
		full := polynomialPSCFullFixture(t)
		m := full.core
		var receipt polynomialPSCNextInvocationReceipt
		s, e := m.beginPSCNextInvocation(&receipt)
		serviceMust(t, e)
		b := &m.pscBuffer
		serviceMust(t, b.constructWithNextReceipt(m, &receipt))
		s.bufferConstructed = true
		s.bufferInvocation = b.invocation
		p := polynomialICCPUnownedTerms(t, m, []int32{79}, [][]MonomialPower{nil})
		serviceMust(t, b.pushWithNextReceipt(m, p, &receipt))
		s.executing = false
		m.pscLeave()
		calls := 0
		var nested error
		h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, v Polynomial) error {
			if v != p {
				return nil
			}
			calls++
			cursor, count, state := b.cursor, b.count, m.memory.State()
			nested = b.finalizePhase(m, owning, owning)
			if b.cursor != cursor || b.count != count || m.memory.State() != state {
				t.Error("recursive H destructor mutated actual buffer")
			}
			return arena.ErrBounds
		})
		serviceMust(t, e)
		serviceMust(t, m.AddDeletionHandler(h))
		returned := b.finalizePhase(m, owning, owning)
		serviceMust(t, m.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		if calls != 1 || !errors.Is(returned, arena.ErrBounds) || !errors.Is(nested, ErrPolynomialPending) {
			t.Errorf("actual H finalizer reentry owning=%v calls=%d returned=%v nested=%v", owning, calls, returned, nested)
		}
		serviceMust(t, m.finishPSCNextPhase(true, true))
		polynomialImpPhysicalAuditNamedRaw(t, full, nil, nil, 0, nil)
	}
}
