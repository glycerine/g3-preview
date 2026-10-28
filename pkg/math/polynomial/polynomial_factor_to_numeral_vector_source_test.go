package polynomial

// Source: src/math/polynomial/upolynomial.h:401-423;
// src/math/polynomial/polynomial.cpp:6683-6725.

import (
	"strconv"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialFactorToNumeralVectorReadsOriginalAfterNormalize(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, full.core, []int32{6, 9}, [][]MonomialPower{{}, {{0, 1}}})
	var output upolyVectorState
	var receipt polynomialFactorNumeralVectorReceipt
	serviceMust(t, full.toNumeralVectorWithReceipt(p, &output, &receipt))
	if output.size != 2 {
		t.Fatal("literal reserve/size", output.size)
	}
	values := full.upm.values(&output)
	if values[0] != unumeral.NewInt(6) || values[1] != unumeral.NewInt(9) {
		t.Fatal("converter must read original p after unused Normalize", values)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	serviceMust(t, full.cleanupFactorNumeralVector(&receipt, false))
	serviceMust(t, full.upm.reset(&output))
	full.upm.freeBacking(&output)
	serviceMust(t, full.cleanupFactorNumeralVector(&receipt, true))
}

func TestPolynomialFactorToNumeralVectorEveryActualBigPrefix(t *testing.T) {
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
			cause := full.toNumeralVectorWithReceipt(p, &output, &receipt)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, bp, original}, []*unumeral.Int{&big}, 0)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal(cause)
			}
			if full.factorNumeralVector.phase == 1 && z.PendingAllocationObject() != (umemory.ObjectRef{}) && full.factorNumeralVector.numeric.Cell.Object != z.PendingAllocationObject() {
				t.Fatal("reached original coefficient SET cell missing concrete vector child")
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, full.cleanupFactorNumeralVector(&receipt, false))
			serviceMust(t, full.upm.reset(&output))
			full.upm.freeBacking(&output)
			serviceMust(t, full.cleanupFactorNumeralVector(&receipt, true))
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

func TestPolynomialFactorToNumeralVectorRetainsHistoryAcrossActualReuse(t *testing.T) {
	full := polynomialArithmeticDirectObjectFixture(t)
	_, e := full.core.monomials.MkVar()
	serviceMust(t, e)
	p := polynomialQueryTerms(t, full.core, []int32{6, 9}, [][]MonomialPower{{}, {{0, 1}}})
	var output upolyVectorState
	var old polynomialFactorNumeralVectorReceipt
	z := full.core.integer
	z.Budget().FailAfter(0)
	cause := full.toNumeralVectorWithReceipt(p, &output, &old)
	if cause == nil || !numericPhysicalReservationRefusal(cause) {
		t.Fatal("actual first source refusal", cause)
	}
	z.Budget().FailAfter(-1)
	serviceMust(t, full.cleanupFactorNumeralVector(&old, false))
	serviceMust(t, full.upm.reset(&output))
	full.upm.freeBacking(&output)
	serviceMust(t, full.cleanupFactorNumeralVector(&old, true))
	before, ok := full.factorNumeralVectorDiagnostic(&old)
	// Refuse a real normalization object allocation in each invocation.
	if !ok || !numericPhysicalReservationRefusal(before.bodyCause) {
		t.Fatal("actual returned source history missing", before)
	}
	invocation := full.factorNumeralVector.invocation
	z.Budget().FailAfter(0)
	archiveCause := full.toNumeralVectorWithReceipt(p, &output, nil)
	if archiveCause == nil || !numericPhysicalReservationRefusal(archiveCause) {
		t.Fatal("actual new source allocation frontier refusal", archiveCause)
	}
	retained, ok := full.factorNumeralVectorDiagnostic(&old)
	if !ok || retained != before || invocation+1 != full.factorNumeralVector.invocation {
		t.Fatal("allocation refusal changed prior history or did not begin actual new source invocation")
	}
	// This refusal is now inside a real new source call, so acknowledge its
	// actual reached child tuple before the next invocation.
	refused := full.factorNumeralVector.direct
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	z.Budget().FailAfter(-1)
	serviceMust(t, full.cleanupFactorNumeralVector(&refused, false))
	serviceMust(t, full.cleanupFactorNumeralVector(&refused, true))
	var later polynomialFactorNumeralVectorReceipt
	serviceMust(t, full.toNumeralVectorWithReceipt(p, &output, &later))
	values := append([]unumeral.Int(nil), full.upm.values(&output)...)
	used := z.Budget().Used()
	serviceMust(t, full.cleanupFactorNumeralVector(&old, true))
	archived, ok := full.factorNumeralVectorDiagnostic(&old)
	if !ok || archived != before || z.Budget().Used() != used || output.size != 2 || full.upm.values(&output)[0] != values[0] || full.upm.values(&output)[1] != values[1] {
		t.Fatal("old exact acknowledgment changed later converter/history")
	}
	serviceMust(t, full.cleanupFactorNumeralVector(&later, false))
	serviceMust(t, full.upm.reset(&output))
	full.upm.freeBacking(&output)
	serviceMust(t, full.cleanupFactorNumeralVector(&later, true))
}
