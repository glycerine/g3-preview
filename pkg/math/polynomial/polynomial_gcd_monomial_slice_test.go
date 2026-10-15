// Source: src/util/ref_buffer.h:32-108;
// src/math/polynomial/polynomial.cpp:3895-4015.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialGCDNativeMonomialSliceOwnsDuplicateReferences(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	before, e := c.monomialRefCount(unit)
	serviceMust(t, e)
	var b polynomialGCDMonomialBuffer
	b.construct()
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, b.close(c, true)) })
	stats, used := c.memory.Stats(), c.integer.Budget().Used()
	c.integer.Budget().FailAfter(0)
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < 4097; i++ {
			serviceMust(t, b.push(c, unit))
		}
		refs, e := c.monomialRefCount(unit)
		serviceMust(t, e)
		if refs != before+4097 {
			t.Fatal("slice growth lost monomial references", refs, before)
		}
		for i := uint32(0); i < uint32(len(b.values)); i++ {
			a, e := b.get(i)
			serviceMust(t, e)
			if a != unit {
				t.Fatal("monomial identity changed", i)
			}
		}
		serviceMust(t, b.reset(c, false))
		refs, e = c.monomialRefCount(unit)
		serviceMust(t, e)
		if refs != before || len(b.values) != 0 {
			t.Fatal("reset failed to release each owned reference", refs, before)
		}
	}
	serviceMust(t, b.close(c, false))
	serviceMust(t, b.close(c, true))
	if b.values != nil || c.memory.Stats() != stats || c.integer.Budget().Used() != used {
		t.Fatal("native reference storage retained backing or charged Runtime")
	}
}

func TestPolynomialGCDNativeMonomialSliceGeneratedOwnershipAndReset(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	var b polynomialGCDMonomialBuffer
	b.construct()
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, b.close(c, true)) })
	a, e := c.generatedVariable(0, 17)
	serviceMust(t, e)
	if !c.generatedActive {
		t.Fatal("fixture did not create monomial")
	}
	serviceMust(t, b.push(c, a))
	if c.generatedActive {
		t.Fatal("slice did not acquire ownership")
	}
	c.integer.Budget().FailAfter(0)
	serviceMust(t, b.reset(c, false))
	c.integer.Budget().FailAfter(-1)
	if len(b.values) != 0 || b.dropping {
		t.Fatal("native recycle interrupted reset", b)
	}
	if _, e := c.monomials.location(a); e == nil {
		t.Fatal("last monomial reference survived reset")
	}
	serviceMust(t, b.close(c, true))
}

func TestPolynomialGCDICCPZPXNativeMonomialGrowthIndependentModel(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	for i := 0; i < 2; i++ {
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
	}
	serviceMust(t, c.modular.SetZPUint64(11))
	// (x+1) times (1+y+...+y^32) has GF11[x] content x+1.
	coefficients := make([]int32, 0, 66)
	powers := make([][]MonomialPower, 0, 66)
	primitive := make(map[[3]uint32]int64)
	for degree := uint32(0); degree <= 32; degree++ {
		for x := uint32(0); x <= 1; x++ {
			var term []MonomialPower
			if x != 0 {
				term = append(term, MonomialPower{0, x})
			}
			if degree != 0 {
				term = append(term, MonomialPower{1, degree})
			}
			coefficients = append(coefficients, 1)
			powers = append(powers, term)
		}
		primitive[[3]uint32{0, degree, 0}] = 1
	}
	p := polynomialQueryTerms(t, c, coefficients, powers)
	cr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var ci unumeral.Int
	t.Cleanup(func() { serviceMust(t, cr.Close()); serviceMust(t, pr.Close()); serviceMust(t, c.integer.Del(&ci)) })
	serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
	polynomialQueryExpect(t, c, cr.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	polynomialQueryExpect(t, c, pr.root.value, primitive)
}
