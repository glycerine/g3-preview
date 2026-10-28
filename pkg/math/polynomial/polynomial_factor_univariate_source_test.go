package polynomial

// Source: src/math/polynomial/polynomial.cpp:6683-6725.

import (
	"reflect"
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestPolynomialFactorUnivariateLiteralSquareFreeBackend(t *testing.T) {
	full := polynomialPSCFullFixtureWithLimit(t, 2<<20)
	primes, e := primegen.NewGenerator(full.core.integer.Budget())
	serviceMust(t, e)
	full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
	p := polynomialQueryTerms(t, full.core, []int32{-1, 1}, [][]MonomialPower{{}, {{0, 2}}})
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	var receipt polynomialFactorUnivariateInvocationReceipt
	serviceMust(t, full.factorSquareFreePrimitiveUnivariateWithReceipt(p, fs, 3, DefaultUPolynomialFactorParams(), &receipt))
	n, e := fs.Distinct()
	serviceMust(t, e)
	if n != 2 {
		t.Fatal("literal univariate factors", n)
	}
	serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, false))
	serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, true))
	serviceMust(t, fs.CloseChecked())
	// The borrowed generator is finalized after the full owner fixture ends.
	t.Cleanup(func() { serviceMust(t, primes.Close()) })
}

func TestPolynomialFactorUnivariateEveryActualReservationPrefix(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 240 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			full := polynomialPSCFullFixtureWithLimit(t, 2<<20)
			primes, e := primegen.NewGenerator(full.core.integer.Budget())
			serviceMust(t, e)
			full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
			p := polynomialQueryTerms(t, full.core, []int32{-1, 1}, [][]MonomialPower{{}, {{0, 2}}})
			fs, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			var receipt polynomialFactorUnivariateInvocationReceipt
			full.core.integer.Budget().FailAfter(prefix)
			cause := full.factorSquareFreePrimitiveUnivariateWithReceipt(p, fs, 3, DefaultUPolynomialFactorParams(), &receipt)
			polynomialFactorsAuditExtra(t, fs, []Polynomial{p}, nil, primes.Stats().TotalBytes)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual univariate source", cause)
			}
			if cause == nil {
				n, e := fs.Distinct()
				serviceMust(t, e)
				if n != 2 {
					t.Fatal("actual success factors", n)
				}
				success = true
			}
			// Retained locals are inspected by their concrete source frame before lifting.
			s := &full.factorUnivariate
			if receipt.frame != s || receipt.invocation != s.invocation {
				t.Fatal("lost actual invoked frame")
			}
			if s.sourceCause != nil {
				before := *s
				retry := full.finishFactorUnivariateSource(&receipt)
				if retry != s.sourceCause || !reflect.DeepEqual(before, *s) {
					t.Fatal("source retry replayed suffix", retry)
				}
			}
			full.core.integer.Budget().FailAfter(-1)
			serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, false))
			serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, true))
			serviceMust(t, fs.CloseChecked())
			// Borrowed generator survives full owner destruction.
			t.Cleanup(func() { serviceMust(t, primes.Close()) })
		})
	}
	if !success {
		t.Fatal("first actual univariate success absent")
	}
}

func TestPolynomialFactorUnivariateLiteralSignParity(t *testing.T) {
	for _, k := range []uint32{1, 2, 3} {
		t.Run(strconv.FormatUint(uint64(k), 10), func(t *testing.T) {
			full := polynomialPSCFullFixtureWithLimit(t, 2<<20)
			primes, e := primegen.NewGenerator(full.core.integer.Budget())
			serviceMust(t, e)
			full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
			p := polynomialQueryTerms(t, full.core, []int32{1, -1}, [][]MonomialPower{{}, {{0, 2}}})
			fs, e := NewPolynomialFactors(full)
			serviceMust(t, e)
			var receipt polynomialFactorUnivariateInvocationReceipt
			serviceMust(t, full.factorSquareFreePrimitiveUnivariateWithReceipt(p, fs, k, DefaultUPolynomialFactorParams(), &receipt))
			want := unumeral.NewInt(1)
			if k%2 == 1 {
				want = unumeral.NewInt(-1)
			}
			if fs.constant != want {
				t.Fatal("literal odd-k flip", fs.constant, want)
			}
			serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, false))
			serviceMust(t, full.cleanupFactorUnivariateInvocation(&receipt, true))
			serviceMust(t, fs.CloseChecked())
			t.Cleanup(func() { serviceMust(t, primes.Close()) })
		})
	}
}
