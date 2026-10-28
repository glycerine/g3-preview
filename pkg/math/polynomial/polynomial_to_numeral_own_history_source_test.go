package polynomial

// Source: src/math/polynomial/upolynomial.h:426-450.

import (
	"strconv"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialToNumeralSelectedVariableMatchedOwnHistory(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 160 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			z := full.core.integer
			p := polynomialQueryTerms(t, full.core, []int32{1, 3}, [][]MonomialPower{{}, {{0, 1}}})
			// Construct an actual BIG constant coefficient before the fault stream.
			var big unumeral.Int
			one := unumeral.NewInt(1)
			serviceMust(t, z.Mul2k(&one, 521, &big))
			var coefficient unumeral.Int
			serviceMust(t, z.Copy(&coefficient, &big))
			bp, e := full.core.MkConstantTake(&coefficient)
			serviceMust(t, e)
			serviceMust(t, full.core.IncRef(bp))
			original := p
			p, e = full.core.Add(p, bp)
			serviceMust(t, e)
			serviceMust(t, full.core.IncRef(p))
			var output upolyVectorState
			var receipt polynomialFactorNumeralVectorReceipt
			z.Budget().FailAfter(prefix)
			cause := full.toNumeralVectorVariableWithReceipt(p, 0, &output, &receipt)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, bp, original}, []*unumeral.Int{&big}, 0)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal(cause)
			}
			if full.factorNumeralVector.phase == 1 && z.PendingAllocationObject() != (umemory.ObjectRef{}) && full.factorNumeralVector.numeric.Cell.Object != z.PendingAllocationObject() {
				t.Fatal("reached original coefficient SET cell missing concrete vector child")
			}
			own := full.cleanupFactorNumeralVector(&receipt, false)
			if own != nil {
				h, ok := full.factorNumeralVectorDiagnostic(&receipt)
				if !ok || h.cleanupCause != own || !numericPhysicalReservationRefusal(own) {
					t.Fatal("actual first false-own history", own, h)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, bp, original}, []*unumeral.Int{&big}, 0)
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, full.cleanupFactorNumeralVector(&receipt, false))
			serviceMust(t, full.upm.reset(&output))
			full.upm.freeBacking(&output)
			z.Budget().FailAfter(0)
			own = full.cleanupFactorNumeralVector(&receipt, true)
			if own != nil {
				h, ok := full.factorNumeralVectorDiagnostic(&receipt)
				if !ok || h.cleanupCause == nil || !numericPhysicalReservationRefusal(own) {
					t.Fatal("actual first true-own history", own, h)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, bp, original}, []*unumeral.Int{&big}, 0)
			}
			h, ok := full.factorNumeralVectorDiagnostic(&receipt)
			if !ok {
				t.Fatal("actual own tuple history")
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, full.cleanupFactorNumeralVector(&receipt, true))
			after, ok := full.factorNumeralVectorDiagnostic(&receipt)
			if !ok || after.cleanupCause != h.cleanupCause || after.bodyCause != h.bodyCause || after.sourceCause != h.sourceCause {
				t.Fatal("own retry changed first histories", h, after)
			}
			if cause == nil {
				success = true
			}
			serviceMust(t, full.core.DecRef(p))
			serviceMust(t, full.core.DecRef(bp))
			serviceMust(t, z.Del(&big))
		})
	}
	if !success {
		t.Fatal("first success absent")
	}
}
