// Source: src/math/polynomial/polynomial.cpp:5358-5480;
// src/test/polynomial.cpp:702-723.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"fmt"
	"math/big"
	"testing"
)

func TestPolynomialResultantIndependentLinearAndQuadraticIdentity(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, {{1, 1}}})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, {{1, 1}}})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, full.Resultant(p, q, 0, r))
	// Monic linear q=x+y: product definition gives p(-y)=y^2+y.
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{0, 2, 0}: 1, {0, 1, 0}: 1})
}
func TestPolynomialResultantRegisteredPSCThirdAssertion(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	chain, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, chain.Close()) })
	serviceMust(t, full.PSCChain(p, q, 1, chain))
	first, e := chain.Get(0)
	serviceMust(t, e)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, full.Resultant(p, q, 1, r))
	same, e := c.Equal(r.root.value, first)
	serviceMust(t, e)
	if !same {
		n, e := c.Neg(r.root.value)
		serviceMust(t, e)
		nr, e := NewPolynomialRef(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, nr.Close()) })
		serviceMust(t, nr.root.assign(c, n))
		same, e = c.Equal(first, n)
		serviceMust(t, e)
	}
	if !same {
		t.Fatal("source third PSC ENSURE against actual Resultant")
	}
}

func TestPolynomialResultantIndependentDegreeAndSignTable(t *testing.T) {
	cases := []struct {
		name string
		a, b []int32
		want int64
	}{
		{"cubic-linear", []int32{-2, 0, 0, 1}, []int32{-3, 1}, -25},
		{"degree-swap", []int32{-3, 1}, []int32{-2, 0, 0, 1}, 25},
		{"equal-quadratic", []int32{-2, 0, 1}, []int32{-3, 0, 1}, 1},
		{"shared-factor", []int32{-2, -1, 1}, []int32{-2, 1}, 0},
		{"constant-first", []int32{3}, []int32{-2, 0, 0, 1}, 27},
		{"constant-second", []int32{-2, 0, 0, 1}, []int32{3}, 27},
		{"both-constant", []int32{3}, []int32{7}, 1},
		{"zero", []int32{0}, []int32{3}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			makePolynomial := func(coeffs []int32) Polynomial {
				var values []int32
				var powers [][]MonomialPower
				for i, a := range coeffs {
					if a == 0 {
						continue
					}
					values = append(values, a)
					if i == 0 {
						powers = append(powers, nil)
					} else {
						powers = append(powers, []MonomialPower{{0, uint32(i)}})
					}
				}
				if len(values) == 0 {
					return c.zero
				}
				return polynomialQueryTerms(t, c, values, powers)
			}
			p, q := makePolynomial(tc.a), makePolynomial(tc.b)
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			serviceMust(t, full.Resultant(p, q, 0, r))
			want := map[[3]uint32]int64{}
			if tc.want != 0 {
				want[[3]uint32{}] = tc.want
			}
			polynomialQueryExpect(t, c, r.root.value, want)
		})
	}
}

func TestPolynomialResultantEveryReservationPrefix(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { c.integer.budget.FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
			c.integer.budget.FailAfter(ordinal)
			cause := full.Resultant(p, q, 1, r)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value}, nil, r.header)
			c.integer.budget.FailAfter(-1)
			if cause != nil {
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("reached physical category", ordinal, cause)
				}
				failures++
				serviceMust(t, full.CleanupGCD())
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value}, nil, r.header)
				return
			}
			success = true
			// Independent elimination in y: (x*y+3)=0 implies y=-3/x;
			// x*f(-3/x) = 9-3*x-3+2*x = 6-x; resultant is x*(6-x).
			polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{2, 0, 0}: -1, {1, 0, 0}: 6})
		})
	}
	if !success || failures == 0 {
		t.Fatal("no full source refusal-to-success horizon", failures, success)
	}
}

func TestPolynomialResultantBigNumeralEveryReservationPrefix(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			z := c.integer
			as := []Int{{}, NewInt(1), NewInt(1), NewInt(2)}
			limbs := make([]uint32, 11)
			limbs[0], limbs[10] = 17, 1
			serviceMust(t, z.SetLimbs(&as[0], limbs))
			t.Cleanup(func() { serviceMust(t, z.Del(&as[0])) })
			powers := [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil}
			ms := make([]Monomial, 4)
			for i := range ms {
				a, e := c.monomials.MkMonomial(powers[i])
				serviceMust(t, e)
				ms[i] = a
			}
			p, e := c.MkTerms(as, ms)
			serviceMust(t, e)
			polynomialCaller(t, c, p)
			q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { z.budget.FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
			z.budget.FailAfter(ordinal)
			cause := full.Resultant(p, q, 1, r)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value}, []*Int{&as[0]}, r.header)
			z.budget.FailAfter(-1)
			if cause != nil {
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual BIG physical category", ordinal, cause)
				}
				failures++
				serviceMust(t, full.CleanupGCD())
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value}, []*Int{&as[0]}, r.header)
				return
			}
			success = true
			// Source resultant product identity: x^2*p(-3/x)=(9*a-3)*x-x^2.
			want := new(big.Int).Lsh(big.NewInt(1), 320)
			want.Add(want, big.NewInt(17))
			want.Mul(want, big.NewInt(9))
			want.Sub(want, big.NewInt(3))
			n, e := c.readWord(r.root.value, 8)
			serviceMust(t, e)
			if n != 2 {
				t.Fatal("BIG resultant term count", n)
			}
			seen := uint32(0)
			for i := uint32(0); i < n; i++ {
				mon, e := c.TermMonomial(r.root.value, i)
				serviceMust(t, e)
				d, e := c.monomials.DegreeOf(mon, 0)
				serviceMust(t, e)
				a, e := c.Coefficient(r.root.value, i)
				serviceMust(t, e)
				decoded := roundPublicationDecode(t, z, &a)
				switch d {
				case 1:
					if decoded.String() != want.String() {
						t.Fatal("independent BIG linear coefficient", decoded.String(), want.String())
					}
					seen |= 1
				case 2:
					if decoded.String() != "-1" {
						t.Fatal("independent quadratic coefficient", decoded.String())
					}
					seen |= 2
				default:
					t.Fatal("unexpected exponent", d)
				}
			}
			if seen != 3 {
				t.Fatal("missing independent source terms", seen)
			}
		})
	}
	if !success || failures == 0 {
		t.Fatal("no BIG refusal-to-success horizon", failures, success)
	}
}

func TestPolynomialResultantAllEightOriginalPSCAssertions(t *testing.T) {
	for i, tc := range polynomialPSCOriginalCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := pscOriginalPolynomial(t, c, tc.p)
			q := pscOriginalPolynomial(t, c, tc.q)
			chain, e := NewPolynomialRefVector(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, chain.Close()) })
			serviceMust(t, full.PSCChain(p, q, tc.x, chain))
			n, e := chain.Size()
			serviceMust(t, e)
			if n == 0 {
				return
			} // Literal source ENSURE is reached only when S.size()>0.
			first, e := chain.Get(0)
			serviceMust(t, e)
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
			serviceMust(t, full.Resultant(p, q, tc.x, r))
			same, e := c.Equal(r.root.value, first)
			serviceMust(t, e)
			if !same {
				neg, e := c.Neg(r.root.value)
				serviceMust(t, e)
				nr, e := NewPolynomialRef(c)
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, nr.Close()) })
				serviceMust(t, nr.root.assign(c, neg))
				same, e = c.Equal(first, neg)
				serviceMust(t, e)
			}
			if !same {
				t.Fatal("original third PSC ENSURE", i)
			}
		})
	}
}
