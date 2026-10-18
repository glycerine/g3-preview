package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestPolynomialCacheBorrowedManagerSupportsRetainedReferenceConsumer(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	budget := m.core.integer.Budget()
	used := budget.Used()
	budget.FailAfter(0)
	manager, e := cache.Manager()
	serviceMust(t, e)
	pm, e := cache.PM()
	serviceMust(t, e)
	budget.FailAfter(-1)
	if manager != m || pm != m || budget.Used() != used {
		t.Fatal("accessor did not borrow the actual manager")
	}
	core, e := pm.ArithmeticCore()
	serviceMust(t, e)
	refs, e := NewPolynomialRefVector(core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, refs.Close()) })
	p := polynomialQueryTerms(t, core, []int32{1, 1}, [][]MonomialPower{nil, {{0, 1}}})
	duplicate := polynomialQueryTerms(t, core, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	representative, e := cache.MkUnique(p)
	serviceMust(t, e)
	serviceMust(t, refs.Push(representative))
	representative, e = cache.MkUnique(duplicate)
	serviceMust(t, e)
	if representative != p {
		t.Fatal("retained consumer lost canonical identity")
	}
	info, e := core.Info(p)
	serviceMust(t, e)
	if info.RefCount != 3 {
		t.Fatal("getter acquired ownership or reference consumer lost its own", info)
	}
	serviceMust(t, cache.Reset())
	again, e := cache.PM()
	serviceMust(t, e)
	if again != pm {
		t.Fatal("private imp replacement changed borrowed manager")
	}
	value, e := refs.Get(0)
	serviceMust(t, e)
	if value != p {
		t.Fatal("cache reset invalidated caller-owned reference")
	}
	info, e = core.Info(p)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("reset changed independent reference owner", info)
	}
	representative, e = cache.MkUnique(duplicate)
	serviceMust(t, e)
	if representative != duplicate {
		t.Fatal("reset retained former representative")
	}
}

func TestPolynomialCacheManagerAccessorOwnerAndResetLifetime(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	var absent *PolynomialCache
	copy := *cache
	for _, c := range []*PolynomialCache{absent, &copy} {
		value, e := c.Manager()
		if value != nil || !errors.Is(e, ErrPolynomialOwner) {
			t.Fatal("invalid cache borrowed manager", value, e)
		}
		value, e = c.PM()
		if value != nil || !errors.Is(e, ErrPolynomialOwner) {
			t.Fatal("invalid alias borrowed manager", value, e)
		}
	}
	budget := m.core.integer.Budget()
	budget.FailAfter(0)
	body := cache.Reset()
	budget.FailAfter(-1)
	if body == nil || cache.imp != nil || !cache.resetting {
		t.Fatal("actual replacement refusal not reached", body)
	}
	value, e := cache.Manager()
	if value != nil || !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("absent source imp exposed manager", value, e)
	}
	value, e = cache.PM()
	if value != nil || !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("alias crossed absent source imp", value, e)
	}
	serviceMust(t, cache.CleanupReset())
	value, e = cache.Manager()
	serviceMust(t, e)
	if value != m {
		t.Fatal("restored source imp lost actual manager")
	}
	serviceMust(t, cache.Close())
	value, e = cache.Manager()
	if value != nil || !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed cache borrowed manager", value, e)
	}
	value, e = cache.PM()
	if value != nil || !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed alias borrowed manager", value, e)
	}
}
