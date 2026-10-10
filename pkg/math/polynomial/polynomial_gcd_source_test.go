// Source: src/math/polynomial/polynomial.cpp:3555-3672,4352-4506;
// src/math/polynomial/polynomial.h:858-868.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialGCDSourceZeroIdentityAndConstantContent(t *testing.T) {
	for route := uint32(0); route < 5; route++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		negative := polynomialQueryTerms(t, c, []int32{-6, -10}, [][]MonomialPower{{{0, 1}}, nil})
		positive := polynomialQueryTerms(t, c, []int32{6, 10}, [][]MonomialPower{{{0, 1}}, nil})
		u, v, want := c.zero, negative, positive
		switch route {
		case 1:
			u, v = negative, c.zero
		case 2:
			u, v = negative, negative
		case 3:
			u = polynomialQueryTerms(t, c, []int32{-14}, [][]MonomialPower{nil})
			want = polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
		case 4:
			u = polynomialQueryTerms(t, c, []int32{15}, [][]MonomialPower{nil})
			want = c.unit
		}
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		serviceMust(t, full.GCD(u, v, r))
		eq, e := c.Equal(r.root.value, want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("independent source shortcut/content", route)
		}
		serviceMust(t, r.Close())
	}
}

func TestPolynomialGCDPRSIndependentFactorTable(t *testing.T) {
	for route := uint32(0); route < 3; route++ {
		full := polynomialPSCFullFixture(t)
		full.usePRSGCD = true // exercise the actual source parameter dispatch, not an alternative algorithm
		c := full.core
		u := polynomialQueryTerms(t, c, []int32{6, 6, -12}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})   //6(x-1)(x+2)
		v := polynomialQueryTerms(t, c, []int32{10, 20, -30}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil}) //10(x-1)(x+3)
		want := polynomialQueryTerms(t, c, []int32{2, -2}, [][]MonomialPower{{{0, 1}}, nil})
		if route == 1 {
			u, v = v, u
		}
		if route == 2 {
			v = polynomialQueryTerms(t, c, []int32{10, 30, 20}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			want = polynomialQueryTerms(t, c, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
		} //10(x+1)(x+2)
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		serviceMust(t, full.GCD(u, v, r))
		eq, e := c.Equal(r.root.value, want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("independent integer content and primitive factor", route)
		}
		serviceMust(t, r.Close())
	}
}

func TestPolynomialGCDDefaultUnivariateModularImages(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	u := polynomialQueryTerms(t, c, []int32{6, 6, -12}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	v := polynomialQueryTerms(t, c, []int32{10, 20, -30}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	want := polynomialQueryTerms(t, c, []int32{2, -2}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	if full.usePRSGCD {
		t.Fatal("source default parameter")
	}
	serviceMust(t, full.GCD(u, v, r))
	eq, e := c.Equal(r.root.value, want)
	serviceMust(t, e)
	if !eq {
		t.Fatal("default modular image/reconstruction result")
	}
	serviceMust(t, r.Close())
}

func TestPolynomialGCDScalarProductSourceIndependentOrder(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 1}}, nil})
	a := unumeral.NewInt(3)
	got, e := full.gcdScalarProduct(&a, p, q)
	serviceMust(t, e)
	got = polynomialCaller(t, c, got)
	polynomialQueryExpect(t, c, got, map[[3]uint32]int64{{2, 0, 0}: 3, {}: -3})
}

func TestPolynomialGCDDefaultMultivariateIndependentCommonFactor(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	u := polynomialQueryTerms(t, c, []int32{6, 6, 6, 6}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}})
	v := polynomialQueryTerms(t, c, []int32{10, 10, 10, 10}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{1, 2}}, {{0, 1}}, {{1, 1}}})
	want := polynomialQueryTerms(t, c, []int32{2, 2}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, full.GCD(u, v, r))
	equal, e := c.Equal(r.root.value, want)
	serviceMust(t, e)
	if !equal {
		t.Fatal("independent 6(x+y)(x+1),10(x+y)(y+1) common factor")
	}
}
