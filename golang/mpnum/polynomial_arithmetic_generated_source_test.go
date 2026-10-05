// Source: src/math/polynomial/polynomial.cpp:1988-2039,862-884.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialRegularProductActualCalleeAcquisition(t *testing.T) {
	saw := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		mm := c.monomials
		_, e := mm.MkVar()
		serviceMust(t, e)
		_, e = mm.MkVar()
		serviceMust(t, e)
		p, e := c.MkVariable(0, 1)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		term, e := mm.MkVariable(1, 1)
		serviceMust(t, e)
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		one := NewInt(1)
		c.integer.budget.FailAfter(ordinal)
		e = c.som.addMulWithArithmeticReceipt(&one, term, p, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		if e != nil && c.generatedCalleePending {
			if r.frame.calleeGeneration != c.generatedCalleeGeneration {
				t.Fatal("actual product callee acquisition absent", ordinal, e, mm.creation)
			}
			saw = true
		}
		if e != nil && c.generatedActive && r.frame.generated != c.generatedMonomial {
			t.Fatal("actual created product authority absent", ordinal, e)
		}
		c.integer.budget.FailAfter(-1)
		serviceMust(t, mm.CleanupCreation())
		serviceMust(t, c.cleanupGeneratedMonomial())
		serviceMust(t, c.som.close())
		serviceMust(t, c.DecRef(p))
		serviceMust(t, mm.CleanupValue(&term))
		if e == nil {
			if !saw {
				t.Fatal("no actual callee failure prefix")
			}
			return
		}
	}
	t.Fatal("actual product horizon not reached")
}

// Source: src/math/polynomial/polynomial.cpp:1885-2156,810-851.
// An actual created product is transferred or consumed at its real leaf.
func TestPolynomialRegularCreatedProductGenericConsumptionAcknowledgesExactScope(t *testing.T) {
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c, mm := full.core, full.core.monomials
		for i := 0; i < 2; i++ {
			_, e := mm.MkVar()
			serviceMust(t, e)
		}
		p, e := c.MkVariable(0, 1)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		term, e := mm.MkVariable(1, 1)
		serviceMust(t, e)
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		one := NewInt(1)
		c.integer.budget.FailAfter(ordinal)
		body := c.som.addMulWithArithmeticReceipt(&one, term, p, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		captured := body != nil && c.generatedActive && r.frame.generated == c.generatedMonomial
		c.integer.budget.FailAfter(-1)
		serviceMust(t, mm.CleanupCreation())
		serviceMust(t, c.cleanupGeneratedMonomial())
		if captured && r.frame.generated != (Monomial{}) {
			t.Fatal("actual owning consumption left stale generated authority", ordinal, body, r.frame.generated)
		}
		var replacement Monomial
		if captured {
			base, e := c.monomialCore(p, 0)
			serviceMust(t, e)
			replacement, e = c.generatedProduct(term, base)
			serviceMust(t, e)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		}
		serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
		serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
		if captured && (!c.generatedActive || c.generatedMonomial != replacement) {
			t.Fatal("old generated receipt swept later actual foreign generation")
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		serviceMust(t, c.cleanupGeneratedMonomial())
		serviceMust(t, c.DecRef(p))
		serviceMust(t, mm.CleanupValue(&term))
		if captured {
			return
		}
		if body == nil {
			t.Fatal("source-created interrupted product prefix not reached", ordinal)
		}
	}
	t.Fatal("source-created interrupted product horizon")
}

func TestPolynomialRegularCreatedProductActualINCTransfersExactScope(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c, mm := full.core, full.core.monomials
	for i := 0; i < 2; i++ {
		_, e := mm.MkVar()
		serviceMust(t, e)
	}
	a, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	b, e := mm.MkVariable(1, 1)
	serviceMust(t, e)
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	// Enter the actual source multiplication subtree directly. This tests its
	// leaf transfer; no NUM/default row or source append is claimed.
	r.frame.executing = true
	product, e := c.som.arithmeticProduct(a, b, &r)
	r.frame.executing = false
	serviceMust(t, e)
	if r.frame.generated != product {
		t.Fatal("actual created product absent")
	}
	serviceMust(t, mm.IncRef(product))
	c.transferGeneratedMonomial()
	if r.frame.generated != (Monomial{}) {
		t.Fatal("actual INC retained old created authority")
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	l, e := mm.location(product)
	serviceMust(t, e)
	refs, e := mm.allocator.ReadUint32(l.raw, 0)
	serviceMust(t, e)
	if refs != 1 {
		t.Fatal("old created abort consumed caller-owned reference", refs)
	}
	serviceMust(t, mm.DecRef(product))
	serviceMust(t, mm.CleanupValue(&a))
	serviceMust(t, mm.CleanupValue(&b))
}
