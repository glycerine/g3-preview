package polynomial

// Source: polynomial.cpp:4018-4051, actual leading-coefficient buffer calls.

import (
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialGCDGlexZPXCapturedBufferReservationPrefixes(t *testing.T) {
	success, captured, monomialCaptured := false, false, false
	for prefix := int64(0); prefix < 128 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		c := full.core
		serviceMust(t, c.modular.SetZPUint64(11))
		p := polynomialQueryTerms(t, c, []int32{3, 4, 5, 1}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 2}}, {{0, 3}, {1, 1}}, nil})
		c.integer.Budget().FailAfter(prefix)
		result, body := full.lcGlexZPX(p, 0)
		c.integer.Budget().FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("unexpected source refusal", prefix, body)
		}
		if len(full.gcdFrames) == 0 {
			if body == nil {
				t.Fatal("successful ZPX call without its actual invocation")
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
			continue
		}
		s := full.gcdFrames[0]
		if body != nil && s.zpxCalleeGeneration != 0 {
			monomialCaptured = true
			if c.monomials.creation.generation != s.zpxCalleeGeneration || c.monomials.creation.parent != c {
				t.Fatal("ZPX lost actual constructor binding", prefix)
			}
			serviceMust(t, full.cleanupGCDZPXMonomial(s))
			if full.gcdZPXMonomialPending(s) {
				t.Fatal("exact monomial acknowledgement retained work", prefix)
			}
		}
		r := full.gcdFrames[0].product
		if body != nil && c.matchesArithmeticInvocation(&r) {
			captured = true
			serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
			serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
			n, e := c.som.ms.size(c)
			serviceMust(t, e)
			if n != 0 || c.som.hasPendingReference {
				t.Fatal("exact buffer receipt left source rows", prefix, n)
			}
		}
		serviceMust(t, full.CleanupGCD())
		if body == nil {
			success = true
			result = polynomialCaller(t, c, result)
			polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 4})
			t.Log("actual first success", prefix)
		} else if result != (Polynomial{}) {
			serviceMust(t, c.CleanupValue(&result))
		}
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, nil, 0)
	}
	if !success || !captured || !monomialCaptured {
		t.Fatal("missed successful result or retained actual buffer receipt", success, captured, monomialCaptured)
	}
}

func TestPolynomialGCDGlexZPXReturnedCallerCallbackComposition(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	serviceMust(t, c.modular.SetZPUint64(11))
	p := polynomialQueryTerms(t, c, []int32{3, 4, 5, 1}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{1, 2}}, {{0, 3}, {1, 1}}, nil})
	old := polynomialICCPUnownedTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{0, 1}}, nil})
	var destination polynomialReference
	serviceMust(t, destination.assign(c, old))
	var refused umemory.Block
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, deleted Polynomial) error {
		if deleted != old {
			return nil
		}
		calls++
		if len(full.gcdFrames) == 0 || full.gcdFrames[0].active || full.gcdFrames[0].depth != 0 {
			t.Errorf("lc_glex_ZpX locals must finish before caller old-DEC: %+v", full.gcdFrames)
		}
		if !destination.assigning || destination.incoming == (Polynomial{}) {
			t.Errorf("caller did not own returned INC before old-DEC: %+v", destination)
		}
		c.integer.Budget().FailAfter(0)
		var callback error
		refused, callback = c.memory.Allocate(8)
		return callback
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	e = full.lcGlexZPXInto(p, 0, &destination)
	if !numericPhysicalReservationRefusal(e) || calls != 1 || !destination.assigning {
		t.Fatal("actual returned assignment callback stop", e, calls, destination)
	}
	polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, destination.value, destination.incoming}, nil, h.header, []umemory.Block{refused})
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, full.CleanupGCD())
	serviceMust(t, destination.close(c))
	if calls != 1 {
		t.Fatal("callback replay", calls)
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}
