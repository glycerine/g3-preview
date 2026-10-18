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
