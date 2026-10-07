// Source: src/math/polynomial/polynomial.cpp:4507-4528;
// src/util/mpz.h:680-694; src/util/mpzzp.h:230-233.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialDerivativeActualUnsignedSetLeafPrefixes(t *testing.T) {
	selected, success := false, false
	for prefix := int64(0); prefix < 128; prefix++ {
		m := polynomialPrimitiveFixture(t)
		c := m.core
		term, e := c.monomials.MkVariable(0, ^uint32(0))
		serviceMust(t, e)
		p, e := c.MkTerms([]util.Int{util.NewInt(1)}, []Monomial{term})
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var receipt polynomialDerivativeInvocationReceipt
		c.integer.Budget().FailAfter(prefix)
		d, e := m.derivativeWithReceipt(p, 0, &receipt)
		polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, d})
		s := &m.derivative
		if e != nil && s.numericPhase == 1 && (s.numeric).NumericAllocationRoots != (polynomialNumericRoots{}).NumericAllocationRoots {
			selected = true
			if !numericPhysicalReservationRefusal(e) || !m.matchesDerivativeInvocation(&receipt) || s.factoryGeneration != 0 || s.calleeGeneration != 0 {
				t.Fatal("unsigned SET ownership preceded its actual source acquisition", prefix, e)
			}
			before := s.numeric
			serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
			if (s.numeric).NumericAllocationRoots != (before).NumericAllocationRoots {
				t.Fatal("older NUM acknowledgment consumed unjoined SET leaf", prefix)
			}
			// The declaration has completed; these are exact unjoined raw/cell
			// acquisitions, retired under the still-selected failure budget.
			serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
			polynomialImpPhysicalAuditValues(t, m, []Polynomial{p, d})
		}
		// Later factory construction may recycle an actual ID using new vector
		// storage. Its owning reservation domain is separate from the SET leaf.
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
		serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
		if e == nil {
			success = true
			coefficient, err := c.coefficientCore(d, 0)
			serviceMust(t, err)
			value, err := c.integer.Uint64(&coefficient)
			serviceMust(t, err)
			monomial, err := c.monomialCore(d, 0)
			serviceMust(t, err)
			degree, err := c.monomials.DegreeOf(monomial, 0)
			serviceMust(t, err)
			if value != uint64(^uint32(0)) || degree != ^uint32(0)-1 {
				t.Fatal("independent d(x^UINT_MAX)/dx coefficient or degree", value, degree)
			}
			serviceMust(t, c.IncRef(d))
			serviceMust(t, c.DecRef(d))
		}
		serviceMust(t, c.DecRef(p))
		if success {
			t.Log("first actual UINT_MAX derivative success", prefix)
			break
		}
	}
	if !selected || !success {
		t.Fatal("actual allocating unsigned SET and first-success horizon missing", selected, success)
	}
}
