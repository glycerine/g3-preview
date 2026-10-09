// Source: src/math/polynomial/polynomial.cpp:3072-3182.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

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
		in := util.NewInt(x)
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

func TestPolynomialSparseNativeMonomialGrowthAcross129Groups(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	coefficients := make([]int32, 129)
	powers := make([][]MonomialPower, 129)
	for i := range coefficients {
		coefficients[i] = int32(i%47 + 1)
		powers[i] = []MonomialPower{{1, uint32(i + 1)}}
	}
	p := polynomialQueryTerms(t, m, coefficients, powers)
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	n, e := sk.NumEntries()
	serviceMust(t, e)
	if n != 129 {
		t.Fatal("independent one-power groups", n)
	}
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.CloseChecked()) })
	input := util.NewInt(3)
	ok, e := s.Add(&input, p)
	serviceMust(t, e)
	if !ok {
		t.Fatal("constant-in-x0 sample rejected")
	}
	ready, e := s.Ready()
	serviceMust(t, e)
	if !ready {
		t.Fatal("one sample should fill each single-power group")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	for i := 0; i < 2; i++ {
		ok, e = s.Interpolate(out)
		serviceMust(t, e)
		if !ok {
			t.Fatal("single-variable coefficients were not reconstructed")
		}
		equal, e := m.Equal(out.root.value, p)
		serviceMust(t, e)
		if !equal {
			t.Fatal("native monomial/coefficient association", i)
		}
		if s.interpolation.active || s.interpolation.mons != nil {
			t.Fatal("native local span survived completed cleanup")
		}
		polynomialSparsePhysical(t, s, out, []Polynomial{p})
	}
}
