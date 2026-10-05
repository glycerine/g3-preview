// Source: src/math/polynomial/polynomial.cpp:2757-2768.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"fmt"
	"testing"
)

func TestPolynomialScalarDivLiteralIntegerAndFieldValues(t *testing.T) {
	for _, field := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		if field {
			serviceMust(t, c.modular.SetZPUint64(23))
		}
		p := polynomialQueryTerms(t, c, []int32{6, 10}, [][]MonomialPower{{{0, 1}}, nil})
		d := NewInt(2)
		r, e := c.scalarDiv(p, &d)
		serviceMust(t, e)
		r = polynomialCaller(t, c, r)
		polynomialQueryExpect(t, c, r, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 5})
		if d != NewInt(2) {
			t.Fatal("source borrowed divisor changed")
		}
	}
}

// The source has no zero-divisor or divisor read before the loop. Its empty
// polynomial path still reaches cheap.mk and the completed NUM destructor.
func TestPolynomialScalarDivEmptyDoesNotReadDivisor(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	for _, d := range []*Int{nil, new(Int)} {
		r, e := c.scalarDiv(c.zero, d)
		serviceMust(t, e)
		r = polynomialCaller(t, c, r)
		zero, e := c.IsZero(r)
		serviceMust(t, e)
		if !zero {
			t.Fatal("literal zero-size loop and cheap.mk")
		}
	}
}

func TestPolynomialScalarDivEveryActualReservationPrefix(t *testing.T) {
	for route := 0; route < 3; route++ {
		success := false
		for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
			t.Run(fmt.Sprintf("%d/%d", route, ordinal), func(t *testing.T) {
				m := polynomialPSCFullFixture(t)
				c := m.core
				_, e := c.monomials.MkVar()
				serviceMust(t, e)
				if route == 1 {
					serviceMust(t, c.modular.SetZPUint64(23))
				}
				p := polynomialQueryTerms(t, c, []int32{6, 10}, [][]MonomialPower{{{0, 1}}, nil})
				d := NewInt(2)
				var divisor *Int = &d
				if route == 2 {
					p = c.zero
					divisor = nil
				}
				var receipt polynomialScalarDivInvocationReceipt
				c.integer.budget.FailAfter(ordinal)
				result, e := c.scalarDivWithReceipt(p, divisor, &receipt)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual scalarDiv refusal", e)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.scalarDivision.returned}, []*Int{&d, &c.scalarDivision.a}, 0)
				c.integer.budget.FailAfter(-1)
				serviceMust(t, c.cleanupScalarDivInvocation(&receipt, false))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.scalarDivision.returned}, []*Int{&d, &c.scalarDivision.a}, 0)
				serviceMust(t, c.cleanupScalarDivInvocation(&receipt, true))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, []*Int{&d, &c.scalarDivision.a}, 0)
				if result != (Polynomial{}) {
					result = polynomialCaller(t, c, result)
				}
				if e == nil {
					if route == 2 {
						zero, x := c.IsZero(result)
						serviceMust(t, x)
						if !zero {
							t.Fatal("zero-size source loop")
						}
					} else {
						polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 5})
					}
					success = true
					t.Log("actual scalarDiv first success", ordinal)
				}
			})
		}
		if !success {
			t.Fatal("scalarDiv first success not reached", route)
		}
	}
}
