package polynomial

import (
	"fmt"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialFlipSignRetainedNLSATLeadingOrderAndContent(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	// Storage puts y first, but graded lex puts x^3 first. Looking only at
	// the first stored coefficient would choose the wrong sign in both cases.
	positive := polynomialQueryTerms(t, c, []int32{2, -4}, [][]MonomialPower{{{0, 3}}, {{1, 1}}})
	negative := polynomialQueryTerms(t, c, []int32{-2, 4}, [][]MonomialPower{{{0, 3}}, {{1, 1}}})
	result, err := full.FlipSignIfLeadingMonomialNegative(positive)
	serviceMust(t, err)
	if result != positive {
		t.Fatal("positive leading polynomial did not alias")
	}
	result, err = full.FlipSignIfLeadingMonomialNegative(negative)
	serviceMust(t, err)
	if result == negative {
		t.Fatal("negative leading polynomial aliased input")
	}
	info, err := c.Info(result)
	serviceMust(t, err)
	if info.RefCount != 0 {
		t.Fatal("source factory acquired an output reference")
	}
	result = polynomialCaller(t, c, result)
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{3, 0, 0}: 2, {0, 1, 0}: -4})
	polynomialQueryExpect(t, c, negative, map[[3]uint32]int64{{3, 0, 0}: -2, {0, 1, 0}: 4})
	alias, err := full.FlipSignIfLeadingMonomialNegative(result)
	serviceMust(t, err)
	if alias != result {
		t.Fatal("second normalization changed identity")
	}
	zero, err := full.FlipSignIfLeadingMonomialNegative(c.zero)
	serviceMust(t, err)
	if zero != c.zero {
		t.Fatal("source zero alias")
	}
	constant := polynomialQueryTerms(t, c, []int32{-6}, [][]MonomialPower{nil})
	constantResult, err := full.FlipSignIfLeadingMonomialNegative(constant)
	serviceMust(t, err)
	constantResult = polynomialCaller(t, c, constantResult)
	polynomialQueryExpect(t, c, constantResult, map[[3]uint32]int64{{}: 6})
}

func TestPolynomialFlipSignActualAllocationCleanupAndRetry(t *testing.T) {
	complete := false
	for stop := int64(0); stop < 128 && !complete; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, []int32{-7, 11}, [][]MonomialPower{{{0, 3}}, {{1, 1}}})
			c.integer.Budget().FailAfter(stop)
			result, cause := m.FlipSignIfLeadingMonomialNegative(p)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual flip refusal", cause)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupFlipSignIfLeadingMonomialNegative())
			if result != (Polynomial{}) {
				result = polynomialCaller(t, c, result)
				polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{3, 0, 0}: 7, {0, 1, 0}: -11})
			}
			if cause == nil {
				complete = true
			}
			retry, err := m.FlipSignIfLeadingMonomialNegative(p)
			serviceMust(t, err)
			retry = polynomialCaller(t, c, retry)
			polynomialQueryExpect(t, c, retry, map[[3]uint32]int64{{3, 0, 0}: 7, {0, 1, 0}: -11})
		})
	}
	if !complete {
		t.Fatal("no completed normalization call")
	}
}

func polynomialFlipBigNegativeInput(t *testing.T, c *MultivariatePolynomialCore) Polynomial {
	t.Helper()
	var big unumeral.Int
	serviceMust(t, c.integer.SetLimbs(&big, []uint32{1, 0, 7}))
	serviceMust(t, c.integer.Neg(&big))
	t.Cleanup(func() { serviceMust(t, c.integer.Del(&big)) })
	x, err := c.monomials.MkVariable(0, 3)
	serviceMust(t, err)
	p, err := c.MkTerms([]unumeral.Int{big}, []Monomial{x})
	serviceMust(t, err)
	return polynomialCaller(t, c, p)
}
func TestPolynomialFlipSignBIGAllocationCleanupAndRetry(t *testing.T) {
	complete := false
	for stop := int64(0); stop < 128 && !complete; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialFlipBigNegativeInput(t, c)
			c.integer.Budget().FailAfter(stop)
			result, cause := m.FlipSignIfLeadingMonomialNegative(p)
			c.integer.Budget().FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual BIG flip refusal", cause)
			}
			serviceMust(t, m.CleanupFlipSignIfLeadingMonomialNegative())
			if result != (Polynomial{}) {
				result = polynomialCaller(t, c, result)
			}
			if cause == nil {
				complete = true
			}
			retry, err := m.FlipSignIfLeadingMonomialNegative(p)
			serviceMust(t, err)
			retry = polynomialCaller(t, c, retry)
			coefficient, err := c.Coefficient(retry, 0)
			serviceMust(t, err)
			value, err := c.integer.String(&coefficient)
			serviceMust(t, err)
			if value != "129127208515966861313" {
				t.Fatal("BIG coefficient content/sign changed", value)
			}
		})
	}
	if !complete {
		t.Fatal("no completed BIG normalization call")
	}
}

func TestPolynomialFlipSignRejectedCallPreservesIndependentNeg(t *testing.T) {
	qualified := false
	for stop := int64(0); stop < 64 && !qualified; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, []int32{-7, 11}, [][]MonomialPower{{{0, 3}}, {{1, 1}}})
			warm, err := m.FlipSignIfLeadingMonomialNegative(p)
			serviceMust(t, err)
			warm = polynomialCaller(t, c, warm)
			old := m.flipLeadingSignCall
			// A BIG coefficient forces a real numerical allocation even after
			// the scalar negation has warmed the reusable polynomial buffers.
			bigInput := polynomialFlipBigNegativeInput(t, c)
			var other polynomialNegInvocationReceipt
			c.integer.Budget().FailAfter(stop)
			result, cause := c.negWithReceipt(bigInput, &other)
			c.integer.Budget().FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("independent negation refusal", cause)
			}
			if c.matchesNegInvocation(&other) {
				used, state := c.integer.Budget().Used(), c.memory.State()
				first := c.negInvocationSourceCause(&other)
				serviceMust(t, m.CleanupFlipSignIfLeadingMonomialNegative())
				if !c.matchesNegInvocation(&other) || c.integer.Budget().Used() != used || c.memory.State() != state || c.negInvocationSourceCause(&other) != first {
					t.Fatal("stale flip owner consumed independent Neg")
				}
				_, entry := m.FlipSignIfLeadingMonomialNegative(bigInput)
				if entry == nil || m.flipLeadingSignCall != old {
					t.Fatal("rejected entry acquired an older invocation", entry)
				}
				qualified = true
			}
			serviceMust(t, c.cleanupNegInvocation(&other, true))
			if result != (Polynomial{}) {
				result = polynomialCaller(t, c, result)
			}
		})
	}
	if !qualified {
		t.Fatal("no actual independent negation stop exercised")
	}
}
