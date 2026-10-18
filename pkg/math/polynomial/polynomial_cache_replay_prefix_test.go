package polynomial

import (
	"errors"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialCacheHitRetainsCallerPrefixAfterReferenceReadStop(t *testing.T) {
	for _, operation := range []string{"factor", "psc"} {
		t.Run(operation, func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			cache, e := NewPolynomialCache(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, cache.Close()) })
			out, e := NewPolynomialRefVector(m.core)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			var p, q Polynomial
			var replay func() error
			if operation == "factor" {
				p = polynomialQueryTerms(t, m.core, []int32{-1, 1}, [][]MonomialPower{nil, {{0, 2}}})
				replay = func() error { return cache.Factor(p, out) }
			} else {
				p = polynomialQueryTerms(t, m.core, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 3}}, {{0, 1}, {1, 2}}, {{1, 1}}, nil})
				q = polynomialQueryTerms(t, m.core, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 2}}, nil})
				replay = func() error { return cache.PSCChain(p, q, 1, out) }
			}
			serviceMust(t, replay())
			n, e := out.Size()
			serviceMust(t, e)
			if n < 2 {
				t.Fatal("fixture needs multiple replay results", n)
			}
			want := append([]Polynomial(nil), out.state.values...)
			stopAt := uint32(1)
			for stopAt < n && want[stopAt] == want[0] {
				stopAt++
			}
			if stopAt == n {
				t.Fatal("fixture needs a later distinct canonical result")
			}
			serviceMust(t, out.Reset())
			baseline, e := m.core.Info(want[0])
			serviceMust(t, e)
			factorInvocation, pscInvocation := m.factorCoreInvocation, m.core.pscLocal.invocation
			l, e := m.core.polynomialLocation(want[stopAt])
			serviceMust(t, e)
			a := m.core.monomials.allocator
			// Use fresh direct backing: a recycled small-object slot can
			// legitimately retain initialized bytes from its previous user.
			const rawSize = 4097
			raw, e := a.Allocate(rawSize)
			serviceMust(t, e)
			old := l.raw
			for _, offset := range []uint64{4, 8} {
				word, e := a.ReadUint32(old, offset)
				serviceMust(t, e)
				serviceMust(t, a.WriteUint32(raw, offset, word))
			}
			l.raw = raw
			defer func() { l.raw = old; serviceMust(t, a.Deallocate(rawSize, raw)) }()
			if _, e := a.ReadUint32(raw, 0); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatal("query fixture needs unreadable reference-count bytes", e)
			}
			// Borrow a query-only header with an unreadable refcount. A hit
			// publishes each successful INC/push before reaching this guard.
			if e := replay(); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatal("cache hit did not reach reference read guard", e)
			}
			if uint32(len(out.state.values)) != stopAt || cache.hasPendingWork() || m.factorCoreInvocation != factorInvocation || m.core.pscLocal.invocation != pscInvocation {
				t.Fatal("hit failure recomputed, acquired a callee, or lost caller prefix")
			}
			info, e := m.core.Info(want[0])
			serviceMust(t, e)
			if info.RefCount != baseline.RefCount+stopAt {
				t.Fatal("successful replay prefix did not acquire its actual references", info.RefCount, baseline.RefCount, stopAt)
			}
			l.raw = old
			serviceMust(t, replay())
			if uint32(len(out.state.values)) != n {
				t.Fatal("retry did not reset prefix and replay complete entry")
			}
			for i, value := range want {
				if out.state.values[i] != value {
					t.Fatal("retry changed cached result identity", i)
				}
			}
			if m.factorCoreInvocation != factorInvocation || m.core.pscLocal.invocation != pscInvocation {
				t.Fatal("retry recomputed a populated cache entry")
			}
		})
	}
}
