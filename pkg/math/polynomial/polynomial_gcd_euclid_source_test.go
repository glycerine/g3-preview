package polynomial

// Source: src/math/polynomial/polynomial.cpp:3639-3672.

import (
	"testing"
)

func TestPolynomialGCDEuclidSourceShortcutsAndIntegerContent(t *testing.T) {
	for route := uint32(0); route < 4; route++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		serviceMust(t, c.modular.SetZPUint64(11))
		p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
		u, v, want := c.zero, p, p
		if route == 1 {
			u, v = p, c.zero
		}
		if route == 2 {
			u, v = p, p
		}
		if route == 3 {
			u = polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
			v = polynomialQueryTerms(t, c, []int32{4}, [][]MonomialPower{nil})
			want = u
		}
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, r.Close()) })
		serviceMust(t, full.gcdEuclid(u, v, &r.root))
		same, e := c.Equal(r.root.value, want)
		serviceMust(t, e)
		if !same {
			t.Fatal("actual Euclid zero/identity/Integer-content branch", route)
		}
	}
}

func TestPolynomialGCDDispatcherConstantNormalizesSourceScalarSets(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	// Existing coefficients stay12/16, but each source scalar SET normalizes
	// to1/5 before the modular wrapper delegates their integer GCD.
	u := polynomialQueryTerms(t, c, []int32{12}, [][]MonomialPower{nil})
	v := polynomialQueryTerms(t, c, []int32{16}, [][]MonomialPower{nil})
	serviceMust(t, c.modular.SetZPUint64(11))
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, full.GCD(u, v, r))
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{}: 1})
}
