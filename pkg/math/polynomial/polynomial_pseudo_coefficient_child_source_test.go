// Source: src/math/polynomial/polynomial.cpp:4939-4959; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"fmt"
	"testing"
)

// A lower degree input still reaches the actual two-buffer Coeff invocation.
// Cancellation at the first while checkpoint isolates this source prelude;
// no later regular product or minus_a SET has begun.
func TestPolynomialPseudoActualCoefficientChildPrefixes(t *testing.T) {
	complete := false
	for prefix := int64(0); prefix < 128 && !complete; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{nil})
			b := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			q, r := polynomialPseudoOutputs(t, c)
			visits := 0
			c.config.Checkpoint = func() bool { visits++; return false }
			d := uint32(99)
			c.integer.Budget().FailAfter(prefix)
			body := c.PseudoDivide(p, b, 0, &d, q, r)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, q.root.value, q.root.incoming, r.root.value, r.root.incoming}, nil, q.header+r.header)
			if body != nil && !numericPhysicalReservationRefusal(body) && !errors.Is(body, ErrUPolynomialCanceled) {
				t.Fatalf("source body prefix=%d: %v", prefix, body)
			}
			if errors.Is(body, ErrUPolynomialCanceled) {
				if visits != 1 || d != 0 {
					t.Fatal("reached actual checkpoint", visits, d)
				}
				complete = true
			}
			beforeRows := [4]uint32{}
			for i, b := range []*polynomialSOM{&c.cheap, &c.cheap2} {
				a, e := b.as.size(c)
				serviceMust(t, e)
				mm, e := b.ms.size(c)
				serviceMust(t, e)
				beforeRows[i*2] = a
				beforeRows[i*2+1] = mm
			}
			serviceMust(t, c.pseudoCleanupPhase(true, false))
			for i, b := range []*polynomialSOM{&c.cheap, &c.cheap2} {
				a, e := b.as.size(c)
				serviceMust(t, e)
				mm, e := b.ms.size(c)
				serviceMust(t, e)
				if a != beforeRows[i*2] || mm != beforeRows[i*2+1] {
					t.Fatal("source-own phase consumed unjoined child rows", prefix)
				}
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, q.root.value, q.root.incoming, r.root.value, r.root.incoming}, nil, q.header+r.header)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.CleanupPseudoDivision())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, b, q.root.value, q.root.incoming, r.root.value, r.root.incoming}, nil, q.header+r.header)
			for _, buf := range []*polynomialSOM{&c.cheap, &c.cheap2} {
				as, e := buf.as.size(c)
				serviceMust(t, e)
				ms, e := buf.ms.size(c)
				serviceMust(t, e)
				if as != 0 || ms != 0 || buf.hasPendingReference {
					t.Fatalf("actual Coeff child omitted by parent owning continuation prefix=%d reduct=%v as=%d ms=%d", prefix, buf == &c.cheap2, as, ms)
				}
			}
			if complete {
				t.Log("actual first completed Coeff/checkpoint", prefix)
			}
		})
	}
	if !complete {
		t.Fatal("first actual checkpoint not reached")
	}
}

func TestPolynomialPseudoReceiptActualFrameAndReuse(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	q, r := polynomialPseudoOutputs(t, c)
	var first polynomialPseudoDivisionInvocationReceipt
	serviceMust(t, c.pseudoEntryWithReceipt(q, r, &first))
	if !c.matchesPseudoInvocation(&first) {
		t.Fatal("actual guarded frame")
	}
	copyFrame := *first.frame
	forged := first
	forged.frame = &copyFrame
	if c.matchesPseudoInvocation(&forged) {
		t.Fatal("copied frame accepted")
	}
	refused := first
	if e := c.pseudoEntryWithReceipt(q, r, &refused); !errors.Is(e, ErrPolynomialPending) || refused != first {
		t.Fatal("busy guard changed incoming receipt", e)
	}
	c.pseudoDivision.executing = false
	serviceMust(t, c.CleanupPseudoDivision())
	var later polynomialPseudoDivisionInvocationReceipt
	serviceMust(t, c.pseudoEntryWithReceipt(q, r, &later))
	if later.invocation == first.invocation || c.matchesPseudoInvocation(&first) || !c.matchesPseudoInvocation(&later) {
		t.Fatal("actual unique invocation")
	}
	before := c.pseudoDivision
	serviceMust(t, c.cleanupPseudoInvocation(&first, true))
	if !testNumericScopeEqual(c.pseudoDivision, before) {
		t.Fatal("stale receipt touched later frame")
	}
	c.pseudoDivision.executing = false
	serviceMust(t, c.cleanupPseudoInvocation(&later, true))
}
