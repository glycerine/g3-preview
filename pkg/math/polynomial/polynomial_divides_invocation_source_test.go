// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/util/obj_ref.h; src/util/scoped_numeral.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialDividesActualMatchedInvocationEveryPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		full, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{5, 4, 0}}})
		c := full.core
		q := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}, {1, 2}}})
		var receipt polynomialDividesInvocationReceipt
		c.integer.Budget().FailAfter(ordinal)
		got, cause := c.dividesWithReceipt(q, p, &receipt)
		polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
		if cause != nil && !numericPhysicalReservationRefusal(cause) {
			t.Fatal("actual divide source prefix", ordinal, cause)
		}
		if cause != nil {
			// Literal p==0 returns before q assertion, prelude and local
			// construction, even while the older failed invocation is retained.
			untouched := receipt
			used := c.integer.Budget().Used()
			zeroAnswer, zeroCause := c.dividesWithReceipt(c.zero, c.zero, &untouched)
			if zeroCause != nil || !zeroAnswer || untouched != receipt || c.integer.Budget().Used() != used {
				t.Fatal("source zero route changed older invocation", ordinal, zeroCause)
			}
			copied := receipt
			if copied.frame != nil {
				frameCopy := *copied.frame
				copied.frame = &frameCopy
				serviceMust(t, c.cleanupDividesInvocation(&copied, true))
				if c.integer.Budget().Used() != used {
					t.Fatal("copied local frame consumed actual roots", ordinal)
				}
			}
			polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
		serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
		polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
		if c.dividesLocal.active {
			t.Fatal("matched owning continuation retained", ordinal)
		}
		if cause == nil {
			if !got {
				t.Fatal("independent monomial divisibility")
			}
			t.Log("actual invocation success", ordinal)
			return
		}
	}
	t.Fatal("actual source divide never completed")
}

func TestPolynomialDividesFalseReturnKeepsPersistentRAndNormalReuse(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	seven := util.NewInt(7)
	p, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	q, e := c.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(q))
	defer func() { serviceMust(t, c.DecRef(q)) }()
	var receipt polynomialDividesInvocationReceipt
	answer, e := c.dividesWithReceipt(q, p, &receipt)
	serviceMust(t, e)
	if answer || receipt.frame.active || c.som.arithmeticScope.active {
		t.Fatal("source false return retained adapter busy state")
	}
	as, e := c.som.as.size(c)
	serviceMust(t, e)
	ms, e := c.som.ms.size(c)
	serviceMust(t, e)
	if as != 1 || ms != 1 {
		t.Fatal("false return changed persistent R rows", as, ms)
	}
	coefficient, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	equal, e := c.modular.Equal(&coefficient, &seven)
	serviceMust(t, e)
	if !equal {
		t.Fatal("source false return changed R coefficient")
	}
	used, rawAs, rawMs := c.integer.Budget().Used(), c.som.as.raw, c.som.ms.raw
	unitRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
	serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	after, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if c.integer.Budget().Used() != used || c.som.as.raw != rawAs || c.som.ms.raw != rawMs || after != unitRC {
		t.Fatal("completed receipt consumed member rows")
	}
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
	answer, e = c.Divides(p, p)
	serviceMust(t, e)
	if !answer || c.dividesLocal.active {
		t.Fatal("ordinary next invocation failed independent coefficient/monomial divisibility")
	}
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
}
