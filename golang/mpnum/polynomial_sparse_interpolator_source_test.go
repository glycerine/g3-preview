// Source: src/math/polynomial/polynomial.cpp:3072-3182.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "testing"

func TestPolynomialSparseInterpolatorIndependentThreeSamples(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1, 3, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()); serviceMust(t, sk.CloseChecked()) })
	for _, x := range []int32{0, 1, 2} {
		in := NewInt(x)
		q := polynomialQueryTerms(t, m, []int32{x*x + 3*x + 2}, [][]MonomialPower{nil})
		ok, e := s.Add(&in, q)
		serviceMust(t, e)
		if !ok {
			t.Fatal("independent sample support", x)
		}
	}
	ready, e := s.Ready()
	serviceMust(t, e)
	if !ready {
		t.Fatal("three actual skeleton powers")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	ok, e := s.Interpolate(out)
	serviceMust(t, e)
	if !ok {
		t.Fatal("actual Vandermonde solution")
	}
	equal, e := m.Equal(out.root.value, p)
	serviceMust(t, e)
	if !equal {
		t.Fatal("independent x^2+3x+2 interpolation")
	}
	polynomialSparsePhysical(t, s, out, []Polynomial{p})
	serviceMust(t, out.Close())
	serviceMust(t, s.CloseChecked())
	serviceMust(t, sk.CloseChecked())
}
