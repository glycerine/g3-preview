// Source: src/math/polynomial/polynomial.cpp:5654-5691.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package polynomial

import (
	"testing"
)

func TestPolynomialLazardDichotomousIndependentCoefficients(t *testing.T) {
	for _, n := range []uint32{0, 1, 2, 3, 4, 7, 8} {
		t.Run(string(rune('A'+n)), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
			y := m.unit
			want := polynomialQueryTerms(t, m, []int32{2 << n, 4 << n}, [][]MonomialPower{{{0, 1}}, nil})
			r, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			serviceMust(t, m.seLazard(n+2, y, p, 0, &r.root))
			equal, e := m.Equal(r.root.value, want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("independent 2^n coefficient oracle", n)
			}
		})
	}
}

func TestPolynomialLazardZeroExponentAvoidsUnreadLeadingCoefficient(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	// n=0 returns before X/C construction or any read of the borrowed Y.
	serviceMust(t, m.seLazard(2, Polynomial{owner: ^uint64(0)}, p, 0, &r.root))
	if r.root.value != p {
		t.Fatal("source identity assignment before unread Y")
	}
}
