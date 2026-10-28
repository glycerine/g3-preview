package polynomial

// Source: src/test/polynomial.cpp:702-723,741-821,1814-1825;
// src/math/polynomial/polynomial.cpp:5390-5480,5654-5901.

import (
	"testing"
)

func polynomialPSCFullFixture(t *testing.T) *MultivariatePolynomialManager {
	t.Helper()
	return polynomialPSCFullFixtureWithLimit(t, 1<<20)
}
func polynomialPSCFullFixtureWithLimit(t *testing.T, limit uint64) *MultivariatePolynomialManager {
	t.Helper()
	q, _ := rationalManagerWithLimit(t, false, limit)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{
		Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil },
	}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	for i := 0; i < 11; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	return m
}
func TestPolynomialPSCRegisteredFirstFixture(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	first := polynomialQueryTerms(t, m, []int32{6, -1}, [][]MonomialPower{{{0, 1}}, {{0, 2}}})
	S, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, S.Close()) })
	serviceMust(t, full.PSCChain(p, q, 1, S))
	n, e := S.Size()
	serviceMust(t, e)
	if n == 0 {
		t.Fatal("active source fixture produced empty chain")
	}
	a, e := S.Get(0)
	serviceMust(t, e)
	same, e := m.Equal(a, first)
	serviceMust(t, e)
	if !same {
		neg, e := m.Neg(first)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(neg))
		same, e = m.Equal(a, neg)
		serviceMust(t, e)
		serviceMust(t, m.DecRef(neg))
	}
	if !same {
		t.Fatal("source first principal coefficient ENSURE")
	}
}
