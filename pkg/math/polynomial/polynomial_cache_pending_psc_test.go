package polynomial

import (
	"errors"
	"reflect"
	"testing"
)

func TestPolynomialCachePendingPSCOriginalStop(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m.core, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	sourceStop, ownStop := errors.New("stop PSC source release"), errors.New("stop PSC owning release")
	armed, calls := true, 0
	var targets []Polynomial
	handler, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		if !armed || !core.pscLocal.finishing || core.pscLocal.executing {
			return nil
		}
		calls++
		targets = append(targets, value)
		if calls == 1 {
			return sourceStop
		}
		if calls == 2 {
			return ownStop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	body := cache.PSCChain(p, q, 1, out)
	armed = false
	if calls != 2 || !errors.Is(body, sourceStop) || !errors.Is(body, ownStop) || !cache.pendingPSC || !cache.pendingBound {
		t.Fatal("actual PSC stop pair not reached", body, calls, targets, cache.pendingPSC)
	}
	if targets[0] == targets[1] {
		t.Fatal("owning continuation replayed stopped callback")
	}
	for _, e := range []error{m.SourceFinalize(), m.CloseChecked(), m.core.SourceFinalizePrefix(), m.core.CloseChecked()} {
		if !errors.Is(e, sourceStop) {
			t.Fatal("borrowed manager lost source PSC stop", e)
		}
	}
	serviceMust(t, cache.CleanupPSC())
	if cache.pendingBound || cache.pendingPSC || m.core.pendingCaches != nil {
		t.Fatal("PSC acknowledgement retained borrowed manager")
	}
}

func TestPolynomialCachePSCCleanupPreservesIndependentLazard(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	cache, e := NewPolynomialCache(full)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	oldOut, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupLazard()); serviceMust(t, oldOut.Close()) })
	out, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	oldInput := polynomialQueryTerms(t, m, []int32{2, 4}, [][]MonomialPower{{{0, 1}}, nil})
	p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	oldStop, sourceStop := errors.New("independent Lazard stop"), errors.New("cache PSC stop")
	oldReached, sourceReached := false, false
	handler, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, _ Polynomial) error {
		if c.lazardLocal.finishing && !oldReached {
			oldReached = true
			return oldStop
		}
		if c.pscLocal.finishing && !sourceReached {
			sourceReached = true
			return sourceStop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	var old polynomialLazardInvocationReceipt
	body := m.seLazardWithReceipt(3, m.unit, oldInput, 0, &oldOut.root, &old)
	if !oldReached || !errors.Is(body, oldStop) || !old.frame.active {
		t.Fatal("independent actual source stop missing", body)
	}
	before := *old.frame
	body = cache.PSCChain(p, q, 1, out)
	if !sourceReached || !errors.Is(body, sourceStop) || cache.pscCall.invocation == 0 {
		t.Fatal("accepted cache PSC stop missing", body)
	}
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("cache PSC cleanup consumed independent Lazard invocation")
	}
	serviceMust(t, cache.CleanupPSC())
	serviceMust(t, cache.Reset())
	serviceMust(t, cache.Close())
	if !reflect.DeepEqual(before, *old.frame) {
		t.Error("cache continuation changed independent Lazard invocation")
	}
	serviceMust(t, m.cleanupLazardInvocation(&old, true))
}

func TestPolynomialCachePSCExactCleanupReservationPrefixes(t *testing.T) {
	success, stopped := false, false
	for prefix := int64(0); prefix < 1024 && !success; prefix++ {
		full := polynomialPSCFullFixture(t)
		m := full.core
		cache, e := NewPolynomialCache(full)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, cache.Close()) })
		out, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, out.Close()) })
		p := polynomialQueryTerms(t, m, []int32{1, 2, 3, 4}, [][]MonomialPower{{{0, 4}}, {{0, 2}}, {{0, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{4, 4, 3}, [][]MonomialPower{{{0, 3}}, {{0, 1}}, nil})
		full.originalBudget.FailAfter(prefix)
		body := cache.PSCChain(p, q, 0, out)
		full.originalBudget.FailAfter(-1)
		if body != nil {
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected PSC refusal", prefix, body)
			}
			stopped = stopped || cache.pscCall.invocation != 0
		}
		if body == nil {
			success = true
			n, e := out.Size()
			serviceMust(t, e)
			if n != 3 {
				t.Fatal("PSC chain length", prefix, n)
			}
			for i, want := range []int64{17109, 132, 16} {
				v, e := out.Get(uint32(i))
				serviceMust(t, e)
				polynomialQueryExpect(t, m, v, map[[3]uint32]int64{{}: want})
			}
		}
		serviceMust(t, cache.CleanupPSC())
		serviceMust(t, out.Close())
		serviceMust(t, cache.Close())
		polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p, q}, nil, 0, nil)
		if success {
			t.Log("actual first success", prefix)
		}
	}
	if !success || !stopped {
		t.Fatal("PSC reservation horizon missed accepted failure or success", stopped, success)
	}
}
