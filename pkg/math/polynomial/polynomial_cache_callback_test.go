package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestPolynomialCacheFactorOutputResetCallbackCannotCloseCache(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	out, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	input := polynomialQueryTerms(t, m.core, []int32{3}, [][]MonomialPower{nil})
	value := unumeral.NewInt(17)
	orphan, e := m.core.MkConstantTake(&value)
	serviceMust(t, e)
	serviceMust(t, out.Push(orphan))
	info, e := m.core.Info(orphan)
	serviceMust(t, e)
	if info.RefCount != 1 {
		t.Fatal("output is not sole actual reference", info)
	}
	reached := false
	handler, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
		if p != orphan {
			return nil
		}
		reached = true
		polynomialCacheCallbackProbe(t, cache, input)
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(handler))
	t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
	body := cache.Factor(input, out)
	if !reached || body != nil || cache.closed {
		t.Error("cache did not survive actual deletion callback", reached, body, cache.closed)
	}
}

func polynomialCacheCallbackProbe(t *testing.T, cache *PolynomialCache, p Polynomial) {
	t.Helper()
	before := *cache
	refs := append([]Polynomial(nil), cache.refs.values...)
	counts := [4]int{len(cache.buckets), len(cache.ids), len(cache.psc), len(cache.factors)}
	used := cache.manager.originalBudget.Used()
	for _, e := range []error{cache.Factor(p, nil), cache.PSCChain(p, p, 0, nil), cache.Reset(), cache.CleanupReset(), cache.CleanupFactor(), cache.CleanupPSC(), cache.Close()} {
		if !errors.Is(e, ErrPolynomialPending) {
			t.Error("callback acquired executing cache", e)
		}
	}
	for _, get := range []func() (*MultivariatePolynomialManager, error){cache.Manager, cache.PM} {
		value, e := get()
		if value != nil || !errors.Is(e, ErrPolynomialPending) {
			t.Error("callback borrowed executing cache manager", value, e)
		}
	}
	value, e := cache.MkUnique(p)
	if !errors.Is(e, ErrPolynomialPending) || value != (Polynomial{}) {
		t.Error("callback acquired canonicalization", value, e)
	}
	if !reflect.DeepEqual(before, *cache) || !reflect.DeepEqual(refs, cache.refs.values) || counts != ([4]int{len(cache.buckets), len(cache.ids), len(cache.psc), len(cache.factors)}) || used != cache.manager.originalBudget.Used() {
		t.Error("refused callback changed cache ownership/publication")
	}
}

func TestPolynomialCacheResetAndCloseCallbacksRetainExecutingOwner(t *testing.T) {
	for _, mode := range []string{"reset", "cleanup-reset", "close"} {
		t.Run(mode, func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			cache, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, cache.Close()) })
			other, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, other.Close()) })
			input := polynomialQueryTerms(t, m.core, []int32{3}, [][]MonomialPower{nil})
			var owned [2]Polynomial
			for i := range owned {
				n := unumeral.NewInt(int32(17 + i))
				owned[i], e = m.core.MkConstantTake(&n)
				serviceMust(t, e)
				_, e = cache.MkUnique(owned[i])
				serviceMust(t, e)
				info, e := m.core.Info(owned[i])
				serviceMust(t, e)
				if info.RefCount != 1 {
					t.Fatal("cache is not sole actual owner", info)
				}
			}
			calls := 0
			continuing, resumed := false, false
			stop := errors.New("stop cache canonical reference deletion")
			handler, e := NewPolynomialDeletionHandler(m.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
				if p != owned[0] && p != owned[1] {
					return nil
				}
				calls++
				polynomialCacheCallbackProbe(t, cache, input)
				serviceMust(t, other.CleanupPSC())
				if continuing {
					resumed = true
				}
				if mode == "cleanup-reset" && calls == 1 {
					return stop
				}
				return nil
			})
			serviceMust(t, e)
			serviceMust(t, m.core.AddDeletionHandler(handler))
			t.Cleanup(func() { serviceMust(t, m.core.RemoveDeletionHandler(handler)); serviceMust(t, handler.CloseChecked()) })
			if mode == "close" {
				serviceMust(t, cache.Close())
			} else {
				body := cache.Reset()
				if mode == "cleanup-reset" {
					if !errors.Is(body, stop) || !cache.resetting || cache.executing {
						t.Fatal("actual stopped reset lost its owner", body)
					}
					continuing = true
					serviceMust(t, cache.CleanupReset())
					if !resumed {
						t.Fatal("owning reset did not reach the later deletion callback")
					}
				} else {
					serviceMust(t, body)
				}
			}
			if calls != 2 || cache.executing || cache.resetting {
				t.Fatal("cache callback lifecycle incomplete", calls, cache.executing, cache.resetting)
			}
		})
	}
}
