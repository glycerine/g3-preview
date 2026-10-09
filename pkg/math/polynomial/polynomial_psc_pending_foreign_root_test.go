// Source: src/math/polynomial/polynomial.cpp:5751-5810,2481-2495.
package polynomial

import (
	"testing"
)

func TestPolynomialPSCActualSingletonINCReceiptAndAbort(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		m.integer.Budget().FailAfter(ordinal)
		cause := m.PSCChain(p, q, 0, S)
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.Budget().FailAfter(-1)
		owned := m.pscReceipts.singleton
		if owned == (Monomial{}) {
			serviceMust(t, m.CleanupPSC())
			serviceMust(t, S.Close())
			continue
		}
		if !numericPhysicalReservationRefusal(cause) || !m.singletonReference || m.singletonMonomial != owned {
			t.Fatal("receipt does not identify actual completed source INC", ordinal, cause, owned, m.singletonMonomial)
		}
		before, e := m.monomialRefCount(owned)
		serviceMust(t, e)
		if before < 2 {
			t.Fatal("actual unit plus source local reference", before)
		}
		serviceMust(t, m.CleanupPSC())
		if m.pscReceipts.singleton != (Monomial{}) || m.singletonReference {
			t.Fatal("owning abort retained completed matching receipt", ordinal)
		}
		after, e := m.monomialRefCount(owned)
		serviceMust(t, e)
		if after >= before {
			t.Fatal("no reached owning DEC", before, after)
		}
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		serviceMust(t, S.Close())
		t.Log("actual singleton INC refusal frontier", ordinal)
		return
	}
	t.Fatal("no real constant-constructor unjoined INC reached")
}
