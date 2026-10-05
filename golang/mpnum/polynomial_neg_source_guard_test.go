// Source: src/math/polynomial/polynomial.cpp:5940-5951;
// src/util/scoped_numeral.h; src/util/mpz.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// Reallocation can leave the actual setter-recorded dead pointer. Literal
// source destruction must stop at its capacity guard; owning cleanup may later
// acknowledge it. This guarded source-pointer domain is not mathematical OOM.
func TestPolynomialNegFailedSetterSourceDestructorGuard(t *testing.T) {
	success, guard := false, false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			var first, large Int
			limbs := make([]uint32, 8192)
			limbs[0] = 1
			limbs[len(limbs)-1] = 1
			serviceMust(t, c.integer.SetLimbs(&large, limbs))
			short := make([]uint32, 4096)
			short[0] = 1
			short[len(short)-1] = 1
			serviceMust(t, c.integer.SetLimbs(&first, short))
			t.Cleanup(func() { serviceMust(t, c.integer.Del(&large)); serviceMust(t, c.integer.Del(&first)) })
			x, e := c.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			p, e := c.MkTerms([]Int{first, large}, []Monomial{x, c.monomials.unit})
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			var old, current polynomialNegInvocationReceipt
			warm, e := c.negWithReceipt(c.unit, &old)
			serviceMust(t, e)
			warm = polynomialCaller(t, c, warm)
			c.integer.budget.FailAfter(ordinal)
			result, e := c.negWithReceipt(p, &current)
			if old.frame != current.frame || old.invocation == current.invocation {
				t.Fatal("actual reused Neg frame lacks unique invocation", old, current)
			}
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual Neg body refusal", e)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, warm, result, c.negativeResult}, []*Int{&first, &large}, 0)
			if errors.Is(c.negativeCleanupCause, arena.ErrInvalidSlot) {
				if !c.negativeActive || !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("source capacity stop lost actual NUM scope", e, c.negativeActive)
				}
				if c.negInvocationSourceCause(&current) != c.negativeCleanupCause || c.negInvocationSourceCause(&old) != nil {
					t.Fatal("Neg source cause is not exact invocation")
				}
				used, state := c.integer.budget.Used(), c.memory.State()
				serviceMust(t, c.cleanupNegInvocation(&old, false))
				serviceMust(t, c.cleanupNegInvocation(&old, true))
				foreign := current
				foreign.manager = &MultivariatePolynomialCore{}
				serviceMust(t, c.cleanupNegInvocation(&foreign, true))
				if !c.negativeActive || c.integer.budget.Used() != used || c.memory.State() != state {
					t.Fatal("stale/foreign Neg receipt consumed current invocation")
				}
				rejected := current
				_, entry := c.negWithReceipt(p, &rejected)
				if entry == nil || rejected != current {
					t.Fatal("Neg rejected entry changed receipt", entry)
				}
				t.Log("actual source capacity stop", ordinal)
				guard = true
			}
			firstCause := c.negInvocationSourceCause(&current)
			c.integer.budget.FailAfter(-1)
			serviceMust(t, c.cleanupNegInvocation(&current, false))
			if c.negInvocationSourceCause(&current) != firstCause {
				t.Fatal("owning NUM acknowledgment erased source first cause")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, warm, result, c.negativeResult}, []*Int{&first, &large}, 0)
			serviceMust(t, c.cleanupNegInvocation(&current, true))
			if c.negInvocationSourceCause(&current) != firstCause {
				t.Fatal("owning abort erased source first cause")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, warm, result}, []*Int{&first, &large}, 0)
			if firstCause != nil {
				stoppedReturn := e
				var later polynomialNegInvocationReceipt
				next, nextCause := c.negWithReceipt(c.unit, &later)
				serviceMust(t, nextCause)
				next = polynomialCaller(t, c, next)
				if c.negInvocationSourceCause(&later) != nil || c.negInvocationSourceCause(&current) != nil || !errors.Is(stoppedReturn, firstCause) {
					t.Fatal("next invocation retargeted the earlier returned source error", stoppedReturn, firstCause)
				}
			}
			if result != (Polynomial{}) {
				result = polynomialCaller(t, c, result)
			}
			if e == nil {
				success = true
				t.Log("actual first success", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("Neg first-success horizon not reached")
	}
	if !guard {
		t.Fatal("literal Neg scoped NUM destructor guard was bypassed by owning deletion")
	}
}
