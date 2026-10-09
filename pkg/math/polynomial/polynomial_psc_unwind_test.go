// Source: src/math/polynomial/polynomial.cpp:5698-5704; src/util/ref_buffer.h:38-99; src/util/obj_ref.h:31-112.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialPSCBufferDestructorStopsOlderSourceLocals(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	// Use the same actual guarded four-reference begin and H constructor as
	// optimized_S_e_1; the malformed Size below remains the guard-only fixture.
	var receipt polynomialPSCNextInvocationReceipt
	s, e := m.beginPSCNextInvocation(&receipt)
	serviceMust(t, e)
	serviceMust(t, s.refs[0].assign(m, m.unit))
	serviceMust(t, m.pscBuffer.constructWithNextReceipt(m, &receipt))
	s.bufferConstructed = true
	s.bufferInvocation = m.pscBuffer.invocation
	serviceMust(t, m.pscBuffer.pushWithNextReceipt(m, p, &receipt))
	s.executing = false
	m.pscLeave()
	// Explicit malformed Size fixture: one initialized coefficient/monomial,
	// advertised Size2. This is a reached physical guard, not valid exhaustion.
	serviceMust(t, m.writeWord(p, 8, 2))
	e = m.finishPSCNext(false)
	if !errors.Is(e, arena.ErrBounds) || s.refs[0].value != m.unit || s.cursor != 0 || m.pscBuffer.cursor != 0 || !m.pscBuffer.drainDec {
		t.Fatal("younger H destructor did not stop older locals", e, s, m.pscBuffer)
	}
	rc, z := m.readWord(m.unit, 0)
	serviceMust(t, z)
	if rc != 2 {
		t.Fatal("older INC changed before its source destructor", rc)
	}
	polynomialPSCPhysical(t, m, nil, nil, nil)
	serviceMust(t, m.writeWord(p, 8, 1)) // Restore ONLY this test-created layout.
	if repeated := m.finishPSCNext(false); repeated != e {
		t.Fatal("ordinary destruction replayed stopped DEC", repeated, e)
	}
	serviceMust(t, m.CleanupPSC())
	rc, z = m.readWord(m.unit, 0)
	serviceMust(t, z)
	if rc != 1 || s.active {
		t.Fatal("owning cursor/older completed destruction", rc, s)
	}
	polynomialPSCPhysical(t, m, nil, nil, nil)
}

func TestPolynomialPSCExpiredFactoryReceiptPreservesForeignReplacement(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m := polynomialCellDirectObjectFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		m.integer.Budget().FailAfter(ordinal)
		e = m.PSCChain(p, q, 1, S)
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.Budget().FailAfter(-1)
		old := m.pscReceipts.factory
		if old == 0 || m.monomials.creation.generation != old || m.monomials.creation.stage == 0 {
			serviceMust(t, m.CleanupPSC())
			serviceMust(t, S.Close())
			continue
		}
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("not an actual owned factory refusal", ordinal, e)
		}
		// The caller explicitly completes the old actual callee, then creates a
		// new independent factory continuation. No PSC source body is executing.
		serviceMust(t, m.monomials.CleanupCreation())
		for m.monomials.nextVar < 17 {
			_, e = m.monomials.MkVar()
			serviceMust(t, e)
		}
		terms := make([]MonomialPower, 17)
		for i := range terms {
			terms[i] = MonomialPower{uint32(i), 1}
		}
		m.integer.Budget().FailAfter(0)
		_, e = m.monomials.MkMonomial(terms)
		if !numericPhysicalReservationRefusal(e) || m.monomials.creation.stage == 0 || m.monomials.creation.generation == old {
			t.Fatal("actual foreign replacement was not reached", ordinal, e, m.monomials.creation)
		}
		foreign := m.monomials.creation
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.Budget().FailAfter(-1)
		serviceMust(t, m.CleanupPSC())
		if m.monomials.creation != foreign {
			t.Fatal("expired PSC lease drained foreign continuation", foreign, m.monomials.creation)
		}
		polynomialPSCPhysicalVectors(t, m, nil, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		serviceMust(t, m.monomials.CleanupCreation())
		serviceMust(t, S.Close())
		t.Log("actual old owned refusal", ordinal, "old/new generation", old, foreign.generation)
		return
	}
	t.Fatal("no genuine source factory refusal discovered")
}
