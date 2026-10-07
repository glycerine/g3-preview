// Source: src/math/polynomial/polynomial.cpp:3769-4506.
package polynomial

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialGCDDefaultIntegerIndependentCommonFactors(t *testing.T) {
	for _, multivariate := range []bool{false, true} {
		t.Run(map[bool]string{false: "univariate", true: "multivariate"}[multivariate], func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			powers := [][]MonomialPower{{{0, 1}}, nil}
			coeffs := []int32{1, 1}
			if multivariate {
				powers = [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil}
				coeffs = []int32{1, 1, 1}
			}
			common := polynomialQueryTerms(t, c, coeffs, powers)
			a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
			b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
			p, e := c.Mul(common, a)
			serviceMust(t, e)
			polynomialCaller(t, c, p)
			q, e := c.Mul(common, b)
			serviceMust(t, e)
			polynomialCaller(t, c, q)
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
			field, e := c.modular.Modular()
			serviceMust(t, e)
			if field || !full.useSparseGCD || full.usePRSGCD {
				t.Fatal("actual default dispatch not used", field, full.useSparseGCD, full.usePRSGCD)
			}
			serviceMust(t, full.GCD(p, q, r))
			same, e := c.Equal(common, r.root.value)
			serviceMust(t, e)
			if !same {
				t.Fatal("independent shared-factor identity")
			}
			field, e = c.modular.Modular()
			serviceMust(t, e)
			if field {
				t.Fatal("completed scoped mode not restored")
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, nil, r.header)
		})
	}
}

func TestPolynomialGCDDefaultIntegerEveryReservationPrefix(t *testing.T) {
	for _, multivariate := range []bool{false, true} {
		t.Run(map[bool]string{false: "univariate", true: "multivariate"}[multivariate], func(t *testing.T) {
			success := false
			failures := 0
			for ordinal := int64(0); ordinal < 1024 && !success; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					full := polynomialPSCFullFixture(t)
					c := full.core
					powers := [][]MonomialPower{{{0, 1}}, nil}
					coeffs := []int32{1, 1}
					if multivariate {
						powers = [][]MonomialPower{{{0, 1}}, {{1, 1}}, nil}
						coeffs = []int32{1, 1, 1}
					}
					common := polynomialQueryTerms(t, c, coeffs, powers)
					a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
					b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
					p, e := c.Mul(common, a)
					serviceMust(t, e)
					polynomialCaller(t, c, p)
					q, e := c.Mul(common, b)
					serviceMust(t, e)
					polynomialCaller(t, c, q)
					r, e := NewPolynomialRef(c)
					serviceMust(t, e)
					t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
					c.integer.Budget().FailAfter(ordinal)
					cause := full.GCD(p, q, r)
					polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, nil, r.header)
					c.integer.Budget().FailAfter(-1)
					if cause != nil {
						if !numericPhysicalReservationRefusal(cause) {
							t.Fatal("actual default source failure category", ordinal, cause)
						}
						failures++
						serviceMust(t, full.CleanupGCD())
						polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, nil, r.header)
						return
					}
					success = true
					same, e := c.Equal(common, r.root.value)
					serviceMust(t, e)
					if !same {
						t.Fatal("independent default common factor")
					}
				})
			}
			if !success || failures == 0 {
				t.Fatal("default source refusal-to-success horizon", failures, success)
			}
		})
	}
}

func TestPolynomialGCDDefaultIntegerCRAReconstruction(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	// The common factor's constant exceeds every uint32 source prime. A single
	// balanced modular image cannot be its integer factor; actual CRA is reached.
	as := []util.Int{util.NewInt(1), {}}
	limbs := make([]uint32, 4)
	limbs[0], limbs[3] = 17, 16
	serviceMust(t, z.SetLimbs(&as[1], limbs))
	t.Cleanup(func() { serviceMust(t, z.Del(&as[1])) })
	x, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
	serviceMust(t, e)
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	common, e := c.MkTerms(as, []Monomial{x, unit})
	serviceMust(t, e)
	polynomialCaller(t, c, common)
	a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
	p, e := c.Mul(common, a)
	serviceMust(t, e)
	polynomialCaller(t, c, p)
	q, e := c.Mul(common, b)
	serviceMust(t, e)
	polynomialCaller(t, c, q)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
	serviceMust(t, full.GCD(p, q, r))
	same, e := c.Equal(common, r.root.value)
	serviceMust(t, e)
	if !same {
		t.Fatal("independent large integer common factor")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, []*util.Int{&as[1]}, r.header)
}

func TestPolynomialGCDSourcePrimeExhaustionUsesLiteralPRS(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	// Every source prime divides the leading coefficient. Thus each literal
	// modular-image degree check rejects its image before the source PRS fallback.
	product := big.NewInt(1)
	for _, p := range upolyBigPrimes {
		product.Mul(product, new(big.Int).SetUint64(uint64(p)))
	}
	as := []util.Int{{}, util.NewInt(1)}
	serviceMust(t, z.SetString(&as[0], product.String()))
	t.Cleanup(func() { serviceMust(t, z.Del(&as[0])) })
	x, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
	serviceMust(t, e)
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	common, e := c.MkTerms(as, []Monomial{x, unit})
	serviceMust(t, e)
	polynomialCaller(t, c, common)
	a := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	b := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}}, nil})
	p, e := c.Mul(common, a)
	serviceMust(t, e)
	polynomialCaller(t, c, p)
	q, e := c.Mul(common, b)
	serviceMust(t, e)
	polynomialCaller(t, c, q)
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupGCD()); serviceMust(t, r.Close()) })
	serviceMust(t, full.GCD(p, q, r))
	same, e := c.Equal(common, r.root.value)
	serviceMust(t, e)
	if !same {
		t.Fatal("actual all-prime rejection/source PRS result")
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{common, a, b, p, q, r.root.value}, []*util.Int{&as[0]}, r.header)
}
