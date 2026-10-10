// Source: src/math/polynomial/polynomial.cpp:3536-3541,3600-3604;
// src/util/obj_ref.h.
package polynomial

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestPolynomialPrimitivePartReturnedTemporaryPrecedesCallerAssignment(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	input := polynomialQueryTerms(t, c, []int32{2, 2}, [][]MonomialPower{{{0, 1}}, nil})
	old := polynomialICCPUnownedTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{0, 1}}, nil})
	var out polynomialReference
	serviceMust(t, out.assign(c, old))
	var refused umemory.Block
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, p Polynomial) error {
		if p != old {
			return nil
		}
		calls++
		s := m.gcdFrames[0]
		if s.refs[0].root.value != (Polynomial{}) || s.depth != 0 {
			t.Errorf("helper c/i destructors must precede actual caller old-DEC: c=%v sourceLocalDepth=%v", s.refs[0].root.value, s.depth)
		}
		c.integer.Budget().FailAfter(0)
		var e error
		refused, e = c.memory.Allocate(8)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual caller callback refusal", e)
		}
		return e
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	e = m.gcdPrimitivePart(input, 0, &out)
	if !numericPhysicalReservationRefusal(e) || calls != 1 || !out.assigning {
		t.Fatal("actual returned assignment stop", e, calls, out)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{input, out.value, out.incoming}, nil, h.header, []umemory.Block{refused})
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.CleanupGCD())
	serviceMust(t, out.close(c))
	if calls != 1 {
		t.Fatal("callback replay", calls)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}
