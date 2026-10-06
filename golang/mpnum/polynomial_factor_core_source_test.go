// Source: src/math/polynomial/polynomial.cpp:6737-6830;
// src/test/polynomial.cpp:1017-1190 (dormant driver after return at 1825).
package mpnum

import (
	"strconv"
	"testing"
)

func polynomialFactorAssertReconstruction(t *testing.T, full *MultivariatePolynomialManager, p Polynomial, out *PolynomialFactors, want uint32) {
	t.Helper()
	n, e := out.Distinct()
	serviceMust(t, e)
	if n != want {
		t.Fatal("source distinct factors", n, want)
	}
	result, e := NewPolynomialRef(full.core)
	serviceMust(t, e)
	serviceMust(t, out.Multiply(result))
	same, e := full.core.Equal(p, result.root.value)
	serviceMust(t, e)
	if !same {
		t.Fatal("independent factor reconstruction")
	}
	serviceMust(t, result.Close())
}
func TestPolynomialFactorCoreLiteralRepeatedAndDispatcherCases(t *testing.T) {
	cases := []struct {
		name   string
		coeff  []int32
		powers [][]MonomialPower
		count  uint32
	}{
		{"repeated", []int32{1, 2, 1}, [][]MonomialPower{{}, {{0, 1}}, {{0, 2}}}, 1},
		{"univariate", []int32{-1, 1}, [][]MonomialPower{{}, {{0, 2}}}, 2},
		{"multivariateQuadratic", []int32{-1, 1}, [][]MonomialPower{{{0, 2}}, {{1, 2}}}, 2},
		{"degreeOneBeforeUnivariate", []int32{2, 2}, [][]MonomialPower{{}, {{0, 1}}}, 1},
		{"literalHigherDegreeTODO", []int32{1, 1}, [][]MonomialPower{{{0, 3}}, {{1, 3}}}, 1},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			var full *MultivariatePolynomialManager
			if entry.name == "univariate" {
				full = polynomialPSCFullFixtureWithLimit(t, 2<<20)
			} else {
				full = polynomialPSCFullFixture(t)
			}
			var primes *PrimeGenerator
			if entry.name == "univariate" {
				var e error
				primes, e = NewPrimeGenerator(full.originalBudget)
				serviceMust(t, e)
				full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
				t.Cleanup(func() { serviceMust(t, primes.Close()) })
			}
			p := polynomialQueryTerms(t, full.core, entry.coeff, entry.powers)
			out, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			serviceMust(t, full.Factor(p, out, DefaultUPolynomialFactorParams()))
			polynomialFactorAssertReconstruction(t, full, p, out, entry.count)
			extra := uint64(0)
			if primes != nil {
				extra = primes.Stats().TotalBytes
			}
			polynomialFactorsAuditExtra(t, out, []Polynomial{p}, nil, extra)
			serviceMust(t, out.CloseChecked())
		})
	}
}
func TestPolynomialFactorCoreEveryReachedRepeatedReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 500 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			p := polynomialQueryTerms(t, full.core, []int32{1, 2, 1}, [][]MonomialPower{{}, {{0, 1}}, {{0, 2}}})
			out, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			full.originalBudget.FailAfter(ordinal)
			cause := full.Factor(p, out, DefaultUPolynomialFactorParams())
			polynomialFactorsAudit(t, out, []Polynomial{p}, nil)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("literal body/source refusal", cause)
			}
			full.originalBudget.FailAfter(-1)
			serviceMust(t, full.CleanupFactor())
			if cause == nil {
				success = true
				polynomialFactorAssertReconstruction(t, full, p, out, 1)
			}
			serviceMust(t, out.CloseChecked())
		})
	}
	if !success {
		t.Fatal("first source Factor success absent")
	}
}
func TestPolynomialFactorZeroPreservesExistingOutputRows(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, full.core, []int32{1, 1}, [][]MonomialPower{{}, {{0, 1}}})
	out, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	serviceMust(t, out.Push(p, 3))
	before, e := out.Distinct()
	serviceMust(t, e)
	factors, degrees := out.factors, out.degrees
	serviceMust(t, full.Factor(full.core.zero, out, DefaultUPolynomialFactorParams()))
	after, e := out.Distinct()
	serviceMust(t, e)
	if out.constant != NewInt(0) || before != after || factors.raw != out.factors.raw || factors.pending != out.factors.pending || degrees.raw != out.degrees.raw {
		t.Fatal("zero wrapper must preserve rows", before, after)
	}
	serviceMust(t, out.CloseChecked())
}
