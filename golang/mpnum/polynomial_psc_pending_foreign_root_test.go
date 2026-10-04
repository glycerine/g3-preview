// Source: src/math/polynomial/polynomial.cpp:5751-5810,2481-2495.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialPSCMatchingPlacementPreservesLaterSingleton(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, S.Close()) })
		m.integer.budget.FailAfter(ordinal)
		cause := m.PSCChain(p, q, 1, S)
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.budget.FailAfter(-1)
		g := m.pscReceipts.generation
		if g == 0 || m.pending.pscOwner != g || m.pending.raw == (memory.ObjectRef{}) || m.singletonReference || m.generatedActive || m.monomials.creation.stage != 0 {
			serviceMust(t, m.CleanupPSC())
			serviceMust(t, S.Close())
			continue
		}
		if !numericPhysicalReservationRefusal(cause) {
			t.Fatal("not real owned placement failure", ordinal, cause)
		}
		pending := m.pending
		_, e = m.MkVariable(0, 17)
		if !errors.Is(e, ErrPolynomialPending) || !m.singletonReference || m.pending != pending {
			t.Fatal("foreign singleton not actually reached", ordinal, e, m.singletonReference)
		}
		foreign := m.singletonMonomial
		rc, e := m.monomialRefCount(foreign)
		serviceMust(t, e)
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		serviceMust(t, m.CleanupPSC())
		if !m.singletonReference || m.singletonMonomial != foreign {
			t.Fatal("matching old PSC placement swept later caller singleton", ordinal, foreign, rc, m.singletonReference, m.singletonMonomial)
		}
		after, e := m.monomialRefCount(foreign)
		serviceMust(t, e)
		if after != rc {
			t.Fatal("caller INC consumed", rc, after)
		}
		serviceMust(t, m.CleanupConstruction())
		serviceMust(t, S.Close())
		return
	}
	t.Fatal("no actual PSC placement frontier reached")
}

func TestPolynomialPSCActualSingletonINCReceiptAndAbort(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		m.integer.budget.FailAfter(ordinal)
		cause := m.PSCChain(p, q, 0, S)
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.budget.FailAfter(-1)
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
