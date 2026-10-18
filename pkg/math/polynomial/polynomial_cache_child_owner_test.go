package polynomial

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialCacheRefusedFactorPreservesOlderInvocation(t *testing.T) {
	reached := false
	for after := int64(0); after < 128 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			cache, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, cache.Close()) })
			out, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			older, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.CleanupFactor()); serviceMust(t, older.CloseChecked()) })
			p := polynomialQueryTerms(t, m.core, []int32{1, 2, 1}, [][]MonomialPower{nil, {{0, 1}}, {{0, 2}}})
			constant := polynomialQueryTerms(t, m.core, []int32{3}, [][]MonomialPower{nil})
			budget := m.core.integer.Budget()
			budget.FailAfter(after)
			body := m.Factor(p, older, DefaultUPolynomialFactorParams())
			budget.FailAfter(-1)
			receipt := m.factorRoot
			if body == nil || !m.matchesFactorCore(&receipt) || !receipt.frame.active {
				return
			}
			reached = true
			before := *receipt.frame
			body = cache.Factor(p, out)
			if !errors.Is(body, ErrPolynomialPending) {
				t.Error("cache did not refuse pending factor", body)
			}
			if !reflect.DeepEqual(before, *receipt.frame) || !receipt.frame.active {
				t.Error("cache adopted independent factor invocation")
			}
			if cache.factorCall != (polynomialFactorCoreInvocationReceipt{}) {
				t.Error("refused factor call stamped a receipt")
			}
			serviceMust(t, cache.CleanupFactor())
			if !reflect.DeepEqual(before, *receipt.frame) || !receipt.frame.active {
				t.Error("cache cleanup adopted independent factor invocation")
			}
			serviceMust(t, cache.Factor(constant, out))
			if !reflect.DeepEqual(before, *receipt.frame) || !receipt.frame.active {
				t.Error("independent constant cache call consumed older factor")
			}
			serviceMust(t, cache.Reset())
			serviceMust(t, cache.Close())
			if !reflect.DeepEqual(before, *receipt.frame) || !receipt.frame.active {
				t.Error("cache reset/close consumed independent factor invocation")
			}
		})
	}
	if !reached {
		t.Fatal("older actual factor refusal not reached")
	}
}

func TestPolynomialCacheRefusedPSCPreservesOlderInvocation(t *testing.T) {
	reached := false
	for after := int64(0); after < 256 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			cache, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, cache.Close()) })
			out, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			older, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.core.CleanupPSC()); serviceMust(t, older.Close()) })
			p := polynomialQueryTerms(t, m.core, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, m.core, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			budget := m.core.integer.Budget()
			budget.FailAfter(after)
			body := m.PSCChain(p, q, 1, older)
			budget.FailAfter(-1)
			frame := &m.core.pscLocal
			if body == nil || !frame.active {
				return
			}
			reached = true
			before := *frame
			body = cache.PSCChain(p, q, 1, out)
			if !errors.Is(body, ErrPolynomialPending) {
				t.Error("cache did not refuse pending PSC", body)
			}
			if !reflect.DeepEqual(before, *frame) || !frame.active {
				t.Error("cache adopted independent PSC invocation")
			}
			if cache.pscCall != (polynomialPSCInvocationReceipt{}) {
				t.Error("refused PSC call stamped a receipt")
			}
			serviceMust(t, cache.CleanupPSC())
			if !reflect.DeepEqual(before, *frame) || !frame.active {
				t.Error("cache cleanup adopted independent PSC invocation")
			}
			serviceMust(t, cache.Reset())
			serviceMust(t, cache.Close())
			if !reflect.DeepEqual(before, *frame) || !frame.active {
				t.Error("cache reset/close consumed independent PSC invocation")
			}
		})
	}
	if !reached {
		t.Fatal("older actual PSC refusal not reached")
	}
}
