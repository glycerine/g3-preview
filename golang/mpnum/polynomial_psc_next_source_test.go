// Source: src/math/polynomial/polynomial.cpp:5694-5749.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialPSCNextIndependentQuadraticLinearIdentity(t *testing.T) {
	for b := int32(-3); b <= 3; b++ {
		m := polynomialCellFixture(t)
		a := polynomialQueryTerms(t, m, []int32{1, b, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
		linear := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
		want := polynomialQueryTerms(t, m, []int32{28 - 8*b}, [][]MonomialPower{nil})
		r, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, m.optimizedSe1(2, 1, a, linear, linear, m.unit, 0, &r.root))
		eq, e := m.Equal(r.root.value, want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("independent L^2*c-L*b*K+K^2", b)
		}
		serviceMust(t, r.Close())
	}
}

func TestPolynomialPSCNextHeapBufferIndependentRemainder(t *testing.T) {
	m := polynomialCellFixture(t)
	a := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 18}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 2}, [][]MonomialPower{{{0, 17}}, nil})
	want := polynomialQueryTerms(t, m, []int32{-2, 3}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, m.optimizedSe1(18, 17, a, q, q, m.unit, 0, &r.root))
	equal, e := m.Equal(r.root.value, want)
	serviceMust(t, e)
	if !equal {
		t.Fatal("x^18+3 modulo x^17+2 = 3-2*x")
	}
	if !m.pscBuffer.finalized || m.pscBuffer.raw != (memory.Block{}) {
		t.Fatal("actual completed source H destructor", m.pscBuffer)
	}
}
