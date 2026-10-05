// Source: src/math/polynomial/polynomial.cpp:4507-4528,5114-5156;
// src/util/scoped_numeral.h; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"testing"
)

// A actual first cold-retention refusal leaves a reached derivative invocation.
// The cleanup authority belongs to that invocation, never a later replacement.
func TestPolynomialDerivativeInvocationAcquisitionAndReplacement(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	b := m.originalBudget
	var first polynomialDerivativeInvocationReceipt
	b.FailAfter(0)
	_, e := m.derivativeWithReceipt(p, 0, &first)
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesDerivativeInvocation(&first) {
		t.Fatal("real guarded derivative start", e, first)
	}
	rejected := first
	if _, e = m.derivativeWithReceipt(p, 0, &rejected); !errors.Is(e, ErrPolynomialPending) || rejected != first {
		t.Fatal("rejected acquisition altered receipt", e)
	}
	serviceMust(t, m.cleanupDerivativeInvocation(&first, false))
	serviceMust(t, m.cleanupDerivativeInvocation(&first, true))
	var next polynomialDerivativeInvocationReceipt
	b.FailAfter(0)
	_, e = m.derivativeWithReceipt(p, 0, &next)
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesDerivativeInvocation(&next) || next == first {
		t.Fatal("new actual invocation", e)
	}
	used := b.Used()
	serviceMust(t, m.cleanupDerivativeInvocation(&first, true))
	if !m.matchesDerivativeInvocation(&next) || b.Used() != used {
		t.Fatal("old derivative receipt drained replacement")
	}
	serviceMust(t, m.cleanupDerivativeInvocation(&next, true))
}

func TestPolynomialExactDivInvocationQuickAndCompleted(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	var receipt polynomialExactDivInvocationReceipt
	result, e := m.exactDivWithReceipt(p, q, &receipt)
	serviceMust(t, e)
	polynomialCaller(t, m, result)
	polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: -1})
	if receipt.invocation == 0 {
		t.Fatal("completed source locals never acquired invocation")
	}
	before := receipt
	zero, e := m.exactDivWithReceipt(m.zero, Polynomial{}, &receipt)
	serviceMust(t, e)
	if zero != m.zero || receipt != before {
		t.Fatal("zero route read divisor or acquired locals")
	}
	serviceMust(t, m.cleanupExactDivInvocation(&receipt, false))
	serviceMust(t, m.cleanupExactDivInvocation(&receipt, true))
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, q, result})
}

func polynomialPrimitiveFixture(t *testing.T) *MultivariatePolynomialManager {
	t.Helper()
	q, _ := rationalManager(t, false)
	control := &polynomialEvalControl{limit: 1 << 20}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: control.checkpoint, Cooperation: control.cooperate}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		if control.checkpoints != control.cooperation {
			t.Fatal("actual bounded unit-control pairing", control)
		}
	})
	for i := 0; i < 2; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	return m
}

func TestPolynomialDerivativeReceiptPreservesEarlierGF2CreatedCohort(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, {{1, 2}}})
	prime := NewInt(2)
	serviceMust(t, c.modular.SetZP(&prime))
	var original polynomialDerivativeInvocationReceipt
	first, e := m.derivativeWithReceipt(p, 0, &original)
	serviceMust(t, e)
	polynomialCaller(t, c, first)
	zero, e := c.IsZero(first)
	serviceMust(t, e)
	if !zero || len(m.derivative.orphans) != 1 {
		t.Fatal("actual GF2 source-created zero term", first, m.derivative.orphans)
	}
	older := m.derivative.orphans[0]
	var later polynomialDerivativeInvocationReceipt
	c.integer.budget.FailAfter(0)
	_, e = m.derivativeWithReceipt(p, 0, &later)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, first}, nil, 0)
	c.integer.budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesDerivativeInvocation(&later) {
		t.Fatal("real later guarded cold-retention refusal", e)
	}
	serviceMust(t, m.cleanupDerivativeInvocation(&later, true))
	if len(m.derivative.orphans) != 1 || m.derivative.orphans[0] != older {
		t.Fatal("later invocation swept earlier successful created cohort", older, m.derivative.orphans)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, first}, nil, 0)
	// A later ordinary nonzero INC through the same source derivative engine
	// consumes the older actual zero-ref root and its provenance at the leaf.
	prime = NewInt(3)
	serviceMust(t, c.modular.SetZP(&prime))
	transferred, e := m.Derivative(p, 0)
	serviceMust(t, e)
	polynomialCaller(t, c, transferred)
	polynomialQueryExpect(t, c, transferred, map[[3]uint32]int64{{1, 0, 0}: -1})
	if m.derivative.orphans[0] != (Monomial{}) || m.derivative.orphanInvocations[0] != 0 {
		t.Fatal("actual successful INC did not transfer old authority")
	}
	used := c.integer.budget.Used()
	serviceMust(t, m.cleanupDerivativeInvocation(&original, true))
	serviceMust(t, m.cleanupDerivativeInvocation(&later, true))
	if c.integer.budget.Used() != used {
		t.Fatal("stale receipt refunded foreign current backing")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, first, transferred}, nil, 0)
	serviceMust(t, m.CleanupDerivative())
}

// Source: src/math/polynomial/polynomial.cpp:5114-5156. The completed
// R.reset and NUM/reference suffix precede reuse of the same persistent SOM.
func TestPolynomialExactDivCompletedPreludeAllowsActualSourceReuse(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	var old, current polynomialExactDivInvocationReceipt
	first, e := m.exactDivWithReceipt(p, q, &old)
	serviceMust(t, e)
	polynomialCaller(t, m, first)
	if m.matchesArithmeticInvocation(&old.prelude) {
		t.Fatal("completed actual source rows retained owning invocation")
	}
	second, e := m.exactDivWithReceipt(p, q, &current)
	serviceMust(t, e)
	polynomialCaller(t, m, second)
	if current.prelude.invocation == old.prelude.invocation {
		t.Fatal("source reuse did not acquire a distinct invocation")
	}
	before := m.integer.budget.Used()
	serviceMust(t, m.cleanupExactDivInvocation(&old, true))
	if m.integer.budget.Used() != before {
		t.Fatal("stale completed prelude mutated newer result")
	}
	for _, result := range []Polynomial{first, second} {
		polynomialQueryExpect(t, m, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: -1})
	}
	polynomialExactDivisionPhysical(t, m, []Polynomial{p, q, first, second})
}
