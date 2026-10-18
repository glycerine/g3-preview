package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialCacheCallbackProtectsBorrowedCoreSourceLifetime(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	input := polynomialQueryTerms(t, m.core, []int32{3}, [][]MonomialPower{nil})
	n := unumeral.NewInt(17)
	orphan, e := m.core.MkConstantTake(&n)
	serviceMust(t, e)
	serviceMust(t, out.Push(orphan))
	reached := false
	handler, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
		if p != orphan {
			return nil
		}
		reached = true
		before := m.core.sourceStage
		for _, e := range []error{m.core.SourceFinalizePrefix(), m.core.CloseChecked(), m.SourceFinalize(), m.CloseChecked()} {
			if !errors.Is(e, ErrPolynomialPending) {
				t.Error("cache callback acquired borrowed polynomial destructor", e)
			}
		}
		if before != m.core.sourceStage || m.core.sourceStarted {
			t.Error("borrowed core destructor changed source prefix")
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	body := cache.Factor(input, out)
	if !reached || body != nil {
		t.Error("cache did not retain its borrowed core", reached, body)
	}
}

func TestPolynomialCacheNestedCallsRestoreBorrowedCoreBinding(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	outer, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, outer.Close()) })
	inner, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, inner.Close()) })
	var values [2]Polynomial
	for i, cache := range []*PolynomialCache{outer, inner} {
		n := unumeral.NewInt(int32(17 + i))
		values[i], e = m.core.MkConstantTake(&n)
		serviceMust(t, e)
		_, e = cache.MkUnique(values[i])
		serviceMust(t, e)
	}
	outerReached, innerReached := false, false
	handler, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
		switch p {
		case values[0]:
			outerReached = true
			if m.core.cacheCaller != outer || !outer.executing {
				t.Error("outer actual borrow missing")
			}
			serviceMust(t, inner.Reset())
			if m.core.cacheCaller != outer || inner.previous != nil || inner.executing || !outer.executing {
				t.Error("nested return did not restore outer borrow")
			}
			if e := m.SourceFinalize(); !errors.Is(e, ErrPolynomialPending) {
				t.Error("outer borrow protection lost after nested return", e)
			}
		case values[1]:
			innerReached = true
			if m.core.cacheCaller != inner || inner.previous != outer || !inner.executing || !outer.executing {
				t.Error("nested actual borrow chain missing")
			}
			for _, e := range []error{outer.Close(), inner.Close(), m.SourceFinalize(), m.CloseChecked(), m.core.SourceFinalizePrefix(), m.core.CloseChecked()} {
				if !errors.Is(e, ErrPolynomialPending) {
					t.Error("nested callback acquired borrowed owner", e)
				}
			}
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	serviceMust(t, outer.Reset())
	if !outerReached || !innerReached || m.core.cacheCaller != nil || m.core.cacheExecuting() || outer.previous != nil || inner.previous != nil || outer.executing || inner.executing {
		t.Fatal("nested cache lifecycle incomplete", outerReached, innerReached)
	}
}

func TestPolynomialCacheStoppedResetRetainsBorrowedManager(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	budget := m.core.integer.Budget()
	budget.FailAfter(0)
	body := cache.Reset()
	budget.FailAfter(-1)
	if body == nil || !cache.resetting || cache.imp != nil {
		t.Fatal("actual reset replacement refusal not reached", body)
	}
	stage, source, started := m.bodyStage, m.core.sourceStage, m.sourceStarted
	for _, e := range []error{m.SourceFinalize(), m.CloseChecked(), m.core.SourceFinalizePrefix(), m.core.CloseChecked()} {
		if !errors.Is(e, ErrPolynomialPending) {
			t.Error("stopped cache lost its borrowed manager", e)
		}
	}
	if stage != m.bodyStage || source != m.core.sourceStage || started != m.sourceStarted {
		t.Error("manager crossed pending cache before cleanup")
	}
	serviceMust(t, cache.CleanupReset())
	if cache.pendingBound || m.core.pendingCaches != nil {
		t.Fatal("completed reset retained manager borrow")
	}
	value, e := cache.Manager()
	serviceMust(t, e)
	if value != m {
		t.Fatal("cleanup did not restore normal manager borrow")
	}
}

func TestPolynomialCacheMultiplePendingBorrowsReleaseIndependently(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	first, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, first.Close()) })
	second, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, second.Close()) })
	for _, cache := range []*PolynomialCache{first, second} {
		m.originalBudget.FailAfter(0)
		body := cache.Reset()
		m.originalBudget.FailAfter(-1)
		if body == nil || !cache.pendingBound || !cache.resetting {
			t.Fatal("actual pending cache borrow not retained", body)
		}
	}
	if m.core.pendingCaches != second || second.nextPending != first {
		t.Fatal("actual pending caller chain incomplete")
	}
	serviceMust(t, first.CleanupReset())
	if first.pendingBound || m.core.pendingCaches != second || second.nextPending != nil {
		t.Fatal("non-head cleanup changed independent pending owner")
	}
	if e := m.SourceFinalize(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("remaining stopped cache lost manager", e)
	}
	serviceMust(t, second.Close())
	if m.core.pendingCaches != nil || second.pendingBound || first.pendingBound {
		t.Fatal("finished cache cleanup retained pending borrows")
	}
	if m.sourceStarted || m.core.sourceStarted {
		t.Fatal("pending cache checks advanced manager source destruction")
	}
}

func TestPolynomialCacheStoppedDeletionRetainsBorrowedManager(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	n := unumeral.NewInt(17)
	p, e := m.core.MkConstantTake(&n)
	serviceMust(t, e)
	_, e = cache.MkUnique(p)
	serviceMust(t, e)
	stop := errors.New("stop actual cache reset deletion")
	armed := true
	handler, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, value Polynomial) error {
		if armed && value == p {
			return stop
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	body := cache.Reset()
	armed = false
	if !errors.Is(body, stop) || !cache.pendingBound {
		t.Fatal("actual deletion did not retain borrow", body)
	}
	for _, e := range []error{m.SourceFinalize(), m.CloseChecked(), m.core.SourceFinalizePrefix(), m.core.CloseChecked()} {
		if !errors.Is(e, stop) {
			t.Fatal("borrowed destructor lost deletion stop", e)
		}
	}
	serviceMust(t, cache.CleanupReset())
	if cache.pendingBound || m.core.pendingCaches != nil {
		t.Fatal("completed cleanup retained borrow")
	}
}
