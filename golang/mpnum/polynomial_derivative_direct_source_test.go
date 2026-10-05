// Source: src/math/polynomial/polynomial.cpp:4507-4528;
// src/util/mpzzp.h; src/util/mpz.cpp. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialDerivativeDirectEveryBigCoefficientPrefix(t *testing.T) {
	success := false
	failures := 0
	beforeLiftNumeric := 0
	for ordinal := int64(0); ordinal < 160; ordinal++ {
		func() {
			m := polynomialPrimitiveFixture(t)
			c := m.core
			var a Int
			digits := make([]uint32, 80)
			digits[0] = 9
			digits[79] = 1
			serviceMust(t, c.integer.SetLimbs(&a, digits))
			term, e := c.monomials.MkVariable(0, 2)
			serviceMust(t, e)
			serviceMust(t, c.monomials.IncRef(term))
			as := []Int{a}
			ms := []Monomial{term}
			p, e := c.MkCoreTake(as, ms)
			serviceMust(t, e)
			serviceMust(t, c.IncRef(p))
			var receipt polynomialDerivativeInvocationReceipt
			c.integer.budget.FailAfter(ordinal)
			d, e := m.derivativeWithReceipt(p, 0, &receipt)
			polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, d})
			if e != nil && m.derivative.numeric != (polynomialNumericRoots{}) && m.derivative.generated == (Monomial{}) && m.derivative.factoryGeneration == 0 {
				beforeLiftNumeric++
				owned := m.derivative.numeric
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				if m.derivative.numeric != owned {
					t.Fatal("completed ack took unjoined leaves before fault lift", ordinal)
				}
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
				if m.derivative.numeric != (polynomialNumericRoots{}) {
					t.Fatal("actual leaf owner remains before fault lift", ordinal)
				}
				polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, d})
			}
			c.integer.budget.FailAfter(-1)
			if e != nil {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual direct leaf/factory refusal", ordinal, e)
				}
			} else {
				success = true
				got, x := c.coefficientCore(d, 0)
				serviceMust(t, x)
				original, x := c.coefficientCore(p, 0)
				serviceMust(t, x)
				var want Int
				two := NewInt(2)
				serviceMust(t, c.integer.Mul(&original, &two, &want))
				same, x := c.integer.Equal(&got, &want)
				serviceMust(t, x)
				if !same {
					t.Fatal("independent coefficient twice input", ordinal)
				}
				serviceMust(t, c.integer.Del(&want))
			}
			c.integer.budget.FailAfter(-1)
			pendingBefore := m.derivative.numeric
			serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
			if m.derivative.numeric != pendingBefore {
				t.Fatal("completed lexical acknowledgment consumed unjoined numeric leaf", ordinal)
			}
			serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
			polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, d})
			if e == nil {
				serviceMust(t, c.IncRef(d))
				serviceMust(t, c.DecRef(d))
			}
			serviceMust(t, c.DecRef(p))
		}()
		if success {
			t.Log("first actual direct derivative success", ordinal)
			break
		}
	}
	t.Log("numeric-only before-lift prefixes", beforeLiftNumeric)
	if !success || failures == 0 {
		t.Fatal("all real prefix horizon", failures, success)
	}
}

func TestPolynomialDerivativeCopiedOwnerCannotMatchReceipt(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	var r polynomialDerivativeInvocationReceipt
	m.core.integer.budget.FailAfter(0)
	_, e := m.derivativeWithReceipt(p, 0, &r)
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	m.core.integer.budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesDerivativeInvocation(&r) {
		t.Fatal("actual guarded source acquisition", e)
	}
	copied := *m
	forged := r
	forged.manager = &copied
	forged.frame = &copied.derivative
	if copied.matchesDerivativeInvocation(&forged) {
		t.Fatal("copied Go owner gained actual source continuation")
	}
	serviceMust(t, m.cleanupDerivativeInvocation(&r, true))
}
