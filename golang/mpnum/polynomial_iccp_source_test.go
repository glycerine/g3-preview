// Source: src/math/polynomial/polynomial.cpp:3453-3530,4352-4506.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "testing"

func TestPolynomialICCPIndependentContentAndPrimitiveBranches(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		p, cwant, ppwant := m.zero, m.unit, m.zero
		iwant := int32(0)
		switch route {
		case 1:
			p = polynomialQueryTerms(t, m, []int32{-7}, [][]MonomialPower{nil})
			ppwant = m.unit
			iwant = -7
		case 2:
			p = polynomialQueryTerms(t, m, []int32{-6, 10}, [][]MonomialPower{{{1, 1}}, nil})
			cwant = polynomialQueryTerms(t, m, []int32{-3, 5}, [][]MonomialPower{{{1, 1}}, nil})
			ppwant = m.unit
			iwant = 2
		case 3:
			p = polynomialQueryTerms(t, m, []int32{6, 12, 18}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}})
			ppwant = polynomialQueryTerms(t, m, []int32{1, 2, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}})
			iwant = 6
		case 4:
			p = polynomialQueryTerms(t, m, []int32{6, 12}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}})
			cwant = polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{1, 1}}})
			ppwant = polynomialQueryTerms(t, m, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
			iwant = 6
		}
		i := NewInt(99)
		c, e := NewPolynomialRef(m)
		serviceMust(t, e)
		pp, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, c.Assign(m.zero))
		serviceMust(t, pp.Assign(m.unit))
		serviceMust(t, full.IntegerContentPrimitiveParts(p, 0, &i, c, pp))
		if i != NewInt(iwant) {
			t.Fatal("independent integer component", route, i, iwant)
		}
		equal, e := m.Equal(c.root.value, cwant)
		serviceMust(t, e)
		if !equal {
			t.Fatal("independent content", route)
		}
		equal, e = m.Equal(pp.root.value, ppwant)
		serviceMust(t, e)
		if !equal {
			t.Fatal("independent primitive part", route)
		}
		serviceMust(t, c.Close())
		serviceMust(t, pp.Close())
		serviceMust(t, m.modular.Del(&i))
	}
}

func TestPolynomialGeneralGCDIndependentCommonContent(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	// u=(x+y)*(x+1), v=(x+y)*(y+2), whose primitive GCD is x+y.
	u := polynomialQueryTerms(t, m, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}})
	v := polynomialQueryTerms(t, m, []int32{1, 1, 2, 2}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}, {{0, 1}}, {{1, 1}}})
	want := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	serviceMust(t, full.GCD(u, v, r))
	equal, e := m.Equal(r.root.value, want)
	serviceMust(t, e)
	if !equal {
		t.Fatal("independent common linear factor")
	}
	serviceMust(t, r.Close())
}
