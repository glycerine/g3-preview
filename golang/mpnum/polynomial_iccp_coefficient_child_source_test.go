// Source: src/math/polynomial/polynomial.cpp:3512-3530.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestPolynomialICCPFirstCoefficientChildPreservesConstructedEmptyCI(t *testing.T) {
	success, firstCI := false, false
	for prefix := int64(0); prefix < 192 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
			var i Int
			var content, primitive polynomialReference
			var child polynomialICCPInvocationReceipt
			t.Cleanup(func() {
				c.integer.budget.FailAfter(-1)
				serviceMust(t, full.CleanupContentPrimitiveParts())
				serviceMust(t, content.close(c))
				serviceMust(t, primitive.close(c))
				serviceMust(t, c.integer.Del(&i))
			})
			c.integer.budget.FailAfter(prefix)
			body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &child)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*Int{&i}, 0)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("source refusal category", prefix, body)
			}
			if body != nil && child.manager == full && child.frame.ciConstructed && content.value == (Polynomial{}) && child.frame.ci.value == (Polynomial{}) {
				firstCI = true
			}
			c.integer.budget.FailAfter(-1)
			serviceMust(t, full.CleanupContentPrimitiveParts())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*Int{&i}, 0)
			for _, b := range []*polynomialSOM{&c.cheap, &c.cheap2} {
				as, e := b.as.size(c)
				serviceMust(t, e)
				ms, e := b.ms.size(c)
				serviceMust(t, e)
				if as != 0 || ms != 0 || b.hasPendingReference {
					t.Fatal("ICCP omitted literal coefficient child", prefix, body, as, ms)
				}
			}
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, content.value, map[[3]uint32]int64{{0, 1, 0}: 1, {}: 1})
				polynomialQueryExpect(t, c, primitive.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
				if i != NewInt(1) {
					t.Fatal("literal integer content", i)
				}
			}
		})
	}
	if !success || !firstCI {
		t.Fatal("first Coeff after constructed empty ci not reached", success, firstCI)
	}
}

// The supported returning deletion-callback boundary stops actual older DEC
// after successful incoming INC; it is not ordinary mathematical allocation OOM.
func TestPolynomialICCPLaterCIAssignmentRetainsActualOldAndIncomingReferences(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 3, 2, 1, 4, 3, 1, 5, 4}, [][]MonomialPower{{{0, 2}, {1, 2}}, {{0, 2}, {1, 1}}, {{0, 2}}, {{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{0, 1}}, {{1, 2}}, {{1, 1}}, nil})
	var i Int
	var content, primitive polynomialReference
	var child polynomialICCPInvocationReceipt
	var oldCI, incoming Polynomial
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.budget, func(_ *PolynomialDeletionHandler, owner *MultivariatePolynomialCore, p Polynomial) error {
		if full.iccpDepth != 1 || len(full.iccpFrames) == 0 {
			return nil
		}
		s := full.iccpFrames[0]
		if s.ci.assigning && s.ci.value == p && s.ci.incoming != (Polynomial{}) && calls == 0 {
			calls++
			oldCI = s.ci.value
			incoming = s.ci.incoming
			return ErrUPolynomialCanceled
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	t.Cleanup(func() {
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		serviceMust(t, full.CleanupContentPrimitiveParts())
		serviceMust(t, content.close(c))
		serviceMust(t, primitive.close(c))
		serviceMust(t, c.integer.Del(&i))
	})
	body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &child)
	if !errors.Is(body, ErrUPolynomialCanceled) || calls != 1 || child.frame == nil {
		t.Fatal("actual later ci assignment stop absent", body, calls)
	}
	s := child.frame
	copiedFrame := *s
	copiedReceipt := child
	copiedReceipt.frame = &copiedFrame
	if full.matchesICCPInvocation(&copiedReceipt) {
		t.Fatal("copied ICCP source frame acquired original child authority")
	}
	if !s.ciConstructed || !s.ci.assigning || !s.ci.decremented || s.ci.value != oldCI || s.ci.incoming != incoming || s.cursor != 0 {
		t.Fatal("later assignment lost actual old ci/incoming or source cursor", s.ci, s.cursor, oldCI, incoming)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*Int{&i}, h.header)
	state, used := c.memory.State(), c.integer.budget.Used()
	cause := s.cleanupCause
	if full.iccpSourceEnd(s) != cause || state != c.memory.State() || used != c.integer.budget.Used() || s.ci.value != oldCI || s.ci.incoming != incoming {
		t.Fatal("literal source retry changed retained ci assignment")
	}
	serviceMust(t, full.cleanupICCPInvocation(&child, false))
	serviceMust(t, full.cleanupICCPInvocation(&child, true))
	if s.cleanupCause != cause || full.iccpInvocationSourceCause(&child) != cause {
		t.Fatal("own acknowledgment erased first ICCP source cause", cause, s.cleanupCause, full.iccpInvocationSourceCause(&child))
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*Int{&i}, h.header)
	var later polynomialICCPInvocationReceipt
	serviceMust(t, full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &later))
	if later.frame != child.frame || later.invocation == child.invocation || full.hasICCPInvocationHistory(&child) || !errors.Is(body, ErrUPolynomialCanceled) || cause != ErrUPolynomialCanceled {
		t.Fatal("new actual invocation retargeted old ICCP receipt or returned error", body, cause, child.invocation, later.invocation)
	}
}

// This reaches the literal Neg setter capacity guard after a BIG-to-larger-BIG
// replacement refusal. It is a guarded source-pointer profile, not mathematical
// OOM. The younger Neg source stop must precede the already constructed ci DT.
func TestPolynomialICCPNegSourceStopProtectsConstructedCI(t *testing.T) {
	success, stopped := false, false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			var first, large, i Int
			short, long := make([]uint32, 4096), make([]uint32, 8192)
			short[len(short)-1] = 1
			long[0], long[len(long)-1] = 1, 1
			serviceMust(t, c.integer.SetLimbs(&first, short))
			serviceMust(t, c.integer.SetLimbs(&large, long))
			serviceMust(t, c.integer.Neg(&first))
			serviceMust(t, c.integer.Neg(&large))
			xy, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}, {1, 1}})
			serviceMust(t, e)
			x, e := c.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			p, e := c.MkTerms([]Int{first, large}, []Monomial{xy, x})
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			var content, primitive polynomialReference
			var child polynomialICCPInvocationReceipt
			t.Cleanup(func() {
				c.integer.budget.FailAfter(-1)
				serviceMust(t, full.CleanupContentPrimitiveParts())
				serviceMust(t, content.close(c))
				serviceMust(t, primitive.close(c))
				serviceMust(t, c.integer.Del(&first))
				serviceMust(t, c.integer.Del(&large))
				serviceMust(t, c.integer.Del(&i))
			})
			c.integer.budget.FailAfter(prefix)
			body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &child)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*Int{&first, &large, &i}, 0)
			var cause error
			if child.frame != nil {
				cause = c.negInvocationSourceCause(&child.frame.neg)
			}
			if errors.Is(cause, arena.ErrInvalidSlot) {
				stopped = true
				if !errors.Is(body, cause) || !child.frame.ciConstructed || child.frame.cursor != 0 || full.iccpInvocationSourceCause(&child) != cause {
					t.Fatal("younger Neg stop did not protect actual older ci", prefix, body, cause, child.frame.cursor)
				}
				used, state := c.integer.budget.Used(), c.memory.State()
				if full.iccpSourceEnd(child.frame) != cause || used != c.integer.budget.Used() || state != c.memory.State() {
					t.Fatal("literal source retry advanced after Neg stop")
				}
				t.Log("actual Neg source DT guard", prefix)
			}
			c.integer.budget.FailAfter(-1)
			serviceMust(t, full.cleanupICCPInvocation(&child, false))
			serviceMust(t, full.cleanupICCPInvocation(&child, true))
			serviceMust(t, full.CleanupContentPrimitiveParts())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, content.incoming, primitive.value, primitive.incoming}, []*Int{&first, &large, &i}, 0)
			if cause != nil && full.iccpInvocationSourceCause(&child) != cause {
				t.Fatal("own ACK erased ICCP Neg source history")
			}
			if body == nil {
				success = true
				t.Log("actual first success", prefix)
			}
		})
	}
	if !success || !stopped {
		t.Fatal("actual Neg source-stop or first-success horizon absent", stopped, success)
	}
}
