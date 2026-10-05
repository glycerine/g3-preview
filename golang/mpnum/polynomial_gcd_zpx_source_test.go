// Source: src/math/polynomial/polynomial.cpp:4018-4051.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"testing"
)

func TestPolynomialGCDGlexZPXIndependentCoefficientModel(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	// Viewed over GF11[x], maximal remaining monomial y^2 has coefficient 3x+4.
	p := polynomialQueryTerms(t, c, []int32{3, 4, 5, 1}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 2}}, {{0, 3}, {1, 1}}, nil})
	a, e := full.lcGlexZPX(p, 0)
	serviceMust(t, e)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, r.root.assign(c, a))
	polynomialQueryExpect(t, c, a, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 4})
}

func TestPolynomialGCDICCPZPXContentAndPrimitiveIndependentModel(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	serviceMust(t, c.modular.SetZPUint64(11))
	// (x+1)(y+2) has GF11[x] content x+1 and primitive y+2.
	p := polynomialQueryTerms(t, c, []int32{1, 2, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
	cr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cr.Close()); serviceMust(t, pr.Close()) })
	var ci Int
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&ci)) })
	serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
	polynomialQueryExpect(t, c, cr.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	polynomialQueryExpect(t, c, pr.root.value, map[[3]uint32]int64{{0, 1, 0}: 1, {}: 2})
}

func TestPolynomialGCDICCPZPXSourceShortcutTable(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		serviceMust(t, c.modular.SetZPUint64(11))
		var p Polynomial
		var cw, pw map[[3]uint32]int64
		ciWant := int64(1)
		switch route {
		case 0:
			p = c.zero
			cw = map[[3]uint32]int64{{}: 1}
			pw = map[[3]uint32]int64{}
			ciWant = 0
		case 1:
			p = polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{nil})
			cw = map[[3]uint32]int64{{}: 1}
			pw = cw
			ciWant = 3
		case 2:
			p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{1, 1}}, nil})
			cw = map[[3]uint32]int64{{}: 1}
			pw = map[[3]uint32]int64{{0, 1, 0}: 1, {}: 1}
		case 3:
			p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 2}}})
			cw = map[[3]uint32]int64{{2, 0, 0}: 1}
			pw = map[[3]uint32]int64{{0, 1, 0}: 1, {}: 1}
		case 4:
			p = polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{2, 1}}})
			cw = map[[3]uint32]int64{{}: 1}
			pw = map[[3]uint32]int64{{1, 1, 0}: 1, {0, 0, 1}: 1}
		}
		cr, e := NewPolynomialRef(c)
		serviceMust(t, e)
		pr, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, cr.Close()); serviceMust(t, pr.Close()) })
		var ci Int
		t.Cleanup(func() { serviceMust(t, c.integer.Del(&ci)) })
		serviceMust(t, full.iccpZPX(p, 0, &ci, &cr.root, &pr.root))
		got, e := c.integer.Int64(&ci)
		serviceMust(t, e)
		if got != ciWant {
			t.Fatal("source ci shortcut", route, got, ciWant)
		}
		polynomialQueryExpect(t, c, cr.root.value, cw)
		polynomialQueryExpect(t, c, pr.root.value, pw)
	}
}

func TestPolynomialGCDIntegerContentFieldChangeSourceInputAssertion(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	p := polynomialQueryTerms(t, c, []int32{12, 16}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	serviceMust(t, c.modular.SetZPUint64(11))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	var content Int
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&content)) })
	e = c.IntegerContent(p, &content, r)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, r.root.value, r.root.incoming}, []*Int{&content}, r.header)
	serviceMust(t, c.CleanupIntegerContent())
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, r.root.value, r.root.incoming}, []*Int{&content}, r.header)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatalf("source div requires normalized dividend: %v", e)
		}
		return
	}
	serviceMust(t, e)
	got, e := c.integer.Int64(&content)
	serviceMust(t, e)
	if got != 4 {
		t.Fatal("integer content source manager", got)
	}
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{1, 0, 0}: 3, {0, 1, 0}: 4})
}
